package examples

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// AppConfig represents application-wide configuration.
type AppConfig struct {
	Port        string
	DatabaseDSN string
	Environment string
}

// AppContainer holds all initialized services ready to serve traffic.
type AppContainer struct {
	Logger          *slog.Logger
	DB              *sql.DB
	NotificationBus *NotificationBus
	PaymentRegistry *ProviderRegistry
	Router          http.Handler
	cleanupFns      []func()
}

// BootstrapApp demonstrates explicit wiring of Platform and Infrastructure layers.
func BootstrapApp(cfg AppConfig) (*AppContainer, error) {
	container := &AppContainer{
		cleanupFns: make([]func(), 0),
	}

	// 1. Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	container.Logger = logger

	// 2. Database Connection Pool
	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	container.DB = db
	container.cleanupFns = append(container.cleanupFns, func() {
		_ = db.Close()
		logger.Info("database pool closed")
	})

	// 3. Platform Primitives
	notifBus := NewNotificationBus()
	container.NotificationBus = notifBus

	// 4. Infrastructure Outbound Adapters
	paymentRegistry := NewProviderRegistry()
	mockAdapter := NewRemotePaymentAdapter("default", "https://api.payment.example.com", "secret-key")
	paymentRegistry.Register("default", mockAdapter)
	container.PaymentRegistry = paymentRegistry

	// 5. Storage Repositories & Use Cases would be instantiated here...

	// 6. Presentation Router
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	container.Router = mux

	logger.Info("application dependencies successfully wired", "env", cfg.Environment)
	return container, nil
}

// Teardown executes cleanup in REVERSE order of initialization.
func (c *AppContainer) Teardown() {
	var wg sync.WaitGroup
	for i := len(c.cleanupFns) - 1; i >= 0; i-- {
		fn := c.cleanupFns[i]
		wg.Add(1)
		go func(clean func()) {
			defer wg.Done()
			clean()
		}(fn)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.Logger.Info("all application resources released cleanly")
	case <-time.After(5 * time.Second):
		c.Logger.Warn("teardown timed out waiting for resources")
	}
}
