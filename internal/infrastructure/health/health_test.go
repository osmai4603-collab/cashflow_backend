package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cashflow_backend/internal/infrastructure/health"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthChecker_Liveness(t *testing.T) {
	// Even if dependency ping fails, liveness must return 200 OK!
	hc := health.NewHealthChecker(&mockPinger{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr := httptest.NewRecorder()

	hc.HandleLiveness(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /livez, got %d", rr.Code)
	}

	var res map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&res)
	if res["status"] != "alive" {
		t.Errorf("expected status 'alive', got '%s'", res["status"])
	}
}

func TestHealthChecker_Readiness(t *testing.T) {
	pinger := &mockPinger{err: nil}
	hc := health.NewHealthChecker(pinger)

	// 1. Initially NOT ready
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	hc.HandleReadiness(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 initially, got %d", rr.Code)
	}

	// 2. Mark ready
	hc.MarkReady()
	rr = httptest.NewRecorder()
	hc.HandleReadiness(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 after MarkReady, got %d", rr.Code)
	}

	// 3. Mark not ready (Drain phase)
	hc.MarkNotReady()
	rr = httptest.NewRecorder()
	hc.HandleReadiness(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 after MarkNotReady, got %d", rr.Code)
	}

	// 4. Dependency failure
	hc.MarkReady()
	pinger.err = errors.New("connection refused")
	rr = httptest.NewRecorder()
	hc.HandleReadiness(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when dependency fails, got %d", rr.Code)
	}
}
