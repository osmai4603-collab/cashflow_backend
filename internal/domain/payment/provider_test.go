package payment

import "testing"

func TestPaymentRefundValidation(t *testing.T) {
	tx := &PaymentTransaction{ID: 10, Amount: 100, Currency: "SAR"}
	refund := &PaymentRefund{OriginalTxID: 10, Amount: 40, Currency: "SAR"}
	if err := refund.Validate(tx); err != nil {
		t.Fatal(err)
	}
	if refund.State != "pending" {
		t.Fatalf("expected pending state, got %q", refund.State)
	}
	if err := (&PaymentRefund{OriginalTxID: 10, Amount: 101, Currency: "SAR"}).Validate(tx); err == nil {
		t.Fatal("expected excessive refund to fail")
	}
}
