package httpadapter_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
	"cashflow_backend/internal/adapters/storage"
	"cashflow_backend/internal/usecase"
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

func setupTestServer() (http.Handler, *storage.MemoryTransactionRepo) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := storage.NewMemoryTransactionRepo()
	uc := usecase.NewTransactionUseCase(repo)
	handler := httpadapter.NewTransactionHandler(uc, logger)
	health := &mockHealthRoutes{}
	router := httpadapter.NewRouter(handler, health, logger)
	return router, repo
}

func TestRouter_Endpoints(t *testing.T) {
	router, _ := setupTestServer()

	// 1. Root
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /, got %d", rr.Code)
	}

	// 2. Health routes
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 on /livez, got %d", rr.Code)
	}

	// 3. Create Transaction
	body := []byte(`{"amount": 120.50, "type": "income", "description": "Salary"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var created httpadapter.TransactionResponse
	if err := json.NewDecoder(rr.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.Amount != 120.50 || created.Type != "income" {
		t.Errorf("unexpected created transaction: %+v", created)
	}

	// 4. Get by ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/transactions/"+created.ID, nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	// 5. Summary
	req = httptest.NewRequest(http.MethodGet, "/api/v1/transactions/summary", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var summary httpadapter.SummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&summary); err != nil {
		t.Fatalf("failed to decode summary: %v", err)
	}
	if summary.TotalIncome != 120.50 || summary.Count != 1 {
		t.Errorf("unexpected summary: %+v", summary)
	}
}

func TestRouter_ValidationErrors(t *testing.T) {
	router, _ := setupTestServer()

	// Invalid amount
	body := []byte(`{"amount": -50, "type": "expense", "description": "Coffee"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rr.Code)
	}

	// Not found
	req = httptest.NewRequest(http.MethodGet, "/api/v1/transactions/non-existent-id", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", rr.Code)
	}
}
