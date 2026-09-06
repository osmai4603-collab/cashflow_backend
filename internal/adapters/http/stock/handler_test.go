package stockhttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	stockhttp "cashflow_backend/internal/adapters/http/stock"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/stock"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func setupTestServer() (http.Handler, *stockstorage.MemoryRepo) {
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil)
	h := stockhttp.NewHandler(uc, nil)

	r := chi.NewRouter()
	r.Route("/api/v1", func(v1 chi.Router) {
		stockhttp.RegisterRoutes(v1, h)
	})
	return r, repo
}

func TestStockLocationsAPI(t *testing.T) {
	handler, _ := setupTestServer()

	// 1. List pre-seeded locations
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stock-locations", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Create child location
	parentID := int64(8)
	body, _ := json.Marshal(stockhttp.CreateLocationRequest{
		Name:     "Rack B",
		Usage:    stock.LocationUsageInternal,
		ParentID: &parentID,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stock-locations", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWarehousesAPI(t *testing.T) {
	handler, _ := setupTestServer()

	// 1. List warehouses
	req := httptest.NewRequest(http.MethodGet, "/api/v1/warehouses", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// 2. Create warehouse
	body, _ := json.Marshal(stockhttp.CreateWarehouseRequest{
		Name:       "Regional Depot",
		Code:       "REG",
		LotStockID: 8,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/warehouses", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPickingsAndStockFlowAPI(t *testing.T) {
	handler, _ := setupTestServer()

	// 1. Create Receipt Picking (incoming)
	body, _ := json.Marshal(stockhttp.CreatePickingRequest{
		PickingType: stock.PickingTypeIncoming,
		Origin:      "PO/2026/00099",
		Moves: []stockhttp.CreateMoveRequest{
			{
				ProductID:  88,
				ProductQty: 50,
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stock-pickings", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Validate Receipt Picking
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stock-pickings/1/validate", bytes.NewReader([]byte("{}")))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on validate, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Check On Hand Stock
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stock/on-hand", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /stock/on-hand, got %d", rec.Code)
	}

	// 4. Adjust Stock via /stock/adjust
	adjBody, _ := json.Marshal(stockhttp.StockAdjustmentRequest{
		ProductID:   88,
		LocationID:  8,
		NewQuantity: 75,
		Note:        "Found 25 additional units during count",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/stock/adjust", bytes.NewReader(adjBody))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /stock/adjust, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Query Moves
	req = httptest.NewRequest(http.MethodGet, "/api/v1/stock/moves", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on /stock/moves, got %d", rec.Code)
	}
}
