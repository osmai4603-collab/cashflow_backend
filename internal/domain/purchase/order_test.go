package purchase_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/purchase"
)

func TestPurchaseOrderLine_ComputeAmounts(t *testing.T) {
	line := purchase.PurchaseOrderLine{
		ProductID:  1,
		Name:       "Raw Steel Plates",
		ProductQty: 10,
		UnitPrice:  100,
		Discount:   10, // 10% discount -> Untaxed: 900
	}

	line.ComputeAmounts([]float64{15.0}) // 15% VAT -> 135

	if line.PriceSubtotal != 900.0 {
		t.Fatalf("expected subtotal 900.0, got %f", line.PriceSubtotal)
	}
	if line.PriceTax != 135.0 {
		t.Fatalf("expected tax 135.0, got %f", line.PriceTax)
	}
	if line.PriceTotal != 1035.0 {
		t.Fatalf("expected total 1035.0, got %f", line.PriceTotal)
	}
}

func TestPurchaseOrder_Validation(t *testing.T) {
	// 1. Missing partner
	po := &purchase.PurchaseOrder{
		DateOrder: time.Now(),
	}
	if err := po.Validate(); err == nil {
		t.Fatalf("expected error on missing partner")
	}

	// 2. Valid minimal order
	po.PartnerID = 5
	if err := po.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if po.Currency != "USD" {
		t.Fatalf("expected default currency USD, got %s", po.Currency)
	}
	if po.State != purchase.OrderStateDraft {
		t.Fatalf("expected default state draft, got %s", po.State)
	}

	// 3. Line with zero quantity
	po.Lines = []purchase.PurchaseOrderLine{
		{
			ProductID:  1,
			Name:       "Items",
			ProductQty: 0,
			UnitPrice:  10,
		},
	}
	if err := po.Validate(); err == nil {
		t.Fatalf("expected error on zero product quantity")
	}
}

func TestPurchaseOrder_Lifecycle(t *testing.T) {
	po := &purchase.PurchaseOrder{
		PartnerID: 5,
		DateOrder: time.Now(),
		State:     purchase.OrderStateDraft,
		Lines: []purchase.PurchaseOrderLine{
			{
				ID:         1,
				ProductID:  10,
				Name:       "Component A",
				ProductQty: 5,
				UnitPrice:  20,
			},
		},
	}
	po.RecomputeTotals()

	// 1. ActionSend
	if err := po.ActionSend(); err != nil {
		t.Fatalf("ActionSend failed: %v", err)
	}
	if po.State != purchase.OrderStateSent {
		t.Fatalf("expected state sent, got %s", po.State)
	}

	// 2. ActionConfirm
	if err := po.ActionConfirm("PO/2026/00001"); err != nil {
		t.Fatalf("ActionConfirm failed: %v", err)
	}
	if po.State != purchase.OrderStatePurchase {
		t.Fatalf("expected state purchase, got %s", po.State)
	}
	if po.Name != "PO/2026/00001" {
		t.Fatalf("expected name PO/2026/00001, got %s", po.Name)
	}
	if po.InvoiceStatus != purchase.InvoiceStatusToInvoice {
		t.Fatalf("expected invoice_status to_invoice, got %s", po.InvoiceStatus)
	}

	// 3. Cannot cancel if partially billed
	po.Lines[0].QtyInvoiced = 2
	po.UpdateBillStatus()
	if po.InvoiceStatus != purchase.InvoiceStatusToInvoice {
		t.Fatalf("expected status to_invoice, got %s", po.InvoiceStatus)
	}
	if err := po.ActionCancel(); err == nil {
		t.Fatalf("expected error when cancelling billed order")
	}

	// 4. Fully billed
	po.Lines[0].QtyInvoiced = 5
	po.UpdateBillStatus()
	if po.InvoiceStatus != purchase.InvoiceStatusInvoiced {
		t.Fatalf("expected status invoiced, got %s", po.InvoiceStatus)
	}

	// 5. ActionDone
	if err := po.ActionDone(); err != nil {
		t.Fatalf("ActionDone failed: %v", err)
	}
	if po.State != purchase.OrderStateDone {
		t.Fatalf("expected state done, got %s", po.State)
	}

	// Cannot cancel done order
	if err := po.ActionCancel(); err == nil {
		t.Fatalf("expected error when cancelling done order")
	}
}

func TestPurchaseOrder_ResetToDraft(t *testing.T) {
	po := &purchase.PurchaseOrder{
		PartnerID: 5,
		DateOrder: time.Now(),
		State:     purchase.OrderStateCancel,
	}

	if err := po.ActionDraft(); err != nil {
		t.Fatalf("ActionDraft failed: %v", err)
	}
	if po.State != purchase.OrderStateDraft {
		t.Fatalf("expected state draft, got %s", po.State)
	}
}
