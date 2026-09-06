package accountinghttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountinghttp "cashflow_backend/internal/adapters/http/accounting"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	"cashflow_backend/internal/domain/accounting"
	accountingusecase "cashflow_backend/internal/usecase/accounting"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *accountingstorage.MemoryRepo) {
	repo := accountingstorage.NewMemoryRepo()
	uc := accountingusecase.New(repo, nil)
	h := accountinghttp.NewHandler(uc, nil)

	r := chi.NewRouter()
	accountinghttp.RegisterRoutes(r, h)
	return r, repo
}

func TestHandler_AccountEndpoints(t *testing.T) {
	r, _ := setupTestServer()

	// 1. List accounts
	req := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// 2. Create account
	accBody := map[string]any{
		"code": "108000",
		"name": "Treasury Account",
		"type": "asset_cash",
	}
	bodyBytes, _ := json.Marshal(accBody)
	req = httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_InvoiceFlowAndReports(t *testing.T) {
	r, _ := setupTestServer()

	// 1. Create Invoice via /invoices
	invoiceReq := map[string]any{
		"move_type":  "out_invoice",
		"partner_id": 1,
		"journal_id": 1,
		"ref":        "INV-2026-TEST",
		"items": []map[string]any{
			{
				"name":       "ERP Implementation Consulting",
				"quantity":   5.0,
				"price_unit": 200.0,
				"tax_ids":    []int64{1}, // 15% Sales VAT
			},
		},
	}
	bodyBytes, _ := json.Marshal(invoiceReq)
	req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var created struct {
		Data accountinghttp.MoveResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode invoice response: %v", err)
	}

	invoiceID := created.Data.ID
	if invoiceID == 0 {
		t.Fatal("expected non-zero invoice ID")
	}
	if created.Data.AmountUntaxed != 1000.0 {
		t.Errorf("expected untaxed 1000.0, got %.2f", created.Data.AmountUntaxed)
	}
	if created.Data.AmountTax != 150.0 {
		t.Errorf("expected tax 150.0, got %.2f", created.Data.AmountTax)
	}
	if created.Data.AmountTotal != 1150.0 {
		t.Errorf("expected total 1150.0, got %.2f", created.Data.AmountTotal)
	}

	// 2. Post Invoice via /moves/{id}/post
	req = httptest.NewRequest(http.MethodPost, "/moves/"+strconvFormat(invoiceID)+"/post", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var posted struct {
		Data accountinghttp.MoveResponse `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&posted)
	if posted.Data.State != accounting.MoveStatePosted {
		t.Errorf("expected state posted, got %s", posted.Data.State)
	}
	if posted.Data.Name != "INV/2026/00001" {
		t.Errorf("expected sequence name INV/2026/00001, got %s", posted.Data.Name)
	}

	// 3. Test Trial Balance Report
	req = httptest.NewRequest(http.MethodGet, "/accounting/reports/trial-balance", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on trial balance, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var tbResp struct {
		Data accounting.TrialBalanceReport `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&tbResp)
	if !tbResp.Data.IsBalanced {
		t.Errorf("trial balance must be balanced, difference: %.4f", tbResp.Data.Difference)
	}

	// 4. Test Profit & Loss Report
	req = httptest.NewRequest(http.MethodGet, "/accounting/reports/profit-loss", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on profit-loss, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// 5. Reverse Invoice
	revReq := map[string]any{
		"reversal_date": time.Now().Format("2006-01-02"),
		"ref":           "Return of services",
	}
	revBytes, _ := json.Marshal(revReq)
	req = httptest.NewRequest(http.MethodPost, "/moves/"+strconvFormat(invoiceID)+"/reverse", bytes.NewReader(revBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on reversal, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func strconvFormat(id int64) string {
	var buf [20]byte
	i := len(buf)
	for id >= 10 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	i--
	buf[i] = byte('0' + id)
	return string(buf[i:])
}
