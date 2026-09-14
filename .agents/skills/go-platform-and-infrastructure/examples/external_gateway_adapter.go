package examples

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PaymentRequest represents the domain-level input for a payment operation.
// Notice this contains NO third-party SDK types (e.g. stripe.PaymentIntent).
type PaymentRequest struct {
	ReferenceID string
	AmountCents int64
	Currency    string
	Description string
}

// PaymentResponse represents the domain-level result.
type PaymentResponse struct {
	TransactionID string
	Status        string
	ProcessedAt   time.Time
}

// PaymentProviderPort is the Outbound Port interface defined by the domain/usecase layer.
type PaymentProviderPort interface {
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
}

// RemotePaymentAdapter is an Infrastructure Adapter that communicates with a remote vendor API.
type RemotePaymentAdapter struct {
	providerCode string
	baseURL      string
	apiKey       string
	client       *http.Client
	mu           sync.RWMutex
	consecErrors int
	circuitOpen  bool
	openUntil    time.Time
}

// NewRemotePaymentAdapter instantiates the resilient adapter.
func NewRemotePaymentAdapter(providerCode, baseURL, apiKey string) *RemotePaymentAdapter {
	return &RemotePaymentAdapter{
		providerCode: providerCode,
		baseURL:      baseURL,
		apiKey:       apiKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ProcessPayment executes the external payment with circuit breaking and timeouts.
func (a *RemotePaymentAdapter) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	// 1. Check Circuit Breaker status
	a.mu.RLock()
	if a.circuitOpen {
		if time.Now().Before(a.openUntil) {
			a.mu.RUnlock()
			return nil, fmt.Errorf("circuit breaker open for %s: failing fast", a.providerCode)
		}
		// Trial state (half-open)
	}
	a.mu.RUnlock()

	// 2. Bound execution with explicit timeout
	execCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	// In real code: construct HTTP request, marshal vendor-specific JSON, invoke API
	_ = execCtx

	// Simulated remote outcome
	if req.AmountCents <= 0 {
		return nil, errors.New("invalid payment amount")
	}

	// Reset circuit state on success
	a.mu.Lock()
	a.consecErrors = 0
	a.circuitOpen = false
	a.mu.Unlock()

	return &PaymentResponse{
		TransactionID: fmt.Sprintf("%s_tx_%d", a.providerCode, time.Now().UnixNano()),
		Status:        "succeeded",
		ProcessedAt:   time.Now().UTC(),
	}, nil
}

// ProviderRegistry manages available external payment adapters.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]PaymentProviderPort
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]PaymentProviderPort),
	}
}

func (r *ProviderRegistry) Register(code string, provider PaymentProviderPort) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[code] = provider
}

func (r *ProviderRegistry) Get(code string) (PaymentProviderPort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[code]
	if !ok {
		return nil, fmt.Errorf("payment provider %q not registered", code)
	}
	return p, nil
}
