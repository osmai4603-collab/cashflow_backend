package examples

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ExternalRequest represents the domain-level input for an outbound operation.
// Notice this contains NO third-party vendor SDK types.
type ExternalRequest struct {
	ResourceID string
	Payload    []byte
	Action     string
}

// ExternalResponse represents the domain-level output from an outbound operation.
type ExternalResponse struct {
	ExternalID  string
	Status      string
	ProcessedAt time.Time
}

// OutboundServicePort is the Outbound Port interface defined by the domain/usecase layer.
// Concrete infrastructure adapters implement this port to communicate with remote SaaS APIs.
type OutboundServicePort interface {
	ProviderCode() string
	Execute(ctx context.Context, req ExternalRequest) (*ExternalResponse, error)
}

// RemoteAPIAdapter is an Infrastructure Adapter communicating with a remote vendor API.
type RemoteAPIAdapter struct {
	providerCode string
	baseURL      string
	apiKey       string
	client       *http.Client
	mu           sync.RWMutex
	consecErrors int
	circuitOpen  bool
	openUntil    time.Time
}

// NewRemoteAPIAdapter instantiates the resilient adapter.
func NewRemoteAPIAdapter(providerCode, baseURL, apiKey string) *RemoteAPIAdapter {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 25,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &RemoteAPIAdapter{
		providerCode: providerCode,
		baseURL:      baseURL,
		apiKey:       apiKey,
		client: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
	}
}

func (a *RemoteAPIAdapter) ProviderCode() string {
	return a.providerCode
}

// Execute performs the remote network request with timeout bounding, idempotency, and circuit breaking.
func (a *RemoteAPIAdapter) Execute(ctx context.Context, req ExternalRequest) (*ExternalResponse, error) {
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

	_ = execCtx

	// In real implementation:
	// httpReq, _ := http.NewRequestWithContext(execCtx, "POST", a.baseURL+"/action", bytes.NewReader(req.Payload))
	// httpReq.Header.Set("Authorization", "Bearer " + a.apiKey)
	// httpReq.Header.Set("Idempotency-Key", req.ResourceID)
	// resp, err := a.client.Do(httpReq)

	if req.ResourceID == "" {
		return nil, errors.New("resource ID required")
	}

	// Reset circuit on success
	a.mu.Lock()
	a.consecErrors = 0
	a.circuitOpen = false
	a.mu.Unlock()

	return &ExternalResponse{
		ExternalID:  fmt.Sprintf("%s_ext_%d", a.providerCode, time.Now().UnixNano()),
		Status:      "completed",
		ProcessedAt: time.Now().UTC(),
	}, nil
}

// ProviderRegistry manages available external adapters in a thread-safe registry.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]OutboundServicePort
}

// NewProviderRegistry instantiates a thread-safe provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]OutboundServicePort),
	}
}

// Register adds an outbound adapter into the registry.
func (r *ProviderRegistry) Register(p OutboundServicePort) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ProviderCode()] = p
}

// Resolve retrieves an outbound adapter by its provider code.
func (r *ProviderRegistry) Resolve(code string) (OutboundServicePort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[code]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", code)
	}
	return p, nil
}
