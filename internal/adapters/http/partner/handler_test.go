package partnerhttp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/platform/response"
	partnerusecase "cashflow_backend/internal/usecase/partner"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter() (chi.Router, *partnerstorage.MemoryRepo) {
	repo := partnerstorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := partnerusecase.New(repo, logger)
	h := partnerhttp.NewHandler(uc, logger)

	r := chi.NewRouter()
	r.Route("/api/v1", func(v1 chi.Router) {
		partnerhttp.RegisterRoutes(v1, h)
	})

	return r, repo
}

func TestPartnerHTTP_Create(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Valid partner creation
	payload := []byte(`{
		"name": "Al-Amal Trading Co.",
		"type": "company",
		"email": "info@al-amal.com",
		"phone": "+966500000000",
		"is_customer": true,
		"city": "Riyadh",
		"country": "Saudi Arabia"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/partners", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var env response.Envelope
	if err := json.NewDecoder(rr.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !env.Success {
		t.Errorf("expected success=true in envelope")
	}

	// 2. Invalid body (empty name)
	badPayload := []byte(`{"name": ""}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/partners", bytes.NewReader(badPayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}
}

func TestPartnerHTTP_Endpoints(t *testing.T) {
	r, repo := setupTestRouter()

	// Seed partners
	p1 := &partner.Partner{
		Name:       "Customer Corp",
		Type:       partner.PartnerTypeCompany,
		IsCustomer: true,
		IsSupplier: false,
		City:       "Riyadh",
	}
	p2 := &partner.Partner{
		Name:       "Supplier Tech",
		Type:       partner.PartnerTypeCompany,
		IsCustomer: false,
		IsSupplier: true,
		City:       "Jeddah",
	}
	_ = repo.Create(t.Context(), p1)
	_ = repo.Create(t.Context(), p2)

	// 1. GET /api/v1/partners
	req := httptest.NewRequest(http.MethodGet, "/api/v1/partners?page=1&limit=10", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on list, got %d", rr.Code)
	}

	// 2. GET /api/v1/partners/customers
	req = httptest.NewRequest(http.MethodGet, "/api/v1/partners/customers", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on customers, got %d", rr.Code)
	}

	// 3. GET /api/v1/partners/suppliers
	req = httptest.NewRequest(http.MethodGet, "/api/v1/partners/suppliers", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on suppliers, got %d", rr.Code)
	}

	// 4. GET /api/v1/partners/{id}
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/partners/%d", p1.ID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on get by id, got %d", rr.Code)
	}

	// 5. PUT /api/v1/partners/{id}
	updatePayload := []byte(`{"city": "Dammam"}`)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/partners/%d", p1.ID), bytes.NewReader(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on update, got %d", rr.Code)
	}

	// 6. DELETE /api/v1/partners/{id}
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/partners/%d", p1.ID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 on delete, got %d", rr.Code)
	}

	// 7. GET deleted partner -> 404
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/partners/%d", p1.ID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}
}
