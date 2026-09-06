package lifecycle

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// =============================================================================
// Complete Production-Ready Server Lifecycle
// =============================================================================
// This file combines ALL 7 phases into a single, production-ready example.
//
// Phase 1: Initialization  → Load config, create dependencies
// Phase 2: Configuration   → Set server timeouts and TLS
// Phase 3: Startup         → Non-blocking ListenAndServe in goroutine
// Phase 4: Serving         → Health checks: /livez, /readyz
// Phase 5: Drain           → Mark not-ready, wait for LB removal
// Phase 6: Graceful Shutdown → server.Shutdown(ctx) with timeout
// Phase 7: Cleanup         → Close DB, workers, files (reverse order)
//
// All patterns follow official Go documentation:
//   - https://pkg.go.dev/net/http#Server
//   - https://pkg.go.dev/net/http#Server.Shutdown
//   - https://pkg.go.dev/os/signal#NotifyContext
//   - https://pkg.go.dev/context
//   - https://pkg.go.dev/sync#WaitGroup
// =============================================================================

func main() {
	// ─────────────────────────────────────────────────────────────────────
	// PHASE 1: Initialization
	// ─────────────────────────────────────────────────────────────────────

	// 1a. Create the logger first — everything else uses it.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// 1b. Load configuration.
	cfg := loadConfig()

	// 1c. Initialize dependencies (DB, cache, etc.)
	// If this fails, exit immediately — don't start a broken server.
	db, err := initDB(cfg.DatabaseDSN)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	logger.Info("dependencies initialized")

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 2: Configuration
	// ─────────────────────────────────────────────────────────────────────

	// 2a. Create the health checker.
	health := &healthChecker{db: db}

	// 2b. Create the router and register handlers.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", health.handleLiveness)
	mux.HandleFunc("GET /readyz", health.handleReadiness)
	mux.HandleFunc("GET /", handleRoot)

	// 2c. Create the server with ALL timeouts configured.
	// NEVER use a zero-value http.Server in production.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 3: Startup (Non-blocking)
	// ─────────────────────────────────────────────────────────────────────

	// 3a. Create a worker manager for background tasks.
	wm := newWorkerManager(logger)

	// 3b. Start example background workers.
	wm.start("metrics-reporter", func(ctx context.Context) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Info("reporting metrics...")
			}
		}
	})

	// 3c. Start the HTTP server in a goroutine.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 3d. Mark the server as ready after startup.
	health.markReady()
	logger.Info("server is ready to accept traffic")

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 4: Serving (wait for signal or error)
	// ─────────────────────────────────────────────────────────────────────

	// 4a. Register signal handler.
	sigCtx, sigStop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // SIGTERM (Kubernetes, systemd)
	)
	defer sigStop()

	// 4b. Wait for either a shutdown signal or a server error.
	select {
	case err := <-serverErr:
		logger.Error("server failed", "error", err)
		os.Exit(1)
	case <-sigCtx.Done():
		logger.Info("shutdown signal received")
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 5: Drain
	// ─────────────────────────────────────────────────────────────────────

	// 5a. Mark as not ready → LB stops sending new traffic.
	health.markNotReady()
	logger.Info("drain phase: marked as not ready")

	// 5b. Wait for LB to update its routing table.
	drainTimeout := time.Duration(cfg.DrainSeconds) * time.Second
	logger.Info("drain phase: waiting for LB to update", "timeout", drainTimeout)
	time.Sleep(drainTimeout)

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 6: Graceful Shutdown
	// ─────────────────────────────────────────────────────────────────────

	shutdownTimeout := time.Duration(cfg.ShutdownSeconds) * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	logger.Info("graceful shutdown starting", "timeout", shutdownTimeout)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed — forcing close", "error", err)
		srv.Close()
	} else {
		logger.Info("graceful shutdown completed")
	}

	// ─────────────────────────────────────────────────────────────────────
	// PHASE 7: Cleanup (reverse order of creation)
	// ─────────────────────────────────────────────────────────────────────

	// 7a. Stop background workers (they may still be writing to DB).
	wm.stopAll()

	// 7b. Close database connection.
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		} else {
			logger.Info("database connection closed")
		}
	}

	// 7c. Final message.
	logger.Info("server exited cleanly")
}

// ─────────────────────────────────────────────────────────────────────────────
// Supporting Types and Functions
// ─────────────────────────────────────────────────────────────────────────────

// config holds application configuration.
type config struct {
	Port            string
	DatabaseDSN     string
	ShutdownSeconds int
	DrainSeconds    int
}

func loadConfig() *config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &config{
		Port:            port,
		DatabaseDSN:     os.Getenv("DATABASE_DSN"),
		ShutdownSeconds: 10,
		DrainSeconds:    5,
	}
}

func initDB(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, nil // No DB configured — skip
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// healthChecker manages liveness and readiness state.
type healthChecker struct {
	ready atomic.Bool
	db    *sql.DB
}

func (h *healthChecker) markReady()    { h.ready.Store(true) }
func (h *healthChecker) markNotReady() { h.ready.Store(false) }

func (h *healthChecker) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness: ONLY check process health. NEVER check DB.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func (h *healthChecker) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !h.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"})
		return
	}

	// Readiness: check critical dependencies.
	if h.db != nil {
		if err := h.db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not_ready",
				"reason": "database unreachable",
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello, World!",
	})
}

// workerManager manages background goroutines.
type workerManager struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

func newWorkerManager(logger *slog.Logger) *workerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &workerManager{ctx: ctx, cancel: cancel, logger: logger}
}

func (wm *workerManager) start(name string, fn func(ctx context.Context)) {
	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()
		wm.logger.Info("worker started", "name", name)
		fn(wm.ctx)
		wm.logger.Info("worker stopped", "name", name)
	}()
}

func (wm *workerManager) stopAll() {
	wm.logger.Info("stopping all workers...")
	wm.cancel()
	wm.wg.Wait()
	wm.logger.Info("all workers stopped")
}
