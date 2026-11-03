package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockPingable mocks a database pool or external service
type mockPingable struct {
	mu  sync.Mutex
	err error
}

func (m *mockPingable) PingContext(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *mockPingable) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func getFreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host port: %v", err)
	}
	return port
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(c *Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "empty public port",
			modify: func(c *Config) {
				c.PublicPort = ""
			},
			wantErr: true,
		},
		{
			name: "same public and management port",
			modify: func(c *Config) {
				c.PublicPort = "8080"
				c.ManagementPort = "8080"
				c.PublicHost = "127.0.0.1"
				c.ManagementHost = "127.0.0.1"
			},
			wantErr: true,
		},
		{
			name: "zero read timeout",
			modify: func(c *Config) {
				c.ReadTimeout = 0
			},
			wantErr: true,
		},
		{
			name: "zero shutdown timeout",
			modify: func(c *Config) {
				c.ShutdownTimeout = 0
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSynchronousPreBinding(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PublicHost = "127.0.0.1"
	cfg.PublicPort = getFreePort(t)
	cfg.ManagementHost = "127.0.0.1"
	cfg.ManagementPort = getFreePort(t)

	pubLn, mgmtLn, err := PreBindListeners(cfg)
	if err != nil {
		t.Fatalf("PreBindListeners failed: %v", err)
	}
	defer pubLn.Close()
	defer mgmtLn.Close()

	if pubLn.Addr() == nil || mgmtLn.Addr() == nil {
		t.Errorf("expected non-nil listener addresses")
	}

	// Test Rollback: occupies management port to force failure
	occupiedLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy port: %v", err)
	}
	defer occupiedLn.Close()

	_, occupiedPort, _ := net.SplitHostPort(occupiedLn.Addr().String())

	failCfg := DefaultConfig()
	failCfg.PublicHost = "127.0.0.1"
	failCfg.PublicPort = getFreePort(t)
	failCfg.ManagementHost = "127.0.0.1"
	failCfg.ManagementPort = occupiedPort

	pLn, mLn, err := PreBindListeners(failCfg)
	if err == nil {
		pLn.Close()
		mLn.Close()
		t.Fatalf("expected PreBindListeners to fail when management port is occupied")
	}

	// Verify public port was rolled back and released
	retryPubLn, err := net.Listen("tcp", failCfg.PublicAddress())
	if err != nil {
		t.Fatalf("expected rolled back public port to be freed, but got: %v", err)
	}
	_ = retryPubLn.Close()
}

func TestHealthChecker_Probes(t *testing.T) {
	mockDB := &mockPingable{}
	hc := NewHealthChecker()
	hc.RegisterDependency("postgres", mockDB)

	mux := http.NewServeMux()
	hc.RegisterRoutes(mux)

	// 1. Initial State: alive=true, ready=false
	reqLive := httptest.NewRequest("GET", "/livez", nil)
	wLive := httptest.NewRecorder()
	mux.ServeHTTP(wLive, reqLive)

	if wLive.Code != http.StatusOK {
		t.Errorf("expected /livez status 200, got %d", wLive.Code)
	}

	reqReady := httptest.NewRequest("GET", "/readyz", nil)
	wReady := httptest.NewRecorder()
	mux.ServeHTTP(wReady, reqReady)

	if wReady.Code != http.StatusServiceUnavailable {
		t.Errorf("expected initial /readyz status 503, got %d", wReady.Code)
	}

	// 2. Mark Ready
	hc.MarkReady()
	wReady2 := httptest.NewRecorder()
	mux.ServeHTTP(wReady2, reqReady)

	if wReady2.Code != http.StatusOK {
		t.Errorf("expected /readyz status 200 after MarkReady, got %d", wReady2.Code)
	}

	// 3. Degraded Dependency
	mockDB.SetError(errors.New("db connection timeout"))
	wReady3 := httptest.NewRecorder()
	mux.ServeHTTP(wReady3, reqReady)

	if wReady3.Code != http.StatusServiceUnavailable {
		t.Errorf("expected /readyz status 503 when DB degraded, got %d", wReady3.Code)
	}

	var res map[string]any
	_ = json.Unmarshal(wReady3.Body.Bytes(), &res)
	if res["status"] != "degraded" {
		t.Errorf("expected status degraded, got %v", res["status"])
	}

	// 4. Mark Dead
	hc.MarkDead()
	wLiveDead := httptest.NewRecorder()
	mux.ServeHTTP(wLiveDead, reqLive)

	if wLiveDead.Code != http.StatusServiceUnavailable {
		t.Errorf("expected /livez status 503 when marked dead, got %d", wLiveDead.Code)
	}
}

func TestDualServer_SecurityIsolation(t *testing.T) {
	publicMux := http.NewServeMux()
	publicMux.HandleFunc("GET /api/v1/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("public ok"))
	})

	handler := PublicSecurityIsolationMiddleware()(publicMux)

	// Business endpoint allowed
	reqPub := httptest.NewRequest("GET", "/api/v1/test", nil)
	wPub := httptest.NewRecorder()
	handler.ServeHTTP(wPub, reqPub)
	if wPub.Code != http.StatusOK {
		t.Errorf("expected 200 on public route, got %d", wPub.Code)
	}

	// Operational endpoints must return 404
	blockedPaths := []string{"/debug/vars", "/info", "/metrics", "/info/details"}
	for _, p := range blockedPaths {
		req := httptest.NewRequest("GET", p, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 on public port for blocked path %s, got %d", p, w.Code)
		}
	}
}

