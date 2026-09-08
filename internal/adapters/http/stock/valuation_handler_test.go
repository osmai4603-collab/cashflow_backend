package stockhttp_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	stockhttp "cashflow_backend/internal/adapters/http/stock"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func setupTestServer() (*chi.Mux, *stockusecase.UseCase) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil, logger)
	h := stockhttp.NewHandler(uc, logger)

	r := chi.NewRouter()
	r.Get("/valuations", h.GetValuationSummaries)
	return r, uc
}

func TestStockHandler_GetValuationSummaries(t *testing.T) {
	r, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/valuations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
