package tap

import (
	paymentinfra "cashflow_backend/internal/infrastructure/payment"
	"net/http"
)

func New(baseURL, token string, client *http.Client) *paymentinfra.RemoteProvider {
	return &paymentinfra.RemoteProvider{Code: "tap", BaseURL: baseURL, AuthToken: token, HTTPClient: client}
}
