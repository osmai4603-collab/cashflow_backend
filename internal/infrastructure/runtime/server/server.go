package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"cashflow_backend/internal/infrastructure/runtime/health"
	"cashflow_backend/internal/infrastructure/runtime/worker"
	platconfig "cashflow_backend/internal/platform/config"
)

// Server coordinates the complete HTTP server lifecycle across all 7 phases:
// Phase 1: Initialization
// Phase 2: Configuration
// Phase 3: Startup (non-blocking)
// Phase 4: Serving (health checks)
// Phase 5: Drain (mark not-ready, wait for LB)
// Phase 6: Graceful Shutdown
// Phase 7: Cleanup (reverse order)
type Server struct {
	httpServer         *http.Server
	managementServer   *http.Server
	managementHandler  http.Handler
	managementListener atomic.Pointer[net.Listener]
	cfg                *platconfig.Configuration
	logger             *slog.Logger
	health             *health.HealthChecker
	workerMgr          *worker.WorkerManager
	resources          []io.Closer
	listener           atomic.Pointer[net.Listener]
}

// NewServer configures a new Server instance.
// Critical: Always sets all 4 network timeouts (Read, ReadHeader, Write, Idle).
func NewServer(
	cfg *platconfig.Configuration,
	handler http.Handler,
	healthChecker *health.HealthChecker,
	workerMgr *worker.WorkerManager,
	logger *slog.Logger,
	resources ...io.Closer,
) *Server {
	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           handler,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}

	return &Server{
		httpServer: srv,
		cfg:        cfg,
		logger:     logger,
		health:     healthChecker,
		workerMgr:  workerMgr,
		resources:  resources,
	}
}

// SetListener allows injecting a custom net.Listener (useful for testing dynamic ports).
func (s *Server) SetListener(l net.Listener) {
	s.listener.Store(&l)
	s.httpServer.Addr = l.Addr().String()
}

// SetManagementHandler wires the isolated management listener (metrics, index,
// probes, optional pprof). It mirrors the public server timeouts. Setting a nil
// handler disables the management listener entirely.
func (s *Server) SetManagementHandler(h http.Handler) {
	if h == nil {
		s.managementServer = nil
		s.managementHandler = nil
		return
	}
	s.managementHandler = h
	s.managementServer = &http.Server{
		Addr:              s.cfg.ManagementAddr(),
		Handler:           h,
		ReadTimeout:       s.cfg.Server.ReadTimeout,
		ReadHeaderTimeout: s.cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      s.cfg.Server.WriteTimeout,
		IdleTimeout:       s.cfg.Server.IdleTimeout,
		MaxHeaderBytes:    s.cfg.Server.MaxHeaderBytes,
	}
}

// SetManagementListener allows injecting a custom listener for the management
// server (useful for testing dynamic ports).
func (s *Server) SetManagementListener(l net.Listener) {
	s.managementListener.Store(&l)
	if s.managementServer != nil {
		s.managementServer.Addr = l.Addr().String()
	}
}

// Addr returns the configured address of the server.
func (s *Server) Addr() string {
	if lp := s.listener.Load(); lp != nil {
		return (*lp).Addr().String()
	}
	return s.httpServer.Addr
}

// ManagementAddr returns the configured or bound address of the management
// listener, or "" when it is disabled.
func (s *Server) ManagementAddr() string {
	if lp := s.managementListener.Load(); lp != nil {
		return (*lp).Addr().String()
	}
	if s.managementServer != nil {
		return s.managementServer.Addr
	}
	return ""
}

