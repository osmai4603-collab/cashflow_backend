package crmhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	crmstorage "cashflow_backend/internal/adapters/storage/crm"
	crmusecase "cashflow_backend/internal/usecase/crm"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *Handler) {
	repo := crmstorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := crmusecase.New(repo, nil, nil, logger)
	h := NewHandler(uc, logger)

	r := chi.NewRouter()
	r.Route("/api/v1", func(v1 chi.Router) {
		RegisterRoutes(v1, h)
	})

	return r, h
}

func TestHandler_LeadEndToEnd(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create Lead
	createReq := CreateLeadRequest{
		Name:            "Big Enterprise Opportunity",
		Type:            "lead",
		ContactName:     "Bob Vance",
		PartnerName:     "Vance Refrigeration",
		EmailFrom:       "bob@vancerefrig.com",
		ExpectedRevenue: 75000.0,
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/leads", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    LeadResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Data.ID == 0 {
		t.Fatal("expected non-zero lead ID")
	}
	leadID := resp.Data.ID

	// 2. Get Lead
	req = httptest.NewRequest(http.MethodGet, "/api/v1/leads/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 3. List Leads
	req = httptest.NewRequest(http.MethodGet, "/api/v1/leads?type=lead", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 4. Convert Lead
	convertReq := ConvertLeadRequest{}
	cbody, _ := json.Marshal(convertReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/leads/1/convert", bytes.NewReader(cbody))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on convert, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Mark Won
	wonReq := MarkWonRequest{CreateSaleOrder: false}
	wbody, _ := json.Marshal(wonReq)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/leads/1/won", bytes.NewReader(wbody))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on mark won, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Pipeline View
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/pipeline", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on pipeline, got %d: %s", w.Code, w.Body.String())
	}

	// 7. Stats View
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/stats", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on stats, got %d: %s", w.Code, w.Body.String())
	}

	// 8. Delete Lead
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/leads/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 NoContent, got %d", w.Code)
	}

	_ = leadID
}

func TestHandler_StagesAndLostReasons(t *testing.T) {
	r, _ := setupTestServer()

	// List Stages
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crm/stages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// List Lost Reasons
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/lost-reasons", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// List Tags
	req = httptest.NewRequest(http.MethodGet, "/api/v1/crm/tags", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
