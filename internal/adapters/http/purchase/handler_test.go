package purchasehttp_test

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

	purchasehttp "cashflow_backend/internal/adapters/http/purchase"
	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/platform/response"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(t *testing.T) (chi.Router, int64, int64) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	purchaseRepo := purchasestorage.NewMemoryRepo()
	requisitionRepo := purchasestorage.NewMemoryRequisitionRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()

	accountingUC := accountingusecase.New(accountingRepo, logger)
	purchaseUC := purchaseusecase.New(purchaseRepo, partnerRepo, productRepo, accountingRepo, accountingUC, logger)
	requisitionUC := purchaseusecase.NewRequisitionUseCase(requisitionRepo, purchaseRepo)
	handler := purchasehttp.NewHandler(purchaseUC, logger, requisitionUC)

	// Seed vendor partner
	vendor := &partner.Partner{Name: "Acme Industrial Supplies", IsSupplier: true, Active: true}
	if err := partnerRepo.Create(ctx, vendor); err != nil {
		t.Fatalf("failed to seed vendor: %v", err)
	}

	// Seed product
	prod := &product.ProductTemplate{Name: "Hydraulic Pump", Type: product.ProductTypeGoods, CostPrice: 150.0, PurchaseOK: true, Active: true}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	r := chi.NewRouter()
	purchasehttp.RegisterRoutes(r, handler)
	return r, vendor.ID, prod.ID
}

func TestPurchaseHTTP_EndToEnd(t *testing.T) {
	r, vendorID, prodID := setupTestRouter(t)

	// 1. Create RFQ
	unitPrice := 150.0
	createReq := purchasehttp.CreatePurchaseOrderRequest{
		PartnerID: vendorID,
		Currency:  "USD",
		Note:      "Raw parts order",
		Lines: []purchasehttp.CreatePurchaseOrderLineRequest{
			{
				ProductID:  prodID,
				ProductQty: 5.0,
				UnitPrice:  &unitPrice,
				Discount:   5.0, // 5%
			},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/purchase-orders", bytes.NewReader(body))
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
	var createdOrder purchasehttp.PurchaseOrderResponse
	if err := json.Unmarshal(orderData, &createdOrder); err != nil {
		t.Fatalf("failed to unmarshal order response: %v", err)
	}

	if createdOrder.ID <= 0 {
		t.Fatalf("expected positive order ID, got %d", createdOrder.ID)
	}
	orderIDStr := strconv.FormatInt(createdOrder.ID, 10)

	// 2. Get RFQ
	req = httptest.NewRequest(http.MethodGet, "/purchase-orders/"+orderIDStr, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", w.Code)
	}

	// 3. ActionSend
	req = httptest.NewRequest(http.MethodPost, "/purchase-orders/"+orderIDStr+"/send", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /send, got %d", w.Code)
	}

	// 4. ConfirmOrder
	req = httptest.NewRequest(http.MethodPost, "/purchase-orders/"+orderIDStr+"/confirm", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /confirm, got %d", w.Code)
	}

	// 5. Create Vendor Bill
	billReq := purchasehttp.CreateBillFromOrderRequest{}
	billBody, _ := json.Marshal(billReq)
	req = httptest.NewRequest(http.MethodPost, "/purchase-orders/"+orderIDStr+"/bill", bytes.NewReader(billBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on /bill, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 6. GetOrderBills
	req = httptest.NewRequest(http.MethodGet, "/purchase-orders/"+orderIDStr+"/bills", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /bills, got %d", w.Code)
	}

	// 7. ListOrders
	req = httptest.NewRequest(http.MethodGet, "/purchase-orders?partner_id="+strconv.FormatInt(vendorID, 10), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /purchase-orders list, got %d", w.Code)
	}
}

func TestPurchaseRequisitionHTTP_EndToEnd(t *testing.T) {
	r, vendorID, prodID := setupTestRouter(t)

	createReq := purchasehttp.CreatePurchaseRequisitionRequest{
		Name:      "REQ-2026-0001",
		Type:      purchasehttp.RequisitionType("blanket_order"),
		VendorID:  &vendorID,
		UserID:    1,
		CurrencyID: 1,
		CompanyID: 1,
		Description: "Annual pump requisition",
		Lines: []purchasehttp.CreateRequisitionLineRequest{
			{
				ProductID:   prodID,
				ProductQty:  4,
				PriceUnit:   120,
				Description: "Hydraulic Pump",
			},
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/purchase-requisitions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on requisition create, got %d. Body: %s", w.Code, w.Body.String())
	}

	var env response.Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode requisition response: %v", err)
	}
	data, _ := json.Marshal(env.Data)
	var created purchasehttp.PurchaseRequisitionResponse
	if err := json.Unmarshal(data, &created); err != nil {
		t.Fatalf("failed to unmarshal requisition response: %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("expected positive requisition ID, got %d", created.ID)
	}

	req = httptest.NewRequest(http.MethodGet, "/purchase-requisitions/"+strconv.FormatInt(created.ID, 10), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on requisition GET, got %d. Body: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/purchase-requisitions/"+strconv.FormatInt(created.ID, 10)+"/confirm", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on requisition confirm, got %d. Body: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/purchase-requisitions/"+strconv.FormatInt(created.ID, 10)+"/create-po", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create PO from requisition, got %d. Body: %s", w.Code, w.Body.String())
	}
}
