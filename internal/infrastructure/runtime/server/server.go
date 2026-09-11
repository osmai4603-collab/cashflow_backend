package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
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
	httpServer *http.Server
	cfg        *platconfig.Configuration
	logger     *slog.Logger
	health     *health.HealthChecker
	workerMgr  *worker.WorkerManager
	resources  []io.Closer
	listener   net.Listener
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
	s.listener = l
	s.httpServer.Addr = l.Addr().String()
}

// Addr returns the configured address of the server.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.httpServer.Addr
}

// Run executes the server through its complete lifecycle until ctx is canceled or a fatal error occurs.
func (s *Server) Run(ctx context.Context) error {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 3: Startup (Non-blocking)
	// ─────────────────────────────────────────────────────────────────────
	serverErr := make(chan error, 1)

	go func() {
		s.logger.Info("starting http server", "addr", s.Addr())
		var err error
		if s.listener != nil {
			err = s.httpServer.Serve(s.listener)
		} else {
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Mark ready to accept traffic
	s.health.MarkReady()
	s.logger.Info("server is ready to accept traffic")

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 4: Serving (Wait for shutdown signal or fatal error)
	// ─────────────────────────────────────────────────────────────────────
	select {
	case err := <-serverErr:
		s.logger.Error("server startup failed", "error", err)
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutdown trigger received")
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 5: Drain Phase
	// ─────────────────────────────────────────────────────────────────────
	// 5a. Mark not ready so load balancers stop sending new requests
	s.health.MarkNotReady()
	s.logger.Info("drain phase: server marked as not ready", "drain_duration", s.cfg.Server.DrainDuration)

	// 5b. Wait for load balancer to update routing tables
	if s.cfg.Server.DrainDuration > 0 {
		time.Sleep(s.cfg.Server.DrainDuration)
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 6: Graceful Shutdown
	// ─────────────────────────────────────────────────────────────────────
	s.logger.Info("graceful shutdown starting", "timeout", s.cfg.Server.ShutdownTimeout)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("graceful shutdown failed — forcing server close", "error", err)
		_ = s.httpServer.Close()
	} else {
		s.logger.Info("graceful shutdown completed successfully")
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 7: Cleanup (Reverse order of creation)
	// ─────────────────────────────────────────────────────────────────────
	// 7a. Stop background workers
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

	s.logger.Info("server exited cleanly")
	return nil
}
