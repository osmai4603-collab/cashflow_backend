package payment

import "context"

type PaymentProviderInterface interface {
	GetCode() string
	InitiatePayment(context.Context, *PaymentTransaction) (*PaymentInitResult, error)
	CapturePayment(context.Context, *PaymentTransaction) error
	VoidPayment(context.Context, *PaymentTransaction) error
	Refund(context.Context, *PaymentTransaction, float64) (*PaymentRefund, error)
	HandleWebhook(context.Context, []byte, map[string]string) (*WebhookResult, error)
	Tokenize(context.Context, int64, map[string]string) (*PaymentToken, error)
	GetPaymentMethods(context.Context) ([]ProviderPaymentMethod, error)
}

type PaymentInitResult struct {
	RedirectURL    string            `json:"redirect_url,omitempty"`
	ClientSecret   string            `json:"client_secret,omitempty"`
	FormData       map[string]string `json:"form_data,omitempty"`
	TransactionRef string            `json:"transaction_ref"`
}

type ProviderPaymentMethod struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
