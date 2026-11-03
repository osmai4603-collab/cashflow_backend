package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// CloserFunc is a function that cleans up a resource during teardown.
type CloserFunc func() error

// Lifecycle coordinates the complete eight-phase operational lifecycle
// of the Go HTTP service.
type Lifecycle struct {
	cfg       *Config
	logger    *slog.Logger
	logCloser *LogCloser
	health    *HealthChecker
	workers   *WorkerManager

	pubLn  net.Listener
	mgmtLn net.Listener

	publicServer *http.Server
	mgmtServer   *http.Server

	mu          sync.Mutex
	cleanups    []CloserFunc
	serversWg   sync.WaitGroup
	serverErrCh chan error
	startTime   time.Time
}

// New constructs and initializes a Lifecycle coordinator.
// Executes Phase 1 (Fail-Fast Initialization) and Phase 2 (Synchronous Pre-Binding).
func New(cfg *Config) (*Lifecycle, error) {
	if cfg == nil {
		var err error
		cfg, err = LoadConfigFromEnv()
		if err != nil {
			return nil, fmt.Errorf("phase 1 (initialization) failed: %w", err)
		}
	} else if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("phase 1 (config validation) failed: %w", err)
	}

	// 1. Initialize multi-target logger
	logger, logCloser, err := InitLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("phase 1 (logger init) failed: %w", err)
	}

	// 2. Synchronous Listener Pre-Binding (Phase 2)
	pubLn, mgmtLn, err := PreBindListeners(cfg)
	if err != nil {
		if logCloser != nil {
			_ = logCloser.Close()
		}
		return nil, fmt.Errorf("phase 2 (pre-binding) failed: %w", err)
	}

	health := NewHealthChecker()
	workers := NewWorkerManager(logger)

	lc := &Lifecycle{
		cfg:         cfg,
		logger:      logger,
		logCloser:   logCloser,
		health:      health,
		workers:     workers,
		pubLn:       pubLn,
		mgmtLn:      mgmtLn,
		serverErrCh: make(chan error, 2),
		startTime:   time.Now(),
	}

	logger.Info("phase 1 & 2 completed: listeners pre-bound synchronously",
		"public_addr", pubLn.Addr().String(),
		"management_addr", mgmtLn.Addr().String(),
	)

	return lc, nil
}

// Logger returns the initialized structured logger.
func (lc *Lifecycle) Logger() *slog.Logger {
	return lc.logger
}

// Health returns the HealthChecker.
func (lc *Lifecycle) Health() *HealthChecker {
	return lc.health
}

// Workers returns the WorkerManager.
func (lc *Lifecycle) Workers() *WorkerManager {
	return lc.workers
}

// Config returns the active lifecycle configuration.
func (lc *Lifecycle) Config() *Config {
	return lc.cfg
}

// PublicListener returns the synchronously bound public TCP listener.
func (lc *Lifecycle) PublicListener() net.Listener {
	return lc.pubLn
}

// ManagementListener returns the synchronously bound management TCP listener.
func (lc *Lifecycle) ManagementListener() net.Listener {
	return lc.mgmtLn
}

// RegisterCleanup registers a teardown cleanup hook executed during Phase 8.
// Hooks are executed in LIFO (reverse) order.
func (lc *Lifecycle) RegisterCleanup(fn CloserFunc) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.cleanups = append(lc.cleanups, fn)
}

// RegisterDependency adds a Pingable dependency (e.g. database pool)
// to readiness checks, and automatically registers its Close() method in cleanups.
func (lc *Lifecycle) RegisterDependency(name string, dep Pingable) {
	lc.health.RegisterDependency(name, dep)
	if closer, ok := dep.(io.Closer); ok {
		lc.RegisterCleanup(func() error {
			lc.logger.Info("closing dependency pool", "dependency", name)
			return closer.Close()
		})
	}
}

// SetupServers builds the public and management http.Servers with strict timeouts (Phase 3).
func (lc *Lifecycle) SetupServers(publicHandler http.Handler) {
	// Build Public Router with health probes and logging
	publicMux := http.NewServeMux()
	lc.health.RegisterRoutes(publicMux)

	if publicHandler != nil {
		publicMux.Handle("/", publicHandler)
	}

	// Wrap public handler with security isolation and logging
	securePublicHandler := PublicSecurityIsolationMiddleware()(publicMux)
	loggedPublicHandler := HTTPLoggingMiddleware(lc.logger)(securePublicHandler)

	lc.publicServer = &http.Server{
		Handler:           loggedPublicHandler,
		ReadTimeout:       lc.cfg.ReadTimeout,
		ReadHeaderTimeout: lc.cfg.ReadHeaderTimeout,
		WriteTimeout:      lc.cfg.WriteTimeout,
		IdleTimeout:       lc.cfg.IdleTimeout,
		MaxHeaderBytes:    lc.cfg.MaxHeaderBytes,
	}

	// Build Management Router with diagnostics and token auth
	mgmtHandler := NewManagementRouter(lc.cfg, lc.health, lc.startTime)
	loggedMgmtHandler := HTTPLoggingMiddleware(lc.logger)(mgmtHandler)

	lc.mgmtServer = &http.Server{
		Handler:           loggedMgmtHandler,
		ReadTimeout:       lc.cfg.ReadTimeout,
		ReadHeaderTimeout: lc.cfg.ReadHeaderTimeout,
		WriteTimeout:      lc.cfg.WriteTimeout,
		IdleTimeout:       lc.cfg.IdleTimeout,
		MaxHeaderBytes:    lc.cfg.MaxHeaderBytes,
	}

	lc.logger.Info("phase 3 completed: servers configured with strict timeouts",
		"read_timeout", lc.cfg.ReadTimeout.String(),
		"read_header_timeout", lc.cfg.ReadHeaderTimeout.String(),
		"write_timeout", lc.cfg.WriteTimeout.String(),
		"idle_timeout", lc.cfg.IdleTimeout.String(),
		"max_header_bytes", lc.cfg.MaxHeaderBytes,
	)
}

