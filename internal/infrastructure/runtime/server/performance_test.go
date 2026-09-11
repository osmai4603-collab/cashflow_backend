package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/platform/config"
)

type mockHealth struct{}

func (m *mockHealth) HandleLiveness(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(http.StatusOK) }
func (m *mockHealth) HandleReadiness(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func BenchmarkRouterRoot(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := &config.Configuration{}

	handlers := &httpadapter.CashflowHandlers{
		Base: httpadapter.NewBaseHandler("test", "1.0", logger),
	}

	router := httpadapter.NewRouterWithHandlers(handlers, &mockHealth{}, logger, cfg)

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouterReadyz(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := &config.Configuration{}

	router := httpadapter.NewRouterWithHandlers(&httpadapter.CashflowHandlers{}, &mockHealth{}, logger, cfg)

	req := httptest.NewRequest("GET", "/readyz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkRouterMetrics(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := &config.Configuration{}

	router := httpadapter.NewRouterWithHandlers(&httpadapter.CashflowHandlers{}, &mockHealth{}, logger, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
