package httpadapter_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/platform/response"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	crmusecase "cashflow_backend/internal/usecase/crm"
	hrusecase "cashflow_backend/internal/usecase/hr"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	paymentusecase "cashflow_backend/internal/usecase/payment"
	productusecase "cashflow_backend/internal/usecase/product"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
	saleusecase "cashflow_backend/internal/usecase/sale"
	stockusecase "cashflow_backend/internal/usecase/stock"
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
	return httpadapter.NewRouter(handler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
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
		t.Errorf("expected success: true on /")
	}

	// 2. Health /livez
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /livez, got %d", rr.Code)
	}

	// 3. Health /readyz
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /readyz, got %d", rr.Code)
	}

	// 4. API v1 Root
	req = httptest.NewRequest(http.MethodGet, "/api/v1/", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/, got %d", rr.Code)
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

	router := httpadapter.NewRouter(baseHandler, health, partnerHandler, nil, nil, nil, nil, nil, nil, nil, nil, logger)

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

	router := httpadapter.NewRouter(baseHandler, health, nil, productHandler, nil, nil, nil, nil, nil, nil, nil, logger)

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

func TestRouter_AccountingIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup memory repo and accounting handler
	repo := accountingstorage.NewMemoryRepo()
	uc := accountingusecase.New(repo, logger)
	accHandler := accountinghttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, accHandler, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/accounts
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/accounts, got %d", rr.Code)
	}

	// Test GET /api/v1/journals
	req = httptest.NewRequest(http.MethodGet, "/api/v1/journals", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/journals, got %d", rr.Code)
	}
}

func TestRouter_SaleIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup sale handler
	saleRepo := salestorage.NewMemoryRepo()
	saleUC := saleusecase.New(saleRepo, nil, nil, nil, nil, logger)
	saleHandler := salehttp.NewHandler(saleUC, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, saleHandler, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/sale-orders
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sale-orders", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/sale-orders, got %d", rr.Code)
	}
}

func TestRouter_PurchaseIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup purchase handler
	purchaseRepo := purchasestorage.NewMemoryRepo()
	purchaseUC := purchaseusecase.New(purchaseRepo, nil, nil, nil, nil, logger)
	purchaseHandler := purchasehttp.NewHandler(purchaseUC, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, purchaseHandler, nil, nil, nil, nil, logger)

	// Test GET /api/v1/purchase-orders
	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/purchase-orders, got %d", rr.Code)
	}
}

func TestRouter_StockIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup stock handler
	stockRepo := stockstorage.NewMemoryRepo()
	stockUC := stockusecase.New(stockRepo, nil, nil, nil, nil, logger)
	stockHandler := stockhttp.NewHandler(stockUC, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, stockHandler, nil, nil, nil, logger)

	// Test GET /api/v1/warehouses
	req := httptest.NewRequest(http.MethodGet, "/api/v1/warehouses", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/warehouses, got %d", rr.Code)
	}

	// Test GET /api/v1/stock-locations
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stock-locations", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/stock-locations, got %d", rr.Code)
	}

	// Test GET /api/v1/stock-pickings
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stock-pickings", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/stock-pickings, got %d", rr.Code)
	}

	// Test GET /api/v1/stock/on-hand
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stock/on-hand", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/stock/on-hand, got %d", rr.Code)
	}
}

func TestRouter_CRMIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup CRM handler
	crmRepo := crmstorage.NewMemoryRepo()
	crmUC := crmusecase.New(crmRepo, nil, nil, logger)
	crmHandler := crmhttp.NewHandler(crmUC, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, crmHandler, nil, nil, logger)

	// Test GET /api/v1/leads
	req := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/leads, got %d", rr.Code)
	}

	// Test GET /api/v1/crm/pipeline
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/pipeline", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/crm/pipeline, got %d", rr.Code)
	}

	// Test GET /api/v1/crm/stats
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/stats", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/crm/stats, got %d", rr.Code)
	}

	// Test GET /api/v1/crm/stages
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/stages", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/crm/stages, got %d", rr.Code)
	}
}

func TestRouter_PaymentIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup Payment handler
	paymentRepo := paymentstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()
	accountingUC := accountingusecase.New(accountingRepo, logger)
	paymentUC := paymentusecase.New(paymentRepo, accountingUC, nil, logger)
	paymentHandler := paymenthttp.NewHandler(paymentUC, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, paymentHandler, nil, logger)

	// Test GET /api/v1/payments
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/payments, got %d", rr.Code)
	}

	// Test GET /api/v1/payments/receivable
	req = httptest.NewRequest(http.MethodGet, "/api/v1/payments/receivable", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/payments/receivable, got %d", rr.Code)
	}

	// Test GET /api/v1/payments/payable
	req = httptest.NewRequest(http.MethodGet, "/api/v1/payments/payable", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/payments/payable, got %d", rr.Code)
	}
}

func TestRouter_HRIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	// Setup HR handler
	hrRepo := hrstorage.NewMemoryRepo()
	hrUC := hrusecase.New(hrRepo, nil, logger)
	hrHandler := hrhttp.NewHandler(hrUC, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, hrHandler, logger)

	// Test GET /api/v1/departments
	req := httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/departments, got %d", rr.Code)
	}

	// Test GET /api/v1/jobs
	req = httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/jobs, got %d", rr.Code)
	}

	// Test GET /api/v1/employees
	req = httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/employees, got %d", rr.Code)
	}

	// Test GET /api/v1/leave-requests
	req = httptest.NewRequest(http.MethodGet, "/api/v1/leave-requests", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/leave-requests, got %d", rr.Code)
	}
}