// Start launches both servers on pre-bound listeners and sets service ready (Phase 4 & 5).
func (lc *Lifecycle) Start() {
	if lc.publicServer == nil {
		lc.SetupServers(nil)
	}

	// Launch Public Server
	lc.serversWg.Add(1)
	go func() {
		defer lc.serversWg.Done()
		lc.logger.Info("public server serving traffic", "addr", lc.pubLn.Addr().String())
		if err := lc.publicServer.Serve(lc.pubLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lc.logger.Error("public server fatal serve error", "error", err)
			lc.serverErrCh <- fmt.Errorf("public server: %w", err)
		}
	}()

	// Launch Management Server
	lc.serversWg.Add(1)
	go func() {
		defer lc.serversWg.Done()
		lc.logger.Info("management server serving traffic", "addr", lc.mgmtLn.Addr().String())
		if err := lc.mgmtServer.Serve(lc.mgmtLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			lc.logger.Error("management server fatal serve error", "error", err)
			lc.serverErrCh <- fmt.Errorf("management server: %w", err)
		}
	}()

	// Phase 5: Mark ready once serving
	lc.health.MarkReady()
	lc.logger.Info("phase 4 & 5 completed: servers active and marked ready")
}

// Run starts the servers and coordinates POSIX signal handling, drain, shutdown, and cleanup.
// Blocks until the entire lifecycle completes.
func (lc *Lifecycle) Run(publicHandler http.Handler) error {
	lc.SetupServers(publicHandler)
	lc.Start()

	// POSIX signal handling (Phase 6 trigger)
	sigCtx, sigStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer sigStop()

	select {
	case err := <-lc.serverErrCh:
		lc.logger.Error("fatal runtime server error encountered", "error", err)
		lc.Shutdown()
		return err
	case <-sigCtx.Done():
		lc.logger.Info("os termination signal received, initiating graceful shutdown")
		lc.Shutdown()
		return nil
	}
}

// Shutdown executes Phase 6 (Drain), Phase 7 (Teardown), and Phase 8 (Cleanup).
func (lc *Lifecycle) Shutdown() {
	// ─────────────────────────────────────────────────────────────────
	// Phase 6: Two-Stage Drain Phase
	// ─────────────────────────────────────────────────────────────────
	lc.logger.Info("phase 6: starting two-stage drain phase")
	// 1. Mark not ready immediately (/readyz returns 503)
	lc.health.MarkNotReady()

	// 2. Wait drain duration for ingress/mesh to reroute traffic
	if lc.cfg.DrainDuration > 0 {
		lc.logger.Info("waiting for ingress traffic drain", "duration", lc.cfg.DrainDuration.String())
		time.Sleep(lc.cfg.DrainDuration)
	}

	// ─────────────────────────────────────────────────────────────────
	// Phase 7: Coordinated Graceful Teardown
	// ─────────────────────────────────────────────────────────────────
	lc.logger.Info("phase 7: coordinated graceful teardown started",
		"shutdown_timeout", lc.cfg.ShutdownTimeout.String(),
	)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), lc.cfg.ShutdownTimeout)
	defer cancel()

	var shutdownWg sync.WaitGroup

	if lc.publicServer != nil {
		shutdownWg.Add(1)
		go func() {
			defer shutdownWg.Done()
			if err := lc.publicServer.Shutdown(shutdownCtx); err != nil {
				lc.logger.Error("public server shutdown error, forcing close", "error", err)
				_ = lc.publicServer.Close()
			}
		}()
	}

	if lc.mgmtServer != nil {
		shutdownWg.Add(1)
		go func() {
			defer shutdownWg.Done()
			if err := lc.mgmtServer.Shutdown(shutdownCtx); err != nil {
				lc.logger.Error("management server shutdown error, forcing close", "error", err)
				_ = lc.mgmtServer.Close()
			}
		}()
	}

	shutdownWg.Wait()
	lc.serversWg.Wait()
	lc.logger.Info("both http servers stopped accepting traffic")

	// ─────────────────────────────────────────────────────────────────
	// Phase 8: Resource Cleanup (Strict Reverse Order)
	// ─────────────────────────────────────────────────────────────────
	lc.logger.Info("phase 8: executing reverse-order cleanup")

	// 1. Stop background workers before closing database pools
	if lc.workers != nil {
		lc.workers.StopAll()
	}

	// 2. Execute registered cleanup callbacks in reverse order (LIFO)
	lc.mu.Lock()
	cleanups := make([]CloserFunc, len(lc.cleanups))
	copy(cleanups, lc.cleanups)
	lc.mu.Unlock()

	for i := len(cleanups) - 1; i >= 0; i-- {
		if err := cleanups[i](); err != nil {
			lc.logger.Error("cleanup handler failed", "error", err)
		}
	}

	// 3. Flush and close logs
	if lc.logCloser != nil {
		_ = lc.logCloser.Sync()
		_ = lc.logCloser.Close()
	}

	lc.logger.Info("lifecycle shutdown complete: all resources cleaned up cleanly")
}
