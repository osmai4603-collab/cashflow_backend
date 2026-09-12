package httpadapter_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/infrastructure/runtime/metrics"
	platconfig "cashflow_backend/internal/platform/config"
)

func managementTestServer(t *testing.T, mutate func(*platconfig.Configuration)) http.Handler {
	t.Helper()
	cfg := platconfig.Defaults()
	if mutate != nil {
		mutate(cfg)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpadapter.NewManagementRouter(cfg, &mockHealthRoutes{}, logger)
}

func TestManagementRouter_DefaultRoutes(t *testing.T) {
	router := managementTestServer(t, nil)

	cases := []struct {
		path     string
		wantCode int
		wantCT   string
	}{
		{"/", http.StatusOK, "text/html"},
		{"/metrics", http.StatusOK, metrics.PrometheusContentType},
		{"/metrics/json", http.StatusOK, "application/json"},
		{"/livez", http.StatusOK, ""},
		{"/readyz", http.StatusOK, ""},
		// pprof is disabled by default
		{"/debug/pprof/", http.StatusNotFound, ""},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		router.ServeHTTP(rr, req)

		if rr.Code != tc.wantCode {
			t.Errorf("%s: expected %d, got %d", tc.path, tc.wantCode, rr.Code)
			continue
		}
		if tc.wantCT != "" && !strings.HasPrefix(rr.Header().Get("Content-Type"), tc.wantCT) {
			t.Errorf("%s: expected Content-Type prefix %q, got %q", tc.path, tc.wantCT, rr.Header().Get("Content-Type"))
		}
	}
}

func TestManagementRouter_PrometheusEndpointsParse(t *testing.T) {
	router := managementTestServer(t, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "cashflow_http_requests_total") && !strings.Contains(body, "cashflow_process_uptime_seconds") {
		t.Fatal("expected cashflow_* families in /metrics text output")
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/metrics/json", nil)
	router.ServeHTTP(rr, req)
	var m map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("expected valid JSON on /metrics/json: %v", err)
	}
	if _, ok := m["http_requests_total"]; !ok {
		t.Error("expected http_requests_total in /metrics/json")
	}
}

func TestManagementRouter_AuthEnforced(t *testing.T) {
	router := managementTestServer(t, func(cfg *platconfig.Configuration) {
		cfg.Management.RequireAuth = true
		cfg.Management.AuthToken = "s3cret-token"
	})

	// No token -> 401
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rr.Code)
	}

	// Wrong token -> 401
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", rr.Code)
	}

	// Correct token -> 200 on every management route
	for _, path := range []string{"/", "/metrics", "/metrics/json", "/livez", "/readyz"} {
		rr = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer s3cret-token")
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("%s: expected 200 with valid token, got %d", path, rr.Code)
		}
	}
}

func TestManagementRouter_PprofOptInOnly(t *testing.T) {
	// Disabled by default -> 404.
	router := managementTestServer(t, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when pprof disabled, got %d", rr.Code)
	}

	// Enabled explicitly -> 200.
	router = managementTestServer(t, func(cfg *platconfig.Configuration) {
		cfg.Management.PprofEnabled = true
	})
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when pprof enabled, got %d", rr.Code)
	}
}

func TestManagementRouter_MetricsDisabled(t *testing.T) {
	router := managementTestServer(t, func(cfg *platconfig.Configuration) {
		cfg.Management.MetricsEnabled = false
	})

	for _, path := range []string{"/metrics", "/metrics/json"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404 when metrics disabled, got %d", path, rr.Code)
		}
	}
	// Index still serves, probes still serve.
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on index with metrics disabled, got %d", rr.Code)
	}
}

func TestManagementRouter_AuthRejectsDoNotLeakHeader(t *testing.T) {
	// Ensures rejected requests never echo back the attempted token.
	var output strings.Builder
	logger := slog.New(slog.NewTextHandler(&output, nil))
	cfg := platconfig.Defaults()
	cfg.Management.RequireAuth = true
	cfg.Management.AuthToken = "real-token"
	router := httpadapter.NewManagementRouter(cfg, &mockHealthRoutes{}, logger)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer attempted-secret")
	router.ServeHTTP(rr, req)

	if strings.Contains(rr.Body.String(), "attempted-secret") {
		t.Error("rejected request leaked the attempted token in the response body")
	}
	if strings.Contains(output.String(), "attempted-secret") {
		t.Error("rejected request leaked the attempted token in the logs")
	}
}
