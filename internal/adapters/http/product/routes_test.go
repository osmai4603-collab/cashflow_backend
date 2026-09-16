package producthttp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	producthttp "cashflow_backend/internal/adapters/http/product"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	productusecase "cashflow_backend/internal/usecase/product"

	"github.com/go-chi/chi/v5"
)

// setupAPIV1Server mounts the product routes under the same /api/v1 prefix used
// in production so route-pollution regressions (the variant closure capturing
// the parent router) are caught exactly where they broke before.
func setupAPIV1Server() *chi.Mux {
	repo := productstorage.NewMemoryRepo()
	uc := productusecase.New(repo, nil)
	h := producthttp.NewHandler(uc, nil)

	r := chi.NewRouter()
	r.Route("/api/v1", func(v1 chi.Router) {
		producthttp.RegisterRoutes(v1, h)
	})
	return r
}

func TestRoutes_UnknownSingleSegmentPathReturns404(t *testing.T) {
	r := setupAPIV1Server()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/account.budget", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched path, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestRoutes_DELETEUnexpectedSingleSegmentPathReturns404(t *testing.T) {
	r := setupAPIV1Server()

	// The old bug registered DELETE /{id} on the parent router, which would have
	// made any unknown single-segment path a destructive route. It must 404.
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/account.budget", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched destructive path, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestRoutes_VariantsScopedUnderVariantsPrefix(t *testing.T) {
	r := setupAPIV1Server()

	// 1. Create a product template.
	createBody, _ := json.Marshal(map[string]any{
		"name":       "Conference Table",
		"type":       "consu",
		"sale_price": 900.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating product, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// 2. Create a variant under the template.
	variantBody, _ := json.Marshal(map[string]any{
		"sku":         "TBL-MAPLE",
		"extra_price": 50.0,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/products/1/variants", bytes.NewReader(variantBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating variant, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	var variantResp struct {
		Data producthttp.VariantResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&variantResp); err != nil {
		t.Fatalf("failed to decode variant response: %v", err)
	}
	path := fmt.Sprintf("/api/v1/variants/%d", variantResp.Data.ID)

	// 3. Standalone variant route must resolve under /variants/{id}, not the
	// parent router as a bare /{id} catch-all.
	req = httptest.NewRequest(http.MethodGet, path, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 fetching variant, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// 4. DELETE must be scoped under /variants/{id}.
	req = httptest.NewRequest(http.MethodDelete, path, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 deleting variant, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}