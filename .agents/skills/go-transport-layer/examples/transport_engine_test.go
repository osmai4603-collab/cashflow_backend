package transportexamples

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockTokenValidator implements TokenValidator for testing
type mockTokenValidator struct {
	validTokens map[string]*SecurityPrincipal
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, token string) (*SecurityPrincipal, error) {
	if p, ok := m.validTokens[token]; ok {
		return p, nil
	}
	return nil, errors.New("token verification failed")
}

func setupTestPipeline() (*PreHandlingPipeline, *mockTokenValidator) {
	validator := &mockTokenValidator{
		validTokens: map[string]*SecurityPrincipal{
			"valid-token-user-1": {
				UserID:   "usr_123",
				TenantID: "tenant_abc",
				Roles:    []string{"operator"},
			},
			"valid-token-admin": {
				UserID:   "usr_admin",
				TenantID: "tenant_abc",
				Roles:    []string{"admin"},
				IsAdmin:  true,
			},
		},
	}
	pipeline := NewPreHandlingPipeline(validator, nil, 2*time.Second)
	return pipeline, validator
}

// ----------------------------------------------------------------------------
// HTTP Ingress Tests
// ----------------------------------------------------------------------------

func TestHTTPMiddleware_Success(t *testing.T) {
	pipeline, _ := setupTestPipeline()
	paymentService := NewPaymentService()

	// Handler executing pure domain logic
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, err := paymentService.ProcessPayment(r.Context(), CreatePaymentDTO{
			AccountID: "acc_456",
			Amount:    150.00,
			Currency:  "USD",
		})
		if err != nil {
			WriteHTTPError(w, r.Header.Get("X-Request-ID"), err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	wrapped := HTTPMiddleware(pipeline)(handler)

	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer valid-token-user-1")
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var res PaymentResult
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", res.Status)
	}
	if res.CallerUserID != "usr_123" {
		t.Errorf("expected caller usr_123, got %s", res.CallerUserID)
	}
	if res.ProtocolUsed != "HTTP" {
		t.Errorf("expected protocol HTTP, got %s", res.ProtocolUsed)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be populated")
	}
}

func TestHTTPMiddleware_Unauthorized(t *testing.T) {
	pipeline, _ := setupTestPipeline()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := HTTPMiddleware(pipeline)(handler)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d", rec.Code)
	}

	var problem ProblemDetails
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode problem details: %v", err)
	}
	if problem.Status != http.StatusUnauthorized {
		t.Errorf("expected problem status 401, got %d", problem.Status)
	}
}

func TestHTTPMiddleware_PanicRecovery(t *testing.T) {
	pipeline, _ := setupTestPipeline()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected nil pointer inside handler")
	})

	wrapped := HTTPMiddleware(pipeline)(handler)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Ensure the test itself doesn't crash
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 on panic recovery, got %d", rec.Code)
	}
}

// ----------------------------------------------------------------------------
// gRPC Ingress Tests
// ----------------------------------------------------------------------------

