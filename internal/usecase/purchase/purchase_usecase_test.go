package purchaseusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/purchase"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
)

func setupTestEnvironment(t *testing.T) (*purchaseusecase.UseCase, *purchasestorage.MemoryRepo, *accountingusecase.UseCase, int64, int64) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	// 1. Repositories
	purchaseRepo := purchasestorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()

	// 2. Accounting UseCase
	accountingUC := accountingusecase.New(accountingRepo, logger)

	// 3. Purchase UseCase
	purchaseUC := purchaseusecase.New(purchaseRepo, partnerRepo, productRepo, accountingRepo, accountingUC, logger)

	// 4. Seed Partner (Vendor / Supplier)
	vendor := &partner.Partner{
		Name:       "Global Suppliers LLC",
		IsSupplier: true,
		Active:     true,
	}
	if err := partnerRepo.Create(ctx, vendor); err != nil {
		t.Fatalf("failed to seed vendor: %v", err)
	}

	// 5. Seed Product
	prod := &product.ProductTemplate{
		Name:       "Industrial Gear",
		Type:       product.ProductTypeGoods,
		SalePrice:  120.0,
		CostPrice:  60.0,
		PurchaseOK: true,
		Active:     true,
	}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	return purchaseUC, purchaseRepo, accountingUC, vendor.ID, prod.ID
}