// Run executes the server through its complete lifecycle until ctx is canceled
// or a fatal error occurs.
//
// P5 lifecycle contract:
//   - Both listeners are bound synchronously on the caller goroutine (fail-fast)
//     BEFORE readiness is announced. A bind failure on either listener returns a
//     fatal error and readiness is never reached (gap 3.3).
//   - A post-start failure on either server still drains and gracefully shuts
//     down BOTH servers (management is never left serving after the public
//     server fails or receives the signal).
//   - A single shutdown path avoids double-close races, and Run waits for both
//     Serve goroutines to exit so no goroutines are leaked.
func (s *Server) Run(ctx context.Context) error {
	startedAt := time.Now()

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 3: Startup
	// ─────────────────────────────────────────────────────────────────────
	// 3a. Pre-bind listeners on the main goroutine. Readiness below is
	//     announced only after BOTH binds have actually succeeded.
	var publicLn net.Listener
	if ptr := s.listener.Load(); ptr != nil {
		publicLn = *ptr
	} else {
		ln, err := net.Listen("tcp", s.cfg.HTTPAddr())
		if err != nil {
			return fmt.Errorf("bind public %s: %w", s.cfg.HTTPAddr(), err)
		}
		publicLn = ln
		s.listener.Store(&ln)
	}
	s.httpServer.Addr = publicLn.Addr().String()

	var mgmtLn net.Listener
	if s.managementServer != nil {
		if ptr := s.managementListener.Load(); ptr != nil {
			mgmtLn = *ptr
		} else {
			ln, err := net.Listen("tcp", s.cfg.ManagementAddr())
			if err != nil {
				_ = publicLn.Close()
				return fmt.Errorf("bind management %s: %w", s.cfg.ManagementAddr(), err)
			}
			mgmtLn = ln
			s.managementListener.Store(&ln)
		}
		s.managementServer.Addr = mgmtLn.Addr().String()
	}

	// 3b. Serve goroutines over the pre-bound listeners. Both servers stay
	//     inside a single WaitGroup so Run can wait for clean teardown.
	serverErr := make(chan error, 2)
	var serveWG sync.WaitGroup

	serveWG.Add(1)
	go func() {
		defer serveWG.Done()
		s.logger.Info("serving", "server_role", "public", "addr", publicLn.Addr().String())
		if err := s.httpServer.Serve(publicLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.pushError(serverErr, err)
		}
	}()

	if s.managementServer != nil {
		serveWG.Add(1)
		go func() {
			defer serveWG.Done()
			s.logger.Info("serving", "server_role", "management", "addr", mgmtLn.Addr().String())
			if err := s.managementServer.Serve(mgmtLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.pushError(serverErr, err)
			}
		}()
	}

	s.health.MarkReady()
	s.logger.Info("server is ready to accept traffic")

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 4: Serving (Wait for shutdown signal or fatal error)
	// ─────────────────────────────────────────────────────────────────────
	shutdownReason := "signal"
	select {
	case err := <-serverErr:
		s.logger.Error("server failure", "error", err)
		shutdownReason = "server_failure"
	case <-ctx.Done():
		s.logger.Info("shutdown trigger received")
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 5: Drain Phase
	// ─────────────────────────────────────────────────────────────────────
	// 5a. Mark not ready so load balancers stop sending new requests.
	s.health.MarkNotReady()
	s.logger.Info("drain phase: server marked as not ready",
		"drain_duration", s.cfg.Server.DrainDuration,
		"shutdown_reason", shutdownReason,
	)

	// 5b. Wait for load balancer to update routing tables.
	drainStarted := time.Now()
	if s.cfg.Server.DrainDuration > 0 {
		time.Sleep(s.cfg.Server.DrainDuration)
	}
	s.logger.Info("drain phase complete", "duration", time.Since(drainStarted).Round(time.Millisecond))

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 6: Graceful Shutdown (both servers, unified timeout)
	// ─────────────────────────────────────────────────────────────────────
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	shutdownStarted := time.Now()
	s.logger.Info("graceful shutdown starting",
		"timeout", s.cfg.Server.ShutdownTimeout,
		"shutdown_reason", shutdownReason,
	)

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("graceful shutdown failed — forcing server close", "server_role", "public", "error", err)
		_ = s.httpServer.Close()
	} else {
		s.logger.Info("graceful shutdown completed", "server_role", "public", "duration", time.Since(shutdownStarted).Round(time.Millisecond))
	}

	if s.managementServer != nil {
		if err := s.managementServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("graceful shutdown failed — forcing server close", "server_role", "management", "error", err)
			_ = s.managementServer.Close()
		} else {
			s.logger.Info("graceful shutdown completed", "server_role", "management", "duration", time.Since(shutdownStarted).Round(time.Millisecond))
		}
	}
	shutdownCancel()

	// Wait for both Serve goroutines to exit (no leaked goroutines).
	serveWG.Wait()
	s.logger.Info("http servers fully stopped")

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 7: Cleanup (Reverse order of creation)
	// ─────────────────────────────────────────────────────────────────────
	// 7a. Stop background workers.
	if s.workerMgr != nil {
		s.workerMgr.StopAll()
	}

	// 7b. Close other dependencies (databases, storage, etc.)
	for i := len(s.resources) - 1; i >= 0; i-- {
		res := s.resources[i]
		if res != nil {
			if err := res.Close(); err != nil {
				s.logger.Error("failed to close resource", "error", err)
			}
		}
	}

	s.logger.Info("server exited cleanly",
		"shutdown_reason", shutdownReason,
		"uptime_seconds", time.Since(startedAt).Seconds(),
	)
	return nil
}

// pushError safely forwards a serving error to the lifecycle channel.
func (s *Server) pushError(serverErr chan<- error, err error) {
	select {
	case serverErr <- err:
		s.logger.Error("server returned fatal error", "error", err)
	default:
	}
}