func TestGRPCUnaryInterceptor_Success(t *testing.T) {
	pipeline, _ := setupTestPipeline()
	paymentService := NewPaymentService()

	outgoingHeaders := make(map[string]string)
	interceptor := GRPCUnaryInterceptor(pipeline, func(ctx context.Context, key, val string) {
		outgoingHeaders[key] = val
	})

	// Mock gRPC context with metadata
	baseCtx := context.Background()
	tc := NewGRPCTransportContext(baseCtx, map[string][]string{
		"authorization": {"Bearer valid-token-admin"},
		"user-agent":    {"grpc-node/1.24"},
	}, "198.51.100.4:50051", GenerateCorrelationID())

	// Execute via pipeline directly or interceptor
	enrichedCtx, err := pipeline.Execute(tc)
	if err != nil {
		t.Fatalf("pipeline execution failed: %v", err)
	}

	// Mock gRPC handler
	unaryHandler := func(ctx context.Context, req any) (any, error) {
		dto := req.(CreatePaymentDTO)
		return paymentService.ProcessPayment(ctx, dto)
	}

	info := &UnaryServerInfo{FullMethod: "/payments.PaymentService/ProcessPayment"}
	resp, err := interceptor(enrichedCtx, CreatePaymentDTO{
		AccountID: "acc_999",
		Amount:    500.00,
		Currency:  "EUR",
	}, info, unaryHandler)

	if err != nil {
		t.Fatalf("expected gRPC call to succeed, got error: %v", err)
	}

	res, ok := resp.(*PaymentResult)
	if !ok {
		t.Fatalf("expected *PaymentResult, got %T", resp)
	}

	if res.Status != "CONFIRMED" {
		t.Errorf("expected CONFIRMED, got %s", res.Status)
	}
	if res.CallerUserID != "usr_admin" {
		t.Errorf("expected usr_admin, got %s", res.CallerUserID)
	}
	if res.ProtocolUsed != "GRPC" {
		t.Errorf("expected protocol GRPC, got %s", res.ProtocolUsed)
	}
}

// ----------------------------------------------------------------------------
// Bidirectional Status Mapping Tests
// ----------------------------------------------------------------------------

func TestDomainErrorMapping(t *testing.T) {
	tests := []struct {
		err          error
		expectedHTTP int
		expectedGRPC GRPCCode
	}{
		{ErrNotFound, http.StatusNotFound, CodeNotFound},
		{ErrUnauthorized, http.StatusUnauthorized, CodeUnauthenticated},
		{ErrForbidden, http.StatusForbidden, CodePermissionDenied},
		{ErrInvalidInput, http.StatusBadRequest, CodeInvalidArgument},
		{ErrConflict, http.StatusConflict, CodeAlreadyExists},
		{ErrPreconditionFailed, http.StatusPreconditionFailed, CodeFailedPrecondition},
		{ErrRateLimited, http.StatusTooManyRequests, CodeResourceExhausted},
		{ErrDeadlineExceeded, http.StatusGatewayTimeout, CodeDeadlineExceeded},
		{ErrInternal, http.StatusInternalServerError, CodeInternal},
	}

	for _, tt := range tests {
		httpStatus := MapToHTTPStatus(tt.err)
		if httpStatus != tt.expectedHTTP {
			t.Errorf("for error %v: expected HTTP status %d, got %d", tt.err, tt.expectedHTTP, httpStatus)
		}

		grpcErr := MapToGRPCError(tt.err)
		statusErr, ok := grpcErr.(*GRPCStatusError)
		if !ok {
			t.Fatalf("expected *GRPCStatusError, got %T", grpcErr)
		}
		if statusErr.Code != tt.expectedGRPC {
			t.Errorf("for error %v: expected gRPC code %d, got %d", tt.err, tt.expectedGRPC, statusErr.Code)
		}
	}
}

// ----------------------------------------------------------------------------
// Context Key Collision Isolation Test
// ----------------------------------------------------------------------------

func TestContextKeyIsolation(t *testing.T) {
	type otherKey int
	const mockCollisionKey otherKey = 0

	ctx := context.Background()
	ctx = context.WithValue(ctx, mockCollisionKey, "external-library-value")

	info := TransportInfo{
		Protocol:  ProtocolHTTP,
		RequestID: "req-1234",
		ClientIP:  "127.0.0.1",
	}
	ctx = WithTransportInfo(ctx, info)

	// Verify that the external key was not clobbered
	extVal := ctx.Value(mockCollisionKey).(string)
	if extVal != "external-library-value" {
		t.Errorf("context key collision detected: external value overwritten")
	}

	// Verify that transport info is retrieved accurately
	retrieved, ok := GetTransportInfo(ctx)
	if !ok || retrieved.RequestID != "req-1234" {
		t.Errorf("failed to retrieve unexported TransportInfo")
	}
}
