package salehttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	salehttp "cashflow_backend/internal/adapters/http/sale"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/platform/response"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	saleusecase "cashflow_backend/internal/usecase/sale"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(t *testing.T) (chi.Router, int64, int64) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	saleRepo := salestorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()

	accountingUC := accountingusecase.New(accountingRepo, logger)
	saleUC := saleusecase.New(saleRepo, partnerRepo, productRepo, accountingRepo, accountingUC, logger)
	handler := salehttp.NewHandler(saleUC, logger)

	// Seed partner
	cust := &partner.Partner{Name: "Global Tech", IsCustomer: true, Active: true}
	if err := partnerRepo.Create(ctx, cust); err != nil {
		t.Fatalf("failed to seed customer: %v", err)
	}

	// Seed product
	prod := &product.ProductTemplate{Name: "Standing Desk", Type: product.ProductTypeGoods, SalePrice: 300.0, Active: true}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	r := chi.NewRouter()
	salehttp.RegisterRoutes(r, handler)
	return r, cust.ID, prod.ID
}

func TestSaleHTTP_EndToEnd(t *testing.T) {
	r, custID, prodID := setupTestRouter(t)

	// 1. Create Quotation
	unitPrice := 300.0
	createReq := salehttp.CreateSaleOrderRequest{
		PartnerID: custID,
		Currency:  "USD",
		Note:      "Delivery by next week",
		Lines: []salehttp.CreateSaleOrderLineRequest{
			{
				ProductID:     prodID,
				ProductUomQty: 2.0,
				UnitPrice:     &unitPrice,
				Discount:      5.0, // 5%
			},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/sale-orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	orderData, _ := json.Marshal(env.Data)
	var createdOrder salehttp.SaleOrderResponse
	if err := json.Unmarshal(orderData, &createdOrder); err != nil {
		t.Fatalf("failed to unmarshal order response: %v", err)
	}

	if createdOrder.ID <= 0 {
		t.Fatalf("expected positive order ID, got %d", createdOrder.ID)
	}
	if createdOrder.State != "draft" {
		t.Errorf("expected draft state, got %s", createdOrder.State)
	}

	// 2. Get Order
	req = httptest.NewRequest(http.MethodGet, "/sale-orders/"+strconvFormat(createdOrder.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on get order, got %d", w.Code)
	}

	// 3. List Orders
	req = httptest.NewRequest(http.MethodGet, "/sale-orders?limit=10", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on list orders, got %d", w.Code)
	}

	// 4. Send Quotation
	req = httptest.NewRequest(http.MethodPost, "/sale-orders/"+strconvFormat(createdOrder.ID)+"/send", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on send order, got %d", w.Code)
	}

	// 5. Confirm Order
	req = httptest.NewRequest(http.MethodPost, "/sale-orders/"+strconvFormat(createdOrder.ID)+"/confirm", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on confirm order, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 6. Create Invoice from Order
	invReq := salehttp.CreateInvoiceFromOrderRequest{
		Date: timePtr(time.Now()),
	}
	invBody, _ := json.Marshal(invReq)
	req = httptest.NewRequest(http.MethodPost, "/sale-orders/"+strconvFormat(createdOrder.ID)+"/invoice", bytes.NewReader(invBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on invoice creation, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 7. Get Order Invoices
	req = httptest.NewRequest(http.MethodGet, "/sale-orders/"+strconvFormat(createdOrder.ID)+"/invoices", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on get order invoices, got %d", w.Code)
	}
}

func strconvFormat(id int64) string {
	return strconv.FormatInt(id, 10)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
