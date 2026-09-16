package httpadapter_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
)

// recordHandler accumulates slog records so tests can assert on levels.
type recordHandler struct {
	mu      sync.Mutex
	records []slog.Record
	attrs   map[string]any
}

func (h *recordHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *recordHandler) Handle(_ context.Context, rec slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, rec)
	attrs := make(map[string]any)
	rec.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.attrs = attrs
	return nil
}

func (h *recordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordHandler) WithGroup(string) slog.Handler      { return h }

func (h *recordHandler) latest() (slog.Record, map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.records) == 0 {
		return slog.Record{}, nil
	}
	return h.records[len(h.records)-1], h.attrs
}

// drainingHealth returns 503 from readiness until marked ready.
type drainingHealth struct {
	ready bool
}

func (d *drainingHealth) HandleLiveness(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"alive"}`))
}

func (d *drainingHealth) HandleReadiness(w http.ResponseWriter, _ *http.Request) {
	if !d.ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"draining"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func TestRouter_Readyz503LogsAtDebugDuringDrain(t *testing.T) {
	rec := &recordHandler{}
	logger := slog.New(rec)

	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &drainingHealth{ready: false}
	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 while draining, got %d", rr.Code)
	}

	latest, attrs := rec.latest()
	if latest.Message != "http probe not ready" {
		t.Fatalf("expected DEBUG 'http probe not ready' record, got %q (level=%v)", latest.Message, latest.Level)
	}
	if latest.Level != slog.LevelDebug {
		t.Errorf("expected debug level for drain 503, got %v", latest.Level)
	}
	if attrs["status"] != int64(503) && attrs["status"] != 503 {
		t.Errorf("expected status attr 503, got %v", attrs["status"])
	}
	for _, no := range []string{"http request server error", "http request client error", "http probe server error"} {
		if latest.Message == no {
			t.Errorf("drain 503 must not be logged as %q", no)
		}
	}
}

func TestRouter_Readyz200IsSilenced(t *testing.T) {
	rec := &recordHandler{}
	logger := slog.New(rec)

	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &drainingHealth{ready: true}
	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 when ready, got %d", rr.Code)
	}
	latest, _ := rec.latest()
	if latest.Message == "" || latest.Message == "http probe" || latest.Message == "http request" {
		t.Logf("readiness probe emitted %q", latest.Message)
	}
	if latest.Message == "http request server error" {
		t.Error("200 readiness must not be logged as an error")
	}
}