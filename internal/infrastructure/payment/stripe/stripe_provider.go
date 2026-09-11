package stripe

import (
	paymentinfra "cashflow_backend/internal/infrastructure/payment"
	"net/http"
)

func New(baseURL, token string, client *http.Client) *paymentinfra.RemoteProvider {
	return &paymentinfra.RemoteProvider{Code: "stripe", BaseURL: baseURL, AuthToken: token, HTTPClient: client}
}
