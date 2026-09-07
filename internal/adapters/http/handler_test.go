package httpadapter_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "cashflow_backend/internal/adapters/http"
	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	attachmenthttp "cashflow_backend/internal/adapters/http/attachment"
	companyhttp "cashflow_backend/internal/adapters/http/company"
	crmhttp "cashflow_backend/internal/adapters/http/crm"
	currencyhttp "cashflow_backend/internal/adapters/http/currency"
	hrhttp "cashflow_backend/internal/adapters/http/hr"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	paymenthttp "cashflow_backend/internal/adapters/http/payment"
	producthttp "cashflow_backend/internal/adapters/http/product"
	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	salehttp "cashflow_backend/internal/adapters/http/sale"
	sequencehttp "cashflow_backend/internal/adapters/http/sequence"
	stockhttp "cashflow_backend/internal/adapters/http/stock"
	userhttp "cashflow_backend/internal/adapters/http/user"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	attachmentstorage "cashflow_backend/internal/adapters/storage/attachment"
	companystorage "cashflow_backend/internal/adapters/storage/company"
	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	currencystorage "cashflow_backend/internal/adapters/storage/currency"
	hrstorage "cashflow_backend/internal/adapters/storage/hr"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	sequencestorage "cashflow_backend/internal/adapters/storage/sequence"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	userstorage "cashflow_backend/internal/adapters/storage/user"
	"cashflow_backend/internal/platform/auth"
	"cashflow_backend/internal/platform/currency"
	"cashflow_backend/internal/platform/response"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	attachmentusecase "cashflow_backend/internal/usecase/attachment"
	companyusecase "cashflow_backend/internal/usecase/company"
	crmusecase "cashflow_backend/internal/usecase/crm"
	currencyusecase "cashflow_backend/internal/usecase/currency"
	hrusecase "cashflow_backend/internal/usecase/hr"
	partnerusecase "cashflow_backend/internal/usecase/partner"
	paymentusecase "cashflow_backend/internal/usecase/payment"
	productusecase "cashflow_backend/internal/usecase/product"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
	saleusecase "cashflow_backend/internal/usecase/sale"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"
	stockusecase "cashflow_backend/internal/usecase/stock"
	userusecase "cashflow_backend/internal/usecase/user"
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
	return httpadapter.NewRouter(handler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
}

func withAuth(req *http.Request, roles ...string) *http.Request {
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	secret := "odoo-go-insecure-dev-secret-key-change-in-production"
	token, err := auth.GenerateToken(1, 1, roles, secret, time.Hour)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestRouter_ProtectsBusinessEndpointsWithoutToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}
	repo := partnerstorage.NewMemoryRepo()
	uc := partnerusecase.New(repo, logger)
	partnerHandler := partnerhttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, partnerHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger, "test-secret-key")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/partners", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth token, got %d", rr.Code)
	}

	token, err := auth.GenerateToken(1, 1, []string{"user"}, "test-secret-key", time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/partners", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid auth token, got %d", rr.Code)
	}
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
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, partnerHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/partners
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/partners", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, productHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/products
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/products", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/products, got %d", rr.Code)
	}

	// Test GET /api/v1/uom
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/uom", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, accHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/accounts
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/accounts, got %d", rr.Code)
	}

	// Test GET /api/v1/journals
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/journals", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, saleHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/sale-orders
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/sale-orders", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, purchaseHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/purchase-orders
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, stockHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/warehouses
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/warehouses", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/warehouses, got %d", rr.Code)
	}

	// Test GET /api/v1/stock-locations
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/stock-locations", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/stock-locations, got %d", rr.Code)
	}

	// Test GET /api/v1/stock-pickings
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/stock-pickings", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/stock-pickings, got %d", rr.Code)
	}

	// Test GET /api/v1/stock/on-hand
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/stock/on-hand", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, crmHandler, nil, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/leads
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/leads, got %d", rr.Code)
	}

	// Test GET /api/v1/crm/pipeline
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/crm/pipeline", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/crm/pipeline, got %d", rr.Code)
	}

	// Test GET /api/v1/crm/stats
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/crm/stats", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/crm/stats, got %d", rr.Code)
	}

	// Test GET /api/v1/crm/stages
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/crm/stages", nil), "user")
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

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, paymentHandler, nil, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/payments
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/payments", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/payments, got %d", rr.Code)
	}

	// Test GET /api/v1/payments/receivable
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/payments/receivable", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/payments/receivable, got %d", rr.Code)
	}

	// Test GET /api/v1/payments/payable
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/payments/payable", nil), "user")
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

	hrHandler := hrhttp.NewHandler(hrUC, nil, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, hrHandler, nil, nil, nil, nil, nil, nil, logger)

	// Test GET /api/v1/departments
	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/departments", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/departments, got %d", rr.Code)
	}

	// Test GET /api/v1/jobs
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/jobs, got %d", rr.Code)
	}

	// Test GET /api/v1/employees
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/employees, got %d", rr.Code)
	}

	// Test GET /api/v1/leave-requests
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/leave-requests", nil), "user")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/leave-requests, got %d", rr.Code)
	}
}

