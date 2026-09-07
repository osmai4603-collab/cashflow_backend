package server_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/infrastructure/health"
	"cashflow_backend/internal/infrastructure/server"
	"cashflow_backend/internal/infrastructure/worker"
	platconfig "cashflow_backend/internal/platform/config"
)

type trackCloser struct {
	closed atomic.Bool
}

func (t *trackCloser) Close() error {
	t.closed.Store(true)
	return nil
}

type dummyPinger struct{}

func (dummyPinger) Ping(ctx context.Context) error {
	return nil
}

func TestServer_FullLifecycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := platconfig.Defaults()
	cfg.Server.Interface = "127.0.0.1"
	cfg.Server.Port = "0"
	cfg.Server.ReadTimeout = 2 * time.Second
	cfg.Server.ReadHeaderTimeout = 1 * time.Second
	cfg.Server.WriteTimeout = 2 * time.Second
	cfg.Server.IdleTimeout = 5 * time.Second
	cfg.Server.DrainDuration = 50 * time.Millisecond
	cfg.Server.ShutdownTimeout = 1 * time.Second
	cfg.Server.MaxHeaderBytes = 1 << 20

	handler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	hc := health.NewHealthChecker(dummyPinger{})
	router := httpadapter.NewRouter(handler, hc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	wm := worker.NewWorkerManager(logger)
	customResource := &trackCloser{}

	srv := server.NewServer(cfg, router, hc, wm, logger, customResource)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on dynamic port: %v", err)
	}
	srv.SetListener(ln)

	ctx, cancel := context.WithCancel(context.Background())
	serverErrCh := make(chan error, 1)

	go func() {
		serverErrCh <- srv.Run(ctx)
	}()

	// Wait briefly for startup
	time.Sleep(50 * time.Millisecond)

	baseURL := "http://" + srv.Addr()

	// 1. Verify /livez returns 200
	resp, err := http.Get(baseURL + "/livez")
	if err != nil {
		t.Fatalf("failed to GET /livez: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on /livez, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Verify /readyz returns 200
	resp, err = http.Get(baseURL + "/readyz")
	if err != nil {
		t.Fatalf("failed to GET /readyz: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on /readyz, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Verify root endpoint
	resp, err = http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("failed to GET /: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on /, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Trigger shutdown
	cancel()

	// Wait for server to finish shutdown
	select {
	case err := <-serverErrCh:
		if err != nil {
			t.Errorf("expected clean server shutdown, got error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server shutdown timed out")
	}

	// 4. Verify resources were closed in cleanup phase
	if !customResource.closed.Load() {
		t.Errorf("expected custom resource to be closed during cleanup phase")
	}
}
