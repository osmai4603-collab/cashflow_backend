package httpadapter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRouterBaseline_ObservabilityRoutesRemovedFromPublicPort locks the POST-P4
// behavior: /metrics and /debug (pprof) are NO LONGER served on the public port.
// They live exclusively on the isolated management listener (127.0.0.1:8066).
func TestRouterBaseline_ObservabilityRoutesRemovedFromPublicPort(t *testing.T) {
	router := setupTestServer()

	for _, path := range []string{"/metrics", "/debug/pprof/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			t.Errorf("expected public router to REJECT %s (P4 isolation), got %d", path, rr.Code)
		}
	}
}

// TestRouterBaseline_ProbesRemainPublic locks that liveness and readiness probes
// keep working on the public port (required by Kubernetes) even after P4.
func TestRouterBaseline_ProbesRemainPublic(t *testing.T) {
	router := setupTestServer()

	for _, path := range []string{"/livez", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 on %s, got %d", path, rr.Code)
		}
	}
}