func TestPurchaseUseCase_EndToEndFlow(t *testing.T) {
	ctx := context.Background()
	uc, _, _, vendorID, productID := setupTestEnvironment(t)

	// 1. Create RFQ without unit price (should fallback to product CostPrice: 60.0)
	order, err := uc.CreateOrder(ctx, purchaseusecase.CreatePurchaseOrderInput{
		PartnerID: vendorID,
		DateOrder: time.Now(),
		Lines: []purchaseusecase.CreatePurchaseOrderLineInput{
			{
				ProductID:  productID,
				ProductQty: 10.0,
				Discount:   10.0,       // 10% discount -> Untaxed: 10 * 60 * 0.9 = 540.0
				TaxIDs:     []int64{2}, // 15% Purchase VAT -> 81.0
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create RFQ: %v", err)
	}

	if order.State != purchase.OrderStateDraft {
		t.Errorf("expected draft state, got %s", order.State)
	}
	if order.AmountUntaxed != 540.0 {
		t.Errorf("expected untaxed amount 540.0, got %f", order.AmountUntaxed)
	}
	if order.AmountTax != 81.0 {
		t.Errorf("expected tax amount 81.0, got %f", order.AmountTax)
	}
	if order.AmountTotal != 621.0 {
		t.Errorf("expected total amount 621.0, got %f", order.AmountTotal)
	}

	// 2. ActionSend (RFQ Sent to Vendor)
	order, err = uc.ActionSend(ctx, order.ID)
	if err != nil {
		t.Fatalf("ActionSend failed: %v", err)
	}
	if order.State != purchase.OrderStateSent {
		t.Errorf("expected state sent, got %s", order.State)
	}

	// 3. ConfirmOrder (Becomes confirmed PO)
	order, err = uc.ConfirmOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("ConfirmOrder failed: %v", err)
	}
	if order.State != purchase.OrderStatePurchase {
		t.Errorf("expected state purchase, got %s", order.State)
	}
	if order.InvoiceStatus != purchase.InvoiceStatusToInvoice {
		t.Errorf("expected invoice_status to_invoice, got %s", order.InvoiceStatus)
	}

	// 4. Create Vendor Bill in Accounting (bill 5 out of 10 items)
	billMove, err := uc.CreateBillFromOrder(ctx, order.ID, purchaseusecase.CreateBillFromOrderInput{
		Date: time.Now(),
		Lines: []purchaseusecase.LineBillQuantityInput{
			{
				LineID:   order.Lines[0].ID,
				Quantity: 5.0,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateBillFromOrder failed: %v", err)
	}
	if billMove.MoveType != accounting.MoveTypeInInvoice {
		t.Errorf("expected move type in_invoice, got %s", billMove.MoveType)
	}
	if billMove.PartnerID == nil || *billMove.PartnerID != vendorID {
		t.Errorf("expected partner id %d, got %v", vendorID, billMove.PartnerID)
	}

	// Check order status after partial bill
	reloaded, err := uc.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to reload order: %v", err)
	}
	if reloaded.Lines[0].QtyInvoiced != 5.0 {
		t.Errorf("expected qty_invoiced 5.0, got %f", reloaded.Lines[0].QtyInvoiced)
	}
	if reloaded.InvoiceStatus != purchase.InvoiceStatusToInvoice {
		t.Errorf("expected invoice_status to_invoice after partial bill, got %s", reloaded.InvoiceStatus)
	}

	// 5. Bill remaining 5 items
	_, err = uc.CreateBillFromOrder(ctx, order.ID, purchaseusecase.CreateBillFromOrderInput{
		Date: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to bill remaining qty: %v", err)
	}

	reloaded, err = uc.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to reload order: %v", err)
	}
	if reloaded.Lines[0].QtyInvoiced != 10.0 {
		t.Errorf("expected qty_invoiced 10.0, got %f", reloaded.Lines[0].QtyInvoiced)
	}
	if reloaded.InvoiceStatus != purchase.InvoiceStatusInvoiced {
		t.Errorf("expected invoice_status invoiced, got %s", reloaded.InvoiceStatus)
	}

	// 6. GetOrderBills
	bills, err := uc.GetOrderBills(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrderBills failed: %v", err)
	}
	if len(bills) != 2 {
		t.Errorf("expected 2 linked bills, got %d", len(bills))
	}

	// 7. ActionDone (Lock completed order)
	order, err = uc.ActionDone(ctx, order.ID)
	if err != nil {
		t.Fatalf("ActionDone failed: %v", err)
	}
	if order.State != purchase.OrderStateDone {
		t.Errorf("expected state done, got %s", order.State)
	}
}

func TestPurchaseUseCase_PriceDifference(t *testing.T) {
	ctx := context.Background()
	uc, purchaseRepo, accountingUC, vendorID, productID := setupTestEnvironment(t)

	// Product CostPrice = 60.0 (receipt valued at 60). The PO line is negotiated at 80,
	// so after 4 units are received the vendor bill raises a positive price difference.
	unitPrice := 80.0
	order, err := uc.CreateOrder(ctx, purchaseusecase.CreatePurchaseOrderInput{
		PartnerID: vendorID,
		DateOrder: time.Now(),
		Lines: []purchaseusecase.CreatePurchaseOrderLineInput{
			{
				ProductID:  productID,
				ProductQty: 10.0,
				UnitPrice:  &unitPrice,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}
	if _, err := uc.ConfirmOrder(ctx, order.ID); err != nil {
		t.Fatalf("ConfirmOrder failed: %v", err)
	}

	// Simulate that 4 units were already received (valued at cost 60.0).
	reloaded, err := purchaseRepo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to load order: %v", err)
	}
	reloaded.Lines[0].QtyReceived = 4.0
	if err := purchaseRepo.UpdateOrder(ctx, reloaded); err != nil {
		t.Fatalf("failed to persist received qty: %v", err)
	}

	// Bill 10 units at 80 → of which 4 cover already-received stock → diff = 4 * (80-60) = 80.
	bill, err := uc.CreateBillFromOrder(ctx, order.ID, purchaseusecase.CreateBillFromOrderInput{Date: time.Now()})
	if err != nil {
		t.Fatalf("CreateBillFromOrder failed: %v", err)
	}
	if bill.MoveType != accounting.MoveTypeInInvoice {
		t.Fatalf("expected vendor bill, got %s", bill.MoveType)
	}

	// The bill itself books expense 800 (10 * 80) but no stock/inventory side for the paid lines.
	if bill.AmountUntaxed != 800.0 {
		t.Fatalf("expected untaxed 800.0, got %.2f", bill.AmountUntaxed)
	}

	// The price-difference entry must exist, be balanced, penetrate the Inventory (5) and
	// Price Difference (17) accounts, and sum to 80 for the received portion.
	page := pagination.PageRequest{Page: 1, Limit: 100}
	res, err := accountingUC.ListMoves(ctx, filter.NewFilter(), page)
	if err != nil {
		t.Fatalf("failed to list moves: %v", err)
	}
	var foundDiff bool
	diffTotal := 0.0
	for _, m := range res.Items {
		full, err := accountingUC.GetMove(ctx, m.ID)
		if err != nil {
			t.Fatalf("failed to load move %d: %v", m.ID, err)
		}
		if err := full.ValidateBalance(); err != nil {
			t.Fatalf("move %d must be balanced: %v", full.ID, err)
		}
		for _, l := range full.Lines {
			if l.AccountID != 17 && l.AccountID != 5 {
				continue
			}
			foundDiff = true
			diffTotal += l.Debit - l.Credit
		}
	}
	if !foundDiff {
		t.Fatal("expected a price-difference entry touching accounts 5 and 17")
	}
	if diffTotal != 80.0 {
		t.Fatalf("expected net stock movement of 80.0, got %.2f", diffTotal)
	}
}

func TestPurchaseUseCase_CancelAndDraft(t *testing.T) {
	ctx := context.Background()
	uc, _, _, vendorID, productID := setupTestEnvironment(t)

	order, err := uc.CreateOrder(ctx, purchaseusecase.CreatePurchaseOrderInput{
		PartnerID: vendorID,
		DateOrder: time.Now(),
		Lines: []purchaseusecase.CreatePurchaseOrderLineInput{
			{
				ProductID:  productID,
				ProductQty: 2.0,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	// Cancel draft
	order, err = uc.CancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}
	if order.State != purchase.OrderStateCancel {
		t.Errorf("expected state cancel, got %s", order.State)
	}

	// Reset to draft
	order, err = uc.ResetToDraft(ctx, order.ID)
	if err != nil {
		t.Fatalf("ResetToDraft failed: %v", err)
	}
	if order.State != purchase.OrderStateDraft {
		t.Errorf("expected state draft, got %s", order.State)
	}
}
