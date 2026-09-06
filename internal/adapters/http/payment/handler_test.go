package paymenthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	paymentstorage "cashflow_backend/internal/adapters/storage/payment"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/payment"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	paymentusecase "cashflow_backend/internal/usecase/payment"

	"github.com/go-chi/chi/v5"
)

func setupTestServer() (*chi.Mux, *Handler, *paymentusecase.UseCase, *accountingusecase.UseCase, *accountingstorage.MemoryRepo) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	paymentRepo := paymentstorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()

	// Seed partner
	_ = partnerRepo.Create(context.Background(), &partner.Partner{
		ID:         1,
		Name:       "Test Customer",
		IsCustomer: true,
		Active:     true,
	})

	accountingUC := accountingusecase.New(accountingRepo, logger)
	paymentUC := paymentusecase.New(paymentRepo, accountingUC, partnerRepo, logger)
	h := NewHandler(paymentUC, logger)

	r := chi.NewRouter()
	r.Route("/api/v1", func(v1 chi.Router) {
		RegisterRoutes(v1, h)
	})

	return r, h, paymentUC, accountingUC, accountingRepo
}

func TestHandler_EndToEnd(t *testing.T) {
	r, _, _, accountingUC, _ := setupTestServer()
	ctx := context.Background()

	// 1. Create a customer invoice in accounting to reconcile with
	invDueDate := time.Now().Add(15 * 24 * time.Hour)
	inv, err := accountingUC.CreateInvoice(ctx, accountingusecase.CreateInvoiceInput{
		MoveType:      accounting.MoveTypeOutInvoice,
		PartnerID:     1,
		JournalID:     1, // Customer Invoices
		Date:          time.Now(),
		InvoiceDate:   &invDueDate,
		PaymentTermID: nil,
		Currency:      "USD",
		Ref:           "INV/2026/001",
		Items: []accountingusecase.InvoiceLineItemInput{
			{
				Name:      "Consulting Service",
				Quantity:  1,
				PriceUnit: 1000.0,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create invoice: %v", err)
	}

	// Post the invoice
	postedInv, err := accountingUC.PostMove(ctx, inv.ID)
	if err != nil {
		t.Fatalf("failed to post invoice: %v", err)
	}

	// 2. POST /api/v1/payments (Create draft payment)
	now := time.Now().UTC()
	createReq := CreatePaymentRequest{
		PartnerID:     1,
		Amount:        600.0,
		PaymentType:   payment.PaymentTypeInbound,
		PaymentMethod: payment.PaymentMethodBankTransfer,
		JournalID:     3, // Bank
		Date:          &now,
		Ref:           "Customer Partial Payment",
		InvoiceIDs:    []int64{postedInv.ID},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Success bool            `json:"success"`
		Data    PaymentResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	paymentID := createResp.Data.ID
	if paymentID == 0 {
		t.Fatalf("expected non-zero payment ID")
	}

	// 3. GET /api/v1/payments/{id}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/payments/1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}

	// 4. POST /api/v1/payments/{id}/post (Post payment with auto-reconcile)
	postBody, _ := json.Marshal(PostPaymentRequest{AutoReconcile: true})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/payments/1/post", bytes.NewReader(postBody))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var postResp struct {
		Success bool            `json:"success"`
		Data    PaymentResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &postResp)
	if postResp.Data.State != payment.PaymentStateReconciled {
		t.Errorf("expected payment state reconciled, got %s", postResp.Data.State)
	}
	if postResp.Data.MoveID == nil {
		t.Errorf("expected linked MoveID")
	}

	// 5. GET /api/v1/payments (List)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/payments", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on list, got %d", w.Code)
	}

	// 6. GET /api/v1/payments/receivable (Aged Receivable)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/payments/receivable", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on receivable aging, got %d: %s", w.Code, w.Body.String())
	}

	var agingResp struct {
		Success bool                `json:"success"`
		Data    AgingReportResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &agingResp); err != nil {
		t.Fatalf("failed to decode aging report: %v", err)
	}
	// The invoice had 1000 total, 600 paid by payment, so residual is 400
	if agingResp.Data.Total.Total != 400.0 {
		t.Errorf("expected aging total 400.0, got %.2f", agingResp.Data.Total.Total)
	}

	// 7. GET /api/v1/payments/payable (Aged Payable)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/payments/payable", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on payable aging, got %d: %s", w.Code, w.Body.String())
	}

	// 8. POST /api/v1/payments/{id}/cancel
	req = httptest.NewRequest(http.MethodPost, "/api/v1/payments/1/cancel", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on cancel, got %d: %s", w.Code, w.Body.String())
	}
}
