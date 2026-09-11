package paymentinfra

import (
	"fmt"
	"net/http"
	"sync"

	"cashflow_backend/internal/domain/payment"
)

// NewDefaultRegistry creates a registry with the built-in manual provider and
// configurable remote gateway adapters.
func NewDefaultRegistry(client *http.Client, endpoints map[string]string, tokens map[string]string) *ProviderRegistry {
	registry := NewProviderRegistry()
	registry.Register("manual", ManualProvider{})
	for _, code := range []string{"stripe", "tap", "paypal"} {
		registry.Register(code, &RemoteProvider{Code: code, BaseURL: endpoints[code], AuthToken: tokens[code], HTTPClient: client})
	}
	return registry
}

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]payment.PaymentProviderInterface
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: make(map[string]payment.PaymentProviderInterface)}
}
func (r *ProviderRegistry) Register(code string, provider payment.PaymentProviderInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[code] = provider
}
func (r *ProviderRegistry) Get(code string) (payment.PaymentProviderInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[code]
	if !ok {
		return nil, fmt.Errorf("payment provider %q is not registered", code)
	}
	return provider, nil
}
