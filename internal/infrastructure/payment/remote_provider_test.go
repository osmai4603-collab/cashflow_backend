package paymentinfra

import (
	"cashflow_backend/internal/domain/payment"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemoteProviderOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" { t.Errorf("missing provider authorization") }
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path { case "/payments": _, _ = w.Write([]byte(`{"transaction_ref":"remote-1"}`)); case "/payments/remote-1/refunds": _, _ = w.Write([]byte(`{"provider_reference":"refund-1"}`)); case "/payment-methods": _, _ = w.Write([]byte(`[{"code":"card","name":"Card","active":true}]`)); default: _, _ = w.Write([]byte(`{}`)) }
	}))
	defer server.Close()
	provider := &RemoteProvider{Code: "stripe", BaseURL: server.URL, AuthToken: "secret", HTTPClient: server.Client()}
	tx := &payment.PaymentTransaction{ID: 4, Reference: "tx-1", Amount: 10, Currency: "SAR", ProviderReference: "remote-1", CompanyID: 1}
	init, err := provider.InitiatePayment(context.Background(), tx); if err != nil || init.TransactionRef != "remote-1" { t.Fatalf("init=%+v err=%v", init, err) }
	refund, err := provider.Refund(context.Background(), tx, 5); if err != nil || refund.ProviderReference != "refund-1" { t.Fatalf("refund=%+v err=%v", refund, err) }
	methods, err := provider.GetPaymentMethods(context.Background()); if err != nil || len(methods) != 1 { t.Fatalf("methods=%+v err=%v", methods, err) }
}