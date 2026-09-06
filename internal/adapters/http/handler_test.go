package httpadapter_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	producthttp "cashflow_backend/internal/adapters/http/product"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	"cashflow_backend/internal/platform/response"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	productusecase "cashflow_backend/internal/usecase/product"
)

type mockHealthRoutes struct{}

func (m *mockHealthRoutes) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"alive"}`))
}

func (m *mockHealthRoutes) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func setupTestServer() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}
	return httpadapter.NewRouter(handler, health, nil, nil, logger)
}

func TestRouter_Endpoints(t *testing.T) {
	router := setupTestServer()

	// 1. Root /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /, got %d", rr.Code)
	}

	var rootEnv response.Envelope
	if err := json.NewDecoder(rr.Body).Decode(&rootEnv); err != nil {
		t.Fatalf("failed to decode root response: %v", err)
	}
	if !rootEnv.Success {
		t.Errorf("expected success=true on root endpoint")
	}

	// 2. Health routes
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /livez, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /readyz, got %d", rr.Code)
	}

	// 3. API v1 mount
	req = httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1, got %d", rr.Code)
	}
}

func TestRouter_PartnerIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup memory repo and partner handler
	repo := partnerstorage.NewMemoryRepo()
	uc := partnerusecase.New(repo, logger)
	partnerHandler := partnerhttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, partnerHandler, nil, logger)

	// Test GET /api/v1/partners
	req := httptest.NewRequest(http.MethodGet, "/api/v1/partners", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/partners, got %d", rr.Code)
	}
}

func TestRouter_ProductIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup memory repo and product handler
	repo := productstorage.NewMemoryRepo()
	uc := productusecase.New(repo, logger)
	productHandler := producthttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, productHandler, logger)

	// Test GET /api/v1/products
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/products, got %d", rr.Code)
	}

	// Test GET /api/v1/uom
	req = httptest.NewRequest(http.MethodGet, "/api/v1/uom", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/uom, got %d", rr.Code)
	}
}
