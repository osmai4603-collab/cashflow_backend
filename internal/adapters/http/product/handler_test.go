package producthttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	producthttp "cashflow_backend/internal/adapters/http/product"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	productusecase "cashflow_backend/internal/usecase/product"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *productstorage.MemoryRepo) {
	repo := productstorage.NewMemoryRepo()
	uc := productusecase.New(repo, nil)
	h := producthttp.NewHandler(uc, nil)

	r := chi.NewRouter()
	producthttp.RegisterRoutes(r, h)
	return r, repo
}

func TestHandler_ProductsCRUD(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create Product
	createBody := map[string]any{
		"name":         "Ergonomic Chair",
		"type":         "consu",
		"sale_price":   350.0,
		"cost_price":   200.0,
		"internal_ref": "CHR-01",
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var createResp struct {
		Data producthttp.ProductResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if createResp.Data.ID <= 0 || createResp.Data.Name != "Ergonomic Chair" {
		t.Fatalf("unexpected product response: %+v", createResp.Data)
	}

	productID := createResp.Data.ID

	// 2. Get Product by ID
	req = httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// 3. Update Product
	updateBody := map[string]any{
		"name":       "Ergonomic Chair Pro",
		"sale_price": 380.0,
	}
	updateBytes, _ := json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/products/1", bytes.NewReader(updateBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// 4. List Products
	req = httptest.NewRequest(http.MethodGet, "/products?page=1&limit=10", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// 5. Create Variant for this product
	variantBody := map[string]any{
		"sku":         "CHR-01-BLACK",
		"extra_price": 20.0,
	}
	vBytes, _ := json.Marshal(variantBody)
	req = httptest.NewRequest(http.MethodPost, "/products/1/variants", bytes.NewReader(vBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 creating variant, got %d", rec.Code)
	}

	// 6. Get Variants
	req = httptest.NewRequest(http.MethodGet, "/products/1/variants", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 getting variants, got %d", rec.Code)
	}

	// 7. Delete Product (Soft delete)
	req = httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	// 8. Verify deleted product gives 404
	req = httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for deleted product, got %d", rec.Code)
	}

	_ = productID
}

func TestHandler_CategoriesAndUoMs(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create Category
	catBody := map[string]any{
		"name": "Furniture",
	}
	bodyBytes, _ := json.Marshal(catBody)
	req := httptest.NewRequest(http.MethodPost, "/product-categories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	// 2. List Categories
	req = httptest.NewRequest(http.MethodGet, "/product-categories", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// 3. Create UoM
	uomBody := map[string]any{
		"name":     "Pallet",
		"category": "unit",
		"ratio":    50.0,
		"rounding": 1.0,
	}
	uomBytes, _ := json.Marshal(uomBody)
	req = httptest.NewRequest(http.MethodPost, "/uom", bytes.NewReader(uomBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	// 4. List UoMs
	req = httptest.NewRequest(http.MethodGet, "/uom", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestHandler_PricelistComputePrice(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create Product
	pBody, _ := json.Marshal(map[string]any{
		"name":       "Office Desk",
		"sale_price": 500.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(pBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var pResp struct {
		Data producthttp.ProductResponse `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&pResp)
	pID := pResp.Data.ID

	// 2. Add rule to Public Pricelist (id=1): 20% discount on 2+ units
	itemBody, _ := json.Marshal(map[string]any{
		"applied_on":    "template",
		"template_id":   pID,
		"min_quantity":  2.0,
		"compute_price": "percentage",
		"percent_price": 20.0,
	})
	req = httptest.NewRequest(http.MethodPost, "/pricelists/1/items", bytes.NewReader(itemBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	// 3. Compute Price for 3 units: (500 - 20% = 400 unit price, 1200 total)
	computeBody, _ := json.Marshal(map[string]any{
		"product_id": pID,
		"quantity":   3.0,
	})
	req = httptest.NewRequest(http.MethodPost, "/pricelists/1/compute-price", bytes.NewReader(computeBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var compResp struct {
		Data producthttp.ComputePriceResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&compResp); err != nil {
		t.Fatalf("failed to decode compute response: %v", err)
	}

	if compResp.Data.UnitPrice != 400.0 || compResp.Data.TotalPrice != 1200.0 {
		t.Fatalf("unexpected compute price: %+v", compResp.Data)
	}
}