func TestManagementAuthMiddleware(t *testing.T) {
	token := "secret-operational-token"
	mw := ManagementAuthMiddleware(token)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("authorized"))
	}))

	// 1. Missing header
	reqNoAuth := httptest.NewRequest("GET", "/info", nil)
	wNoAuth := httptest.NewRecorder()
	handler.ServeHTTP(wNoAuth, reqNoAuth)
	if wNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth header, got %d", wNoAuth.Code)
	}

	// 2. Invalid token
	reqBadAuth := httptest.NewRequest("GET", "/info", nil)
	reqBadAuth.Header.Set("Authorization", "Bearer wrong-token")
	wBadAuth := httptest.NewRecorder()
	handler.ServeHTTP(wBadAuth, reqBadAuth)
	if wBadAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", wBadAuth.Code)
	}

	// 3. Valid token
	reqValidAuth := httptest.NewRequest("GET", "/info", nil)
	reqValidAuth.Header.Set("Authorization", "Bearer "+token)
	wValidAuth := httptest.NewRecorder()
	handler.ServeHTTP(wValidAuth, reqValidAuth)
	if wValidAuth.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", wValidAuth.Code)
	}
}

func TestHTTPLoggingAndProbeSuppression(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // Failing probe
	})
	mux.HandleFunc("GET /api/success", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/client-error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	mux.HandleFunc("GET /api/server-error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := HTTPLoggingMiddleware(logger)(mux)

	// 1. Healthy probe (should be suppressed)
	buf.Reset()
	reqLive := httptest.NewRequest("GET", "/livez", nil)
	wLive := httptest.NewRecorder()
	handler.ServeHTTP(wLive, reqLive)

	if buf.Len() != 0 {
		t.Errorf("expected healthy /livez log to be suppressed, but got: %s", buf.String())
	}

	// 2. Failing probe (should NOT be suppressed)
	buf.Reset()
	reqReady := httptest.NewRequest("GET", "/readyz", nil)
	wReady := httptest.NewRecorder()
	handler.ServeHTTP(wReady, reqReady)

	if buf.Len() == 0 {
		t.Errorf("expected failing /readyz (503) to be logged, but got empty buffer")
	}

	// 3. 500 error logged at ERROR
	buf.Reset()
	req500 := httptest.NewRequest("GET", "/api/server-error", nil)
	w500 := httptest.NewRecorder()
	handler.ServeHTTP(w500, req500)

	var log500 map[string]any
	if err := json.Unmarshal(buf.Bytes(), &log500); err != nil {
		t.Fatalf("failed to parse log json: %v", err)
	}
	if log500["level"] != "ERROR" {
		t.Errorf("expected level ERROR for 500, got %v", log500["level"])
	}

	// 4. 400 error logged at WARN
	buf.Reset()
	req400 := httptest.NewRequest("GET", "/api/client-error", nil)
	w400 := httptest.NewRecorder()
	handler.ServeHTTP(w400, req400)

	var log400 map[string]any
	if err := json.Unmarshal(buf.Bytes(), &log400); err != nil {
		t.Fatalf("failed to parse log json: %v", err)
	}
	if log400["level"] != "WARN" {
		t.Errorf("expected level WARN for 400, got %v", log400["level"])
	}
}

func TestWorkerManager_LifecycleAndPanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	wm := NewWorkerManager(logger)

	var executed sync.WaitGroup
	executed.Add(2)

	var stoppedNormally bool
	var mu sync.Mutex

	// Worker 1: Normal cancellable worker
	wm.Start("normal-worker", func(ctx context.Context) {
		executed.Done()
		<-ctx.Done()
		mu.Lock()
		stoppedNormally = true
		mu.Unlock()
	})

	// Worker 2: Panicking worker
	wm.Start("panic-worker", func(ctx context.Context) {
		executed.Done()
		panic("simulated fatal worker crash")
	})

	executed.Wait()
	time.Sleep(10 * time.Millisecond)

	// Stop all
	wm.StopAll()

	mu.Lock()
	defer mu.Unlock()
	if !stoppedNormally {
		t.Errorf("expected normal worker to observe cancellation context")
	}
}

func TestCompleteLifecycle_StartupDrainShutdown(t *testing.T) {
	tempDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.PublicHost = "127.0.0.1"
	cfg.PublicPort = getFreePort(t)
	cfg.ManagementHost = "127.0.0.1"
	cfg.ManagementPort = getFreePort(t)
	cfg.DrainDuration = 50 * time.Millisecond
	cfg.ShutdownTimeout = 2 * time.Second
	cfg.LogDir = tempDir
	cfg.LogToFile = true

	lc, err := New(cfg)
	if err != nil {
		t.Fatalf("Lifecycle New failed: %v", err)
	}

	cleanupOrder := make([]string, 0)
	var cleanupMu sync.Mutex

	lc.RegisterCleanup(func() error {
		cleanupMu.Lock()
		cleanupOrder = append(cleanupOrder, "db_pool")
		cleanupMu.Unlock()
		return nil
	})

	lc.RegisterCleanup(func() error {
		cleanupMu.Lock()
		cleanupOrder = append(cleanupOrder, "worker_pool")
		cleanupMu.Unlock()
		return nil
	})

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("GET /api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	lc.SetupServers(publicMux)
	lc.Start()

	// Wait for servers to be active
	time.Sleep(50 * time.Millisecond)

	// 1. Verify Public Business endpoint
	pubURL := fmt.Sprintf("http://%s/api/v1/ping", cfg.PublicAddress())
	resp, err := http.Get(pubURL)
	if err != nil {
		t.Fatalf("failed to call public endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from public ping, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. Verify Management endpoint on management port
	mgmtURL := fmt.Sprintf("http://%s/info", cfg.ManagementAddress())
	respMgmt, err := http.Get(mgmtURL)
	if err != nil {
		t.Fatalf("failed to call management endpoint: %v", err)
	}
	if respMgmt.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from management /info, got %d", respMgmt.StatusCode)
	}
	_ = respMgmt.Body.Close()

	// 3. Verify Isolation: Management endpoint on public port returns 404
	pubIsolatedURL := fmt.Sprintf("http://%s/info", cfg.PublicAddress())
	respIso, err := http.Get(pubIsolatedURL)
	if err != nil {
		t.Fatalf("failed to call public isolated endpoint: %v", err)
	}
	if respIso.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for /info on public port, got %d", respIso.StatusCode)
	}
	_ = respIso.Body.Close()

	// 4. Verify Readiness is active before shutdown
	readyURL := fmt.Sprintf("http://%s/readyz", cfg.PublicAddress())
	respReady, err := http.Get(readyURL)
	if err != nil {
		t.Fatalf("failed to check readyz: %v", err)
	}
	if respReady.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from readyz before shutdown, got %d", respReady.StatusCode)
	}
	_ = respReady.Body.Close()

	// 5. Trigger Shutdown (Phase 6, 7, 8)
	lc.Shutdown()

	// 6. Verify reverse-order cleanup: worker_pool then db_pool (LIFO)
	cleanupMu.Lock()
	if len(cleanupOrder) != 2 || cleanupOrder[0] != "worker_pool" || cleanupOrder[1] != "db_pool" {
		t.Errorf("expected LIFO cleanup order [worker_pool, db_pool], got %v", cleanupOrder)
	}
	cleanupMu.Unlock()

	// 7. Verify log file was created
	appLogPath := filepath.Join(tempDir, "app.log")
	if _, err := os.Stat(appLogPath); os.IsNotExist(err) {
		t.Errorf("expected %s to be created and written", appLogPath)
	}
}
