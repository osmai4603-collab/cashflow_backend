package paymentinfra

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/payment"
)

type ManualProvider struct{}

func (ManualProvider) GetCode() string { return "manual" }
func (ManualProvider) InitiatePayment(_ context.Context, tx *payment.PaymentTransaction) (*payment.PaymentInitResult, error) {
	return &payment.PaymentInitResult{TransactionRef: tx.Reference}, nil
}
func (ManualProvider) CapturePayment(_ context.Context, tx *payment.PaymentTransaction) error {
	return tx.Transition(payment.TransactionStateDone)
}
func (ManualProvider) VoidPayment(_ context.Context, tx *payment.PaymentTransaction) error {
	return tx.Transition(payment.TransactionStateCancelled)
}
func (ManualProvider) Refund(_ context.Context, tx *payment.PaymentTransaction, amount float64) (*payment.PaymentRefund, error) {
	refund := &payment.PaymentRefund{OriginalTxID: tx.ID, Amount: amount, Currency: tx.Currency, State: "done", CompanyID: tx.CompanyID, CreatedAt: time.Now().UTC()}
	return refund, refund.Validate(tx)
}
func (ManualProvider) HandleWebhook(_ context.Context, payload []byte, _ map[string]string) (*payment.WebhookResult, error) {
	return &payment.WebhookResult{RawPayload: payload}, nil
}
func (ManualProvider) Tokenize(_ context.Context, partnerID int64, tokenData map[string]string) (*payment.PaymentToken, error) {
	return &payment.PaymentToken{PartnerID: partnerID, ProviderRef: tokenData["provider_ref"], DisplayName: tokenData["display_name"], Active: true}, nil
}
func (ManualProvider) GetPaymentMethods(_ context.Context) ([]payment.ProviderPaymentMethod, error) {
	return []payment.ProviderPaymentMethod{{Code: "bank_transfer", Name: "Bank transfer", Active: true}}, nil
}
