package paymentstorage

import (
	"cashflow_backend/internal/domain/payment"
	"context"
	"testing"
)

func TestMemoryRepoPersistsPaymentPhase2Artifacts(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()
	token := &payment.PaymentToken{ProviderID: 1, PartnerID: 9, ProviderRef: "tok_1", DisplayName: "Visa", CompanyID: 1, Active: true}
	if err := repo.CreatePaymentToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	tokens, err := repo.ListPaymentTokens(ctx, 9)
	if err != nil || len(tokens) != 1 {
		t.Fatalf("tokens=%v err=%v", tokens, err)
	}
	log := &payment.WebhookLog{ProviderCode: "stripe", IdempotencyKey: "evt_1", Payload: []byte(`{}`)}
	if err := repo.CreateWebhookLog(ctx, log); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetWebhookLogByIdempotencyKey(ctx, "evt_1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateWebhookLog(ctx, &payment.WebhookLog{ProviderCode: "stripe", IdempotencyKey: "evt_1"}); err == nil {
		t.Fatal("expected duplicate webhook to fail")
	}
}
