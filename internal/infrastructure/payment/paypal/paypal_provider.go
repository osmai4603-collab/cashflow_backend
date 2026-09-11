package paypal

import (
	paymentinfra "cashflow_backend/internal/infrastructure/payment"
	"net/http"
)

func New(baseURL, token string, client *http.Client) *paymentinfra.RemoteProvider {
	return &paymentinfra.RemoteProvider{Code: "paypal", BaseURL: baseURL, AuthToken: token, HTTPClient: client}
}
