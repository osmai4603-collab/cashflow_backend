package pos

import "testing"

func TestPosOrderComputesTotalsAndSupportsMultiplePayments(t *testing.T) {
	order := &PosOrder{
		ClientUUID: "offline-order-1",
		SessionID:  10,
		UserID:     20,
		CompanyID:  30,
		Lines: []PosOrderLine{{
			ProductID: 1,
			Qty:       2,
			PriceUnit: 50,
			Discount:  10,
			TaxRate:   15,
		}},
	}
	if err := order.Validate(); err != nil {
		t.Fatalf("validate order: %v", err)
	}
	order.RecomputeTotals()
	if order.AmountUntaxed != 90 || order.AmountTax != 13.5 || order.AmountTotal != 103.5 {
		t.Fatalf("unexpected totals: untaxed=%v tax=%v total=%v", order.AmountUntaxed, order.AmountTax, order.AmountTotal)
	}
	if err := order.AddPayment(PosPayment{PaymentMethodID: 1, Amount: 50}); err != nil {
		t.Fatalf("add first payment: %v", err)
	}
	if err := order.AddPayment(PosPayment{PaymentMethodID: 2, Amount: 60}); err != nil {
		t.Fatalf("add second payment: %v", err)
	}
	if order.State != OrderStatePaid || order.AmountReturn != 6.5 {
		t.Fatalf("unexpected payment result: state=%s paid=%v return=%v", order.State, order.AmountPaid, order.AmountReturn)
	}
}

func TestPosOrderRejectsInvalidDiscount(t *testing.T) {
	order := &PosOrder{
		ClientUUID: "order-1",
		SessionID:  1,
		UserID:     1,
		CompanyID:  1,
		Lines:      []PosOrderLine{{ProductID: 1, Qty: 1, PriceUnit: 10, Discount: 101}},
	}
	if err := order.Validate(); err == nil {
		t.Fatal("expected invalid discount to fail validation")
	}
}
