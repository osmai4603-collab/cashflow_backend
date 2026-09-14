package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// RecordedRequest tracks headers and body received by the mock server.
type RecordedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   []byte
}

// MockExternalService provides an in-process HTTP mock server.
type MockExternalService struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []RecordedRequest
	handler  http.HandlerFunc
}

// NewMockExternalService creates and starts a new mock server.
func NewMockExternalService(handler http.HandlerFunc) *MockExternalService {
	m := &MockExternalService{
		handler: handler,
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		m.mu.Lock()
		m.requests = append(m.requests, RecordedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Header: r.Header.Clone(),
			Body:   body,
		})
		m.mu.Unlock()

		if m.handler != nil {
			m.handler(w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))

	return m
}

// URL returns the base URL of the mock server.
func (m *MockExternalService) URL() string {
	return m.server.URL
}

// Close terminates the mock server.
func (m *MockExternalService) Close() {
	m.server.Close()
}

// Requests returns all recorded incoming requests.
func (m *MockExternalService) Requests() []RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]RecordedRequest, len(m.requests))
	copy(copied, m.requests)
	return copied
}

// ExternalPaymentClient represents a service client communicating with an external API.
type ExternalPaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewExternalPaymentClient(baseURL string) *ExternalPaymentClient {
	return &ExternalPaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *ExternalPaymentClient) Charge(ctx context.Context, amount int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/charges", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected upstream status: %d", resp.StatusCode)
	}

	var result struct {
		TransactionID string `json:"tx_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.TransactionID, nil
}

// ExampleMockServerTest demonstrates how to test an external integration using the mock.
func ExampleMockServerTest(t *testing.T) {
	t.Parallel()

	// 1. Start mock server with custom canned behavior
	mock := NewMockExternalService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tx_id":"txn_999000"}`))
	})
	t.Cleanup(mock.Close)

	// 2. Point client to mock URL
	client := NewExternalPaymentClient(mock.URL())

	// 3. Perform operation
	txID, err := client.Charge(context.Background(), 5000)
	if err != nil {
		t.Fatalf("unexpected charge error: %v", err)
	}

	if txID != "txn_999000" {
		t.Errorf("expected tx_id 'txn_999000', got %s", txID)
	}

	// 4. Verify request was received
	reqs := mock.Requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(reqs))
	}
	if reqs[0].Path != "/v1/charges" {
		t.Errorf("expected path '/v1/charges', got %s", reqs[0].Path)
	}
}
