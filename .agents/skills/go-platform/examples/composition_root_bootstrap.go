package examples

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// AppConfig represents application-wide configuration.
type AppConfig struct {
	Port        string
	DatabaseDSN string
	Environment string
}

// AppContainer holds all initialized platform services and root components.
type AppContainer struct {
	Logger      *slog.Logger
	Clock       Clock
	IDGen       IDGenerator
	EventBus    *EventBus
	WorkerPool  *WorkerPool
	DB          *sql.DB
	Server      *http.Server
	cleanupFns  []func(ctx context.Context) error
}

// BootstrapApp demonstrates explicit topological wiring of the Platform layer and Composition Root.
func BootstrapApp(cfg AppConfig) (*AppContainer, error) {
	container := &AppContainer{
		cleanupFns: make([]func(ctx context.Context) error, 0),
	}

	// 1. Logger & Observability
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	container.Logger = logger

	// 2. Fundamental Platform Primitives (Clock, ID Generator, EventBus, Concurrency)
	clock := NewRealClock()
	container.Clock = clock
	container.IDGen = NewTimeOrderedIDGenerator(clock)
	container.EventBus = NewEventBus()

	workerPool := NewWorkerPool(4, 64)
	container.WorkerPool = workerPool
	container.cleanupFns = append(container.cleanupFns, func(ctx context.Context) error {
		workerPool.Shutdown()
		logger.Info("worker pool gracefully drained and stopped")
		return nil
	})

	// 3. Database Connection Pool with Official Go database/sql Tuning
	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	container.DB = db
	container.cleanupFns = append(container.cleanupFns, func(ctx context.Context) error {
		err := db.Close()
		logger.Info("database pool closed")
		return err
	})

	// 4. HTTP Router & Presentation Setup
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	container.Server = srv
	container.cleanupFns = append(container.cleanupFns, func(ctx context.Context) error {
		logger.Info("shutting down HTTP server...")
		return srv.Shutdown(ctx)
	})

	logger.Info("application platform dependencies successfully wired", "env", cfg.Environment)
	return container, nil
}

// RunWithGracefulShutdown runs the application and listens for termination signals.
func (c *AppContainer) RunWithGracefulShutdown() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	go func() {
		c.Logger.Info("HTTP server listening", "addr", c.Server.Addr)
		if err := c.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		c.Logger.Info("termination signal received, initiating graceful teardown")
		return c.Teardown(10 * time.Second)
	}
}

// Teardown executes cleanup in STRICT REVERSE ORDER of initialization.
// Step 1: Drain HTTP server traffic.
// Step 2: Drain background workers.
// Step 3: Close database connection pool last, preventing queries from crashing.
func (c *AppContainer) Teardown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var aggErr error
	for i := len(c.cleanupFns) - 1; i >= 0; i-- {
		fn := c.cleanupFns[i]
		if err := fn(ctx); err != nil {
			c.Logger.Error("error during resource teardown", "err", err)
			aggErr = errors.Join(aggErr, err)
		}
	}

	if aggErr != nil {
		return fmt.Errorf("teardown completed with errors: %w", aggErr)
	}
	c.Logger.Info("all application platform resources released cleanly in reverse order")
	return nil
}
