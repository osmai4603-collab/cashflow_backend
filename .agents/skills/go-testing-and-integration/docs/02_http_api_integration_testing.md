# HTTP API Integration Testing with `net/http/httptest`

In Go, integration testing of HTTP REST endpoints does not require starting an external process or opening a real TCP port. The standard library provides `net/http/httptest`, allowing in-memory execution of the full HTTP stack: routing, middlewares, context propagation, authentication, and error formatting.

---

## 1. Anatomy of an `httptest` Suite

An HTTP endpoint integration test verifies:

1. Routing correctness and URL parameter extraction.
2. Middleware execution (CORS, Auth, Tenant extraction, Rate limiting).
3. Status codes and header responses (e.g. `Content-Type: application/json`).
4. JSON request unmarshaling and response validation.

```go
func TestInvoiceEndpoint_Create(t *testing.T) {
 t.Parallel()

 // 1. Setup router and handler under test
 router := SetupRouter(testUsecase)

 // 2. Prepare mock request payload
 payload := `{"title":"Server Hosting","amount":25000}`
 req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", strings.NewReader(payload))
 req.Header.Set("Content-Type", "application/json")
 req.Header.Set("Authorization", "Bearer valid-test-token")

 // 3. Prepare response recorder
 rec := httptest.NewRecorder()

 // 4. Dispatch request in-memory
 router.ServeHTTP(rec, req)

 // 5. Assert HTTP Status Code
 if rec.Code != http.StatusCreated {
  t.Fatalf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
 }

 // 6. Assert Headers
 if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
  t.Errorf("expected application/json Content-Type, got %s", ct)
 }

 // 7. Parse and assert JSON response body
 var resp struct {
  Data struct {
   ID     string `json:"id"`
   Amount int64  `json:"amount"`
  } `json:"data"`
 }
 if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
  t.Fatalf("failed to decode response JSON: %v", err)
 }
 if resp.Data.ID == "" {
  t.Error("expected generated invoice ID in response data")
 }
}
```

---

## 2. Testing Error Handling & RFC 7807

Always write negative test cases to verify error codes and Problem Details formatting:

- Invalid JSON syntax $\rightarrow$ `400 Bad Request`.
- Missing required fields $\rightarrow$ `422 Unprocessable Entity` with field-level details.
- Missing authorization header $\rightarrow$ `401 Unauthorized`.
- Insufficient permissions $\rightarrow$ `403 Forbidden`.
- Conflicting resources $\rightarrow$ `409 Conflict`.
