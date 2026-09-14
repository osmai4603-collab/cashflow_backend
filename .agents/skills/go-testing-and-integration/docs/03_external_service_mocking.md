# Zero-Dependency External Service Mocking

Production backend services frequently interact with external third-party HTTP APIs (payment processors, SMS gateways, email delivery, currency exchange feeds). Tests must **never** make live network requests to third-party endpoints.

Using `net/http/httptest.NewServer` from the standard library, we can instantiate an in-process HTTP mock server that simulates real network responses with zero external dependencies.

---

## 1. Mock Server Architecture

```text
┌────────────────────────────────────────────────────────┐
│                      Go Test Runner                    │
│                                                        │
│  ┌───────────────────────┐   HTTP   ┌───────────────┐  │
│  │   Service / Client    ├─────────►│ httptest.     │  │
│  │   baseURL = mock.URL  │◄─────────┤ Server (Mock) │  │
│  └───────────────────────┘          └───────────────┘  │
└────────────────────────────────────────────────────────┘
```

---

## 2. Implementing a Configurable Mock Server

```go
func TestPaymentGateway_ChargeSuccess(t *testing.T) {
 t.Parallel()

 // 1. Create mock HTTP server with custom handler
 mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  // Assert method and path
  if r.Method != http.MethodPost || r.URL.Path != "/v1/charges" {
   http.Error(w, "not found", http.StatusNotFound)
   return
  }

  // Assert authorization header was sent
  if r.Header.Get("Authorization") != "Bearer test-api-key" {
   http.Error(w, "unauthorized", http.StatusUnauthorized)
   return
  }

  // Respond with mock success payload
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  _, _ = w.Write([]byte(`{"id":"ch_123","status":"succeeded","amount":5000}`))
 }))
 // Guarantee server teardown at end of test
 t.Cleanup(mockServer.Close)

 // 2. Initialize client with mockServer.URL
 client := NewPaymentClient(mockServer.URL, "test-api-key")

 // 3. Execute client call
 charge, err := client.Charge(context.Background(), 5000, "USD")
 if err != nil {
  t.Fatalf("unexpected charge error: %v", err)
 }

 // 4. Verify client received and parsed response
 if charge.ID != "ch_123" || charge.Status != "succeeded" {
  t.Errorf("unexpected charge state: %+v", charge)
 }
}
```

---

## 3. Simulating Network Failures & Edge Cases

Mock servers can easily simulate production error scenarios:

- **500 Internal Server Error**: Return `http.StatusInternalServerError` to verify retry and backoff logic.
- **Timeouts & Slow Responses**: Insert `time.Sleep(100 * time.Millisecond)` in the mock handler to verify client context cancellation (`context.WithTimeout`).
- **Malformed JSON**: Return invalid byte slices to test decoder resilience.
