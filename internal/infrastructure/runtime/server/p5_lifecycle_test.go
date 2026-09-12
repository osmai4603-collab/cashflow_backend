package server_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/infrastructure/runtime/health"
	"cashflow_backend/internal/infrastructure/runtime/server"
	"cashflow_backend/internal/infrastructure/runtime/worker"
)

// TestServer_PreBoundBindFailureNeverDeclaresReady locks gap 3.3: when the
// management port is occupied, Run must return a fatal error BEFORE readiness is
// ever reached, and the already-bound public listener must not leak.
func TestServer_PreBoundBindFailureNeverDeclaresReady(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	freeLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer freeLn.Close()

	cfg := newTestConfig()
	cfg.Management.Port = strconv.Itoa(freeLn.Addr().(*net.TCPAddr).Port)

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

	if hc.IsReady() {
		t.Error("readiness must never be reached when management bind fails")
	}

	acceptCh := make(chan error, 1)
	go func() { _, err := pubLn.Accept(); acceptCh <- err }()
	select {
	case err := <-acceptCh:
		if !errors.Is(err, net.ErrClosed) {
			t.Errorf("already-bound public listener must be closed after fatal bind failure, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("public listener still open after fatal bind failure")
	}
}

// TestServer_ShutdownReleasesBothListenersAndGoroutines verifies that a normal
// shutdown closes both servers, returns cleanly, and leaves no Serve goroutines
// behind.
func TestServer_ShutdownReleasesBothListenersAndGoroutines(t *testing.T) {
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

	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	// Let the servers start serving.
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get("http://" + srv.ManagementAddr() + "/metrics")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("management server never became reachable: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}

	if hc.IsReady() {
		t.Error("readiness must be false after shutdown")
	}

	assertListenerClosed(t, pubLn, "public")
	assertListenerClosed(t, mgtLn, "management")

	// No leaked Serve goroutines.
	settleDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(settleDeadline) {
		if runtime.NumGoroutine() <= baseline+4 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("goroutines did not settle after shutdown: baseline=%d now=%d", baseline, runtime.NumGoroutine())
}

// TestServer_ShutdownIsIdempotent locks the "double shutdown" lifecycle
// checklist item: PHASE 6 relies on http.Server.Shutdown, whose contract is
// that repeated calls return immediately with no error and never double-close
// the listener. A second teardown trigger must therefore never fatal or hang.
func TestServer_ShutdownIsIdempotent(t *testing.T) {
	var served atomic.Bool
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served.Store(true)
		w.WriteHeader(http.StatusOK)
	})}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() { _ = srv.Serve(ln) }()

	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get("http://" + ln.Addr().String())
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became reachable: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !served.Load() {
		t.Fatal("server did not serve a probe request")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("second (double) shutdown must be idempotent, got %v", err)
	}
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("third shutdown must also be idempotent, got %v", err)
	}
	if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Errorf("listener close after shutdown: %v", err)
	}
}

// assertListenerClosed verifies a listener returns net.ErrClosed from Accept
// within a bounded window.
func assertListenerClosed(t *testing.T, ln net.Listener, name string) {
	t.Helper()
	acceptCh := make(chan error, 1)
	go func() { _, err := ln.Accept(); acceptCh <- err }()
	select {
	case err := <-acceptCh:
		if !errors.Is(err, net.ErrClosed) {
			t.Errorf("%s listener still open after shutdown: %v", name, err)
		}
	case <-time.After(time.Second):
		t.Errorf("%s listener still open after shutdown", name)
	}
}

// TestServer_StructuredLifecycleLogs verifies the P5 structured log contract:
// server_role/addr on serve, shutdown_reason on drain/shutdown and final exit.
func TestServer_StructuredLifecycleLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
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

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}

	out := buf.String()
	for _, want := range []string{
		`server_role=public`,
		`server_role=management`,
		`shutdown_reason=signal`,
		`server exited cleanly`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("lifecycle logs missing %q", want)
		}
	}
	if strings.Contains(out, "server startup failed") {
		t.Error("old fatal log wording must not be emitted on a clean signal shutdown")
	}
}