func TestRouter_CompanyIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	repo := companystorage.NewMemoryRepo()
	uc := companyusecase.New(repo, logger)
	companyHandler := companyhttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, companyHandler, nil, nil, nil, nil, nil, logger)

	// POST /api/v1/companies
	body := `{"name":"Acme Corp","currency_id":3}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/api/v1/companies", strings.NewReader(body)), "admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on POST /api/v1/companies, got %d", rr.Code)
	}

	// GET /api/v1/companies/default
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/companies/default", nil), "admin")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/v1/companies/default, got %d", rr.Code)
	}

	// GET /api/v1/companies
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil), "admin")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on GET /api/v1/companies, got %d", rr.Code)
	}
}

func TestRouter_UserIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	userRepo := userstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	uc := userusecase.New(userRepo, partnerRepo, logger, "", 0)
	userHandler := userhttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, userHandler, nil, nil, nil, nil, logger)

	// POST /api/v1/users
	body := `{"login":"admin","name":"Admin","password":"admin123","partner_name":"Administrator","company_id":1}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(body)), "admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on POST /api/v1/users, got %d", rr.Code)
	}

	// POST /api/v1/users/login
	loginBody := `{"login":"admin","password":"admin123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on POST /api/v1/users/login, got %d", rr.Code)
	}

	// GET /api/v1/users
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil), "admin")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on GET /api/v1/users, got %d", rr.Code)
	}
}

func TestRouter_CurrencyIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	currencyRepo := currencystorage.NewMemoryRepo()
	rateRepo := currencystorage.NewMemoryRateRepo()
	converter := currency.NewConverter(rateRepo)
	uc := currencyusecase.New(currencyRepo, rateRepo, converter, logger)
	currencyHandler := currencyhttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, currencyHandler, nil, nil, nil, logger)

	// POST /api/v1/currencies (create USD)
	body := `{"name":"USD","full_name":"US Dollar","symbol":"$"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/api/v1/currencies", strings.NewReader(body)), "admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on POST /api/v1/currencies, got %d", rr.Code)
	}

	// GET /api/v1/currencies
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/currencies", nil), "admin")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on GET /api/v1/currencies, got %d", rr.Code)
	}
}

func TestRouter_SequenceIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	repo := sequencestorage.NewMemoryRepo()
	uc := sequenceusecase.New(repo, logger)
	sequenceHandler := sequencehttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sequenceHandler, nil, nil, logger)

	// POST /api/v1/sequences
	body := `{"name":"Invoice","code":"sale.invoice","prefix":"INV/","padding":5}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/api/v1/sequences", strings.NewReader(body)), "admin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on POST /api/v1/sequences, got %d", rr.Code)
	}

	// GET /api/v1/sequences
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/sequences", nil), "admin")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on GET /api/v1/sequences, got %d", rr.Code)
	}
}

func TestRouter_AttachmentIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}

	tmpDir := t.TempDir()
	attachmenthttp.UploadDir = tmpDir

	repo := attachmentstorage.NewMemoryRepo()
	uc := attachmentusecase.New(repo, logger)
	attachmentHandler := attachmenthttp.NewHandler(uc, logger)

	router := httpadapter.NewRouter(baseHandler, health, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, attachmentHandler, nil, logger)

	// POST /api/v1/attachments (multipart)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("name", "Report")
	_ = mw.WriteField("res_model", "sale.order")
	_ = mw.WriteField("res_id", "5")
	fileWriter, err := mw.CreateFormFile("file", "report.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("hello attachment")); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	_ = mw.Close()

	req := withAuth(httptest.NewRequest(http.MethodPost, "/api/v1/attachments", &buf), "admin")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on POST /api/v1/attachments, got %d", rr.Code)
	}

	// GET /api/v1/attachments?res_model=sale.order&res_id=5
	req = withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/attachments?res_model=sale.order&res_id=5", nil), "admin")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on GET /api/v1/attachments, got %d", rr.Code)
	}
}
