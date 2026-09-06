package sale_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/sale"
)

func TestSaleOrderLine_ComputeAmounts(t *testing.T) {
	line := sale.SaleOrderLine{
		ProductID:     1,
		Name:          "Desk Combination",
		ProductUomQty: 2.0,
		UnitPrice:     100.0,
		Discount:      10.0, // 10%
	}

	// 15% VAT
	line.ComputeAmounts([]float64{15.0})

	// Untaxed = 2 * 100 * 0.9 = 180
	if line.PriceSubtotal != 180.0 {
		t.Errorf("expected PriceSubtotal 180.0, got %f", line.PriceSubtotal)
	}

	// Tax = 180 * 0.15 = 27.0
	if line.PriceTax != 27.0 {
		t.Errorf("expected PriceTax 27.0, got %f", line.PriceTax)
	}

	// Total = 180 + 27 = 207.0
	if line.PriceTotal != 207.0 {
		t.Errorf("expected PriceTotal 207.0, got %f", line.PriceTotal)
	}
}

func TestSaleOrder_RecomputeTotals(t *testing.T) {
	order := sale.SaleOrder{
		Lines: []sale.SaleOrderLine{
			{
				PriceSubtotal: 100.0,
				PriceTax:      15.0,
				PriceTotal:    115.0,
			},
			{
				PriceSubtotal: 200.0,
				PriceTax:      30.0,
				PriceTotal:    230.0,
			},
		},
	}

	order.RecomputeTotals()

	if order.AmountUntaxed != 300.0 {
		t.Errorf("expected AmountUntaxed 300.0, got %f", order.AmountUntaxed)
	}
	if order.AmountTax != 45.0 {
		t.Errorf("expected AmountTax 45.0, got %f", order.AmountTax)
	}
	if order.AmountTotal != 345.0 {
		t.Errorf("expected AmountTotal 345.0, got %f", order.AmountTotal)
	}
}

func TestSaleOrder_StateMachine(t *testing.T) {
	order := &sale.SaleOrder{
		PartnerID: 1,
		DateOrder: time.Now(),
		State:     sale.OrderStateDraft,
		Lines: []sale.SaleOrderLine{
			{
				ProductID:     1,
				Name:          "Laptop",
				ProductUomQty: 1.0,
				UnitPrice:     1000.0,
			},
		},
	}

	// Validate draft
	if err := order.Validate(); err != nil {
		t.Fatalf("unexpected validation error on draft: %v", err)
	}

	// Draft -> Sent
	if err := order.ActionSend(); err != nil {
		t.Fatalf("failed to send quotation: %v", err)
	}
	if order.State != sale.OrderStateSent {
		t.Errorf("expected state 'sent', got '%s'", order.State)
	}

	// Sent -> Sale (Confirm)
	if err := order.ActionConfirm("SO/2026/00001"); err != nil {
		t.Fatalf("failed to confirm order: %v", err)
	}
	if order.State != sale.OrderStateSale {
		t.Errorf("expected state 'sale', got '%s'", order.State)
	}
	if order.Name != "SO/2026/00001" {
		t.Errorf("expected name 'SO/2026/00001', got '%s'", order.Name)
	}
	if order.InvoiceStatus != sale.InvoiceStatusToInvoice {
		t.Errorf("expected invoice_status 'to_invoice', got '%s'", order.InvoiceStatus)
	}

	// Cannot confirm again
	if err := order.ActionConfirm("SO/2026/00002"); err == nil {
		t.Errorf("expected error confirming an already confirmed order")
	}

	// Update invoice quantities
	order.Lines[0].QtyInvoiced = 1.0
	order.UpdateInvoiceStatus()
	if order.InvoiceStatus != sale.InvoiceStatusInvoiced {
		t.Errorf("expected invoice_status 'invoiced', got '%s'", order.InvoiceStatus)
	}

	// Cannot cancel invoiced order
	if err := order.ActionCancel(); err == nil {
		t.Errorf("expected error cancelling invoiced order")
	}

	// Sale -> Done
	if err := order.ActionDone(); err != nil {
		t.Fatalf("failed to lock order: %v", err)
	}
	if order.State != sale.OrderStateDone {
		t.Errorf("expected state 'done', got '%s'", order.State)
	}
}

func TestSaleOrder_CancelAndResetDraft(t *testing.T) {
	order := &sale.SaleOrder{
		PartnerID: 1,
		DateOrder: time.Now(),
		State:     sale.OrderStateDraft,
		Lines: []sale.SaleOrderLine{
			{
				ProductID:     2,
				Name:          "Office Chair",
				ProductUomQty: 1.0,
				UnitPrice:     150.0,
			},
		},
	}

	// Cancel draft
	if err := order.ActionCancel(); err != nil {
		t.Fatalf("failed to cancel draft: %v", err)
	}
	if order.State != sale.OrderStateCancel {
		t.Errorf("expected state 'cancel', got '%s'", order.State)
	}

	// Reset to draft
	if err := order.ActionDraft(); err != nil {
		t.Fatalf("failed to reset to draft: %v", err)
	}
	if order.State != sale.OrderStateDraft {
		t.Errorf("expected state 'draft', got '%s'", order.State)
	}
}
