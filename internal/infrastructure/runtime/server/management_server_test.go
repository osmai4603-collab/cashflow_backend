package server_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/infrastructure/runtime/health"
	"cashflow_backend/internal/infrastructure/runtime/metrics"
	"cashflow_backend/internal/infrastructure/runtime/server"
	"cashflow_backend/internal/infrastructure/runtime/worker"
	platconfig "cashflow_backend/internal/platform/config"
)

func newTestConfig() *platconfig.Configuration {
	cfg := platconfig.Defaults()
	cfg.Server.Interface = "127.0.0.1"
	cfg.Server.Port = "0"
	cfg.Server.ReadTimeout = 2 * time.Second
	cfg.Server.ReadHeaderTimeout = 1 * time.Second
	cfg.Server.WriteTimeout = 2 * time.Second
	cfg.Server.IdleTimeout = 5 * time.Second
	cfg.Server.DrainDuration = 10 * time.Millisecond
	cfg.Server.ShutdownTimeout = 1 * time.Second
	cfg.Server.MaxHeaderBytes = 1 << 20
	return cfg
}

// TestServer_ManagementListenerServesMetrics verifies the isolated 8066-style
// listener serves the index, Prometheus text and JSON, and that shutdown closes
// both listeners.
func TestServer_ManagementListenerServesMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := newTestConfig()

	hc := health.NewHealthChecker(dummyPinger{})
	handler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	router := httpadapter.NewRouter(handler, hc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	wm := worker.NewWorkerManager(logger)

	srv := server.NewServer(cfg, router, hc, wm, logger)
	srv.SetManagementHandler(httpadapter.NewManagementRouter(cfg, hc, logger))

	pubLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen public: %v", err)
	}
	mgtLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen management: %v", err)
	}
	srv.SetListener(pubLn)
	srv.SetManagementListener(mgtLn)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	mgtURL := "http://" + srv.ManagementAddr()

	t.Run("index", func(t *testing.T) {
		resp, err := http.Get(mgtURL + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 on management index, got %d", resp.StatusCode)
		}
	})

	t.Run("metrics text", func(t *testing.T) {
		resp, err := http.Get(mgtURL + "/metrics")
		if err != nil {
			t.Fatalf("GET /metrics: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on /metrics, got %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != metrics.PrometheusContentType {
			t.Errorf("unexpected Content-Type %q", ct)
		}
	})

	t.Run("metrics json", func(t *testing.T) {
		resp, err := http.Get(mgtURL + "/metrics/json")
		if err != nil {
			t.Fatalf("GET /metrics/json: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 on /metrics/json, got %d", resp.StatusCode)
		}
		var m map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			t.Fatalf("expected valid JSON: %v", err)
		}
		if _, ok := m["http_requests_total"]; !ok {
			t.Error("expected http_requests_total in JSON output")
		}
	})

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}
}

// TestServer_ManagementBindFailureIsFatal asserts a bind error on the
// management port prevents startup (fatal), so a server can never run with
// observability silently absent.
func TestServer_ManagementBindFailureIsFatal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Occupy a port so the management ListenAndServe must fail.
	freeLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer freeLn.Close()
	occupiedPort := freeLn.Addr().(*net.TCPAddr).Port

	cfg := newTestConfig()
	cfg.Management.Port = strconv.Itoa(occupiedPort)

	hc := health.NewHealthChecker(dummyPinger{})
	handler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	router := httpadapter.NewRouter(handler, hc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
	wm := worker.NewWorkerManager(logger)

	srv := server.NewServer(cfg, router, hc, wm, logger)
	srv.SetManagementHandler(httpadapter.NewManagementRouter(cfg, hc, logger))

	pubLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen public: %v", err)
	}
	srv.SetListener(pubLn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected fatal error when management port is unavailable")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return a fatal error on management bind failure")
	}
}
