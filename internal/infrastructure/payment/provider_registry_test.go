package paymentinfra

import "testing"

func TestNewDefaultRegistry(t *testing.T) {
	registry := NewDefaultRegistry(nil, map[string]string{"stripe": "https://stripe.test"}, map[string]string{"stripe": "secret"})
	for _, code := range []string{"manual", "stripe", "tap", "paypal"} {
		provider, err := registry.Get(code)
		if err != nil || provider.GetCode() != code {
			t.Fatalf("provider %s: %v", code, err)
		}
	}
}
