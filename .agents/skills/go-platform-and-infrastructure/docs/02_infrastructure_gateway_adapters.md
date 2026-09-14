# Infrastructure Gateway Adapters & Vendor SDK Isolation

When an application integrates with external third-party services (payment processors, notification providers, regulatory portals), improper encapsulation allows vendor-specific data structures to leak into core business logic. This creates high coupling, making provider changes difficult and unit testing brittle.

---

## 1. The Outbound Adapter Pattern

The core domain declares what it needs via an **Outbound Port Interface**. The Infrastructure layer provides a concrete **Adapter** that implements this interface, translating internal domain entities into external vendor API calls.

```text
┌────────────────────────────────────────────────────────┐
│                   Domain / Use Case                    │
│   type PaymentGateway interface {                      │
│       Charge(ctx context.Context, p Payment) (Result)  │
│   }                                                    │
└───────────────────────────┬────────────────────────────┘
                            │ implements interface
                            ▼
┌────────────────────────────────────────────────────────┐
│             Infrastructure Adapter Layer               │
│   type StripeAdapter struct {                          │
│       client *stripe.Client                            │
│   }                                                    │
│   func (s *StripeAdapter) Charge(...) (Result) {       │
│       // Converts Payment -> stripe.PaymentIntentParams│
│       // Calls Stripe API                              │
│       // Converts stripe.PaymentIntent -> Result       │
│   }                                                    │
└────────────────────────────────────────────────────────┘
```

---

## 2. The Provider Registry Pattern

When an application supports multiple external vendors for the same functional capability (e.g. `manual`, `stripe`, `paypal`, `tap`), use a thread-safe `ProviderRegistry`:

```go
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]PaymentProviderInterface
}

func (r *ProviderRegistry) Register(code string, p PaymentProviderInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[code] = p
}

func (r *ProviderRegistry) Get(code string) (PaymentProviderInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[code]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", code)
	}
	return p, nil
}
```

---

## 3. Resilience: Timeouts, Retries & Circuit Breaking

External calls can hang indefinitely, exhaust connection pools, and cascade failures back into the core application. Every infrastructure adapter must enforce:

1. **Context-Bounded Timeouts**:
   ```go
   ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
   defer cancel()
   ```
2. **Exponential Backoff with Full Jitter**:
   Retry idempotent network failures (HTTP 502, 503, 504) with randomized backoff. Never retry non-idempotent operations or client errors (HTTP 4xx).
3. **Circuit Breaker**:
   If an external provider fails consistently (e.g., 5 failures in 10 seconds), open the circuit to fail fast immediately without making network calls, allowing the provider time to recover.
