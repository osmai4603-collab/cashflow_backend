package authlifecycle

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Helper to generate an RSA test keypair
func generateTestRSAKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA test key: %v", err)
	}
	return privKey, &privKey.PublicKey
}

// Helper to craft signed JWT access tokens
func createTestToken(
	t *testing.T,
	privKey *rsa.PrivateKey,
	subject, tenantID string,
	roles, scopes []string,
	issuer, audience string,
	expiresAt time.Time,
) string {
	t.Helper()

	claims := CustomClaims{
		TenantID: tenantID,
		Roles:    roles,
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return tokenString
}

// ============================================================================
// 1. JWT VALIDATOR TESTS
// ============================================================================

func TestJWTTokenValidator_Success(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	validator := NewJWTTokenValidator(pubKey, "https://auth.example.com", "api://cashflow", 10*time.Second)

	rawToken := createTestToken(
		t, privKey, "user_123", "tenant_abc",
		[]string{"manager", "auditor"},
		[]string{"invoices:read", "invoices:approve"},
		"https://auth.example.com", "api://cashflow",
		time.Now().Add(15*time.Minute),
	)

	principal, err := validator.ValidateToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if principal.ID != "user_123" {
		t.Errorf("expected ID 'user_123', got '%s'", principal.ID)
	}
	if principal.TenantID != "tenant_abc" {
		t.Errorf("expected TenantID 'tenant_abc', got '%s'", principal.TenantID)
	}
	if !principal.HasRole("manager") || !principal.HasRole("auditor") {
		t.Errorf("missing expected roles: %+v", principal.Roles)
	}
	if !principal.HasScope("invoices:approve") {
		t.Errorf("missing expected scope 'invoices:approve'")
	}
}

func TestJWTTokenValidator_Expired(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	validator := NewJWTTokenValidator(pubKey, "https://auth.example.com", "api://cashflow", 1*time.Second)

	// Token expired 10 minutes ago
	rawToken := createTestToken(
		t, privKey, "user_123", "tenant_abc",
		[]string{"user"}, []string{"read"},
		"https://auth.example.com", "api://cashflow",
		time.Now().Add(-10*time.Minute),
	)

	_, err := validator.ValidateToken(context.Background(), rawToken)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got: %v", err)
	}
}

func TestJWTTokenValidator_AlgConfusionAttackDefense(t *testing.T) {
	_, pubKey := generateTestRSAKey(t)
	validator := NewJWTTokenValidator(pubKey, "https://auth.example.com", "api://cashflow", 10*time.Second)

	// Attacker attempts to sign a token using HMAC-SHA256
	claims := CustomClaims{
		TenantID: "tenant_evil",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "hacker",
			Issuer:   "https://auth.example.com",
			Audience: jwt.ClaimStrings{"api://cashflow"},
		},
	}
	hmacToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	forgedToken, err := hmacToken.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to create HMAC token: %v", err)
	}

	_, err = validator.ValidateToken(context.Background(), forgedToken)
	if err == nil {
		t.Fatal("expected validator to reject HMAC signed token, but it succeeded")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got: %v", err)
	}
}

// ============================================================================
// 2. CONTEXT SAFETY TESTS
// ============================================================================

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()

	// Initially empty
	_, ok := ExtractPrincipal(ctx)
	if ok {
		t.Fatal("expected ExtractPrincipal to return false on empty context")
	}

	_, err := MustExtractPrincipal(ctx)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}

	p := &Principal{ID: "usr_99", TenantID: "ten_99"}
	ctxWithPrincipal := InjectPrincipal(ctx, p)

	extracted, ok := ExtractPrincipal(ctxWithPrincipal)
	if !ok || extracted == nil {
		t.Fatal("expected principal to be successfully extracted")
	}
	if extracted.ID != "usr_99" {
		t.Errorf("expected ID 'usr_99', got '%s'", extracted.ID)
	}
}

// ============================================================================
// 3. HTTP MIDDLEWARE INTEGRATION TESTS
// ============================================================================

func TestHTTPAuthAndScopeMiddleware(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	validator := NewJWTTokenValidator(pubKey, "https://auth.example.com", "api://cashflow", 10*time.Second)

	authMw := HTTPAuthMiddleware(validator)
	scopeMw := RequireScopeHTTPMiddleware("reports:export")

	handlerExecuted := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := MustExtractPrincipal(r.Context())
		if err != nil {
			t.Errorf("handler failed to extract principal: %v", err)
		}
		if p.ID != "user_test" {
			t.Errorf("unexpected user ID in handler: %s", p.ID)
		}
		handlerExecuted = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap in middleware chain
	securedHandler := authMw(scopeMw(testHandler))

	t.Run("Missing Authorization Header -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/reports", nil)
		rec := httptest.NewRecorder()
		securedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Valid Token but Missing Scope -> 403", func(t *testing.T) {
		tokenWithoutScope := createTestToken(
			t, privKey, "user_test", "tenant_test",
			[]string{"viewer"}, []string{"reports:view"}, // lacks reports:export
			"https://auth.example.com", "api://cashflow",
			time.Now().Add(15*time.Minute),
		)

		req := httptest.NewRequest(http.MethodGet, "/reports", nil)
		req.Header.Set("Authorization", "Bearer "+tokenWithoutScope)
		rec := httptest.NewRecorder()
		securedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", rec.Code)
		}
	})

	t.Run("Valid Token with Required Scope -> 200", func(t *testing.T) {
		handlerExecuted = false
		validToken := createTestToken(
			t, privKey, "user_test", "tenant_test",
			[]string{"viewer"}, []string{"reports:export"},
			"https://auth.example.com", "api://cashflow",
			time.Now().Add(15*time.Minute),
		)

		req := httptest.NewRequest(http.MethodGet, "/reports", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()
		securedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
		if !handlerExecuted {
			t.Errorf("expected final handler to be executed")
		}
	})
}

// ============================================================================
// 4. GRPC INTERCEPTOR TESTS
// ============================================================================

func TestGRPCUnaryInterceptors(t *testing.T) {
	privKey, pubKey := generateTestRSAKey(t)
	validator := NewJWTTokenValidator(pubKey, "https://auth.example.com", "api://cashflow", 10*time.Second)

	rawToken := createTestToken(
		t, privKey, "grpc_user", "grpc_tenant",
		[]string{"admin"}, []string{"admin:write"},
		"https://auth.example.com", "api://cashflow",
		time.Now().Add(15*time.Minute),
	)

	// Mock incoming metadata carrier
	carrier := MapHeaderCarrier{
		"authorization": "Bearer " + rawToken,
	}

	authInterceptor := GRPCUnaryAuthInterceptor(validator, func(ctx context.Context) (HeaderCarrier, bool) {
		return carrier, true
	})

	scopeInterceptor := GRPCUnaryScopeInterceptor("admin:write")

	info := &UnaryServerInfo{FullMethod: "/api.v1.Service/AdminAction"}

	// Chain the interceptors
	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		p, err := MustExtractPrincipal(ctx)
		if err != nil {
			return nil, err
		}
		if p.ID != "grpc_user" {
			t.Errorf("expected user grpc_user, got %s", p.ID)
		}
		handlerCalled = true
		return "success", nil
	}

	chainedHandler := func(ctx context.Context, req any) (any, error) {
		return scopeInterceptor(ctx, req, info, handler)
	}

	resp, err := authInterceptor(context.Background(), "req_data", info, chainedHandler)
	if err != nil {
		t.Fatalf("unexpected gRPC interceptor chain error: %v", err)
	}
	if resp != "success" || !handlerCalled {
		t.Errorf("expected handler execution with 'success' response")
	}
}

// ============================================================================
// 5. APPLICATION USE CASE & FINE-GRAINED PDP TEST
// ============================================================================

type mockInvoiceRepo struct {
	invoices map[string]*Invoice
}

func (m *mockInvoiceRepo) GetByID(ctx context.Context, tenantID, invoiceID string) (*Invoice, error) {
	inv, ok := m.invoices[invoiceID]
	if !ok || inv.TenantID != tenantID {
		return nil, nil // Return nil to simulate not found under this tenant
	}
	return inv, nil
}

func (m *mockInvoiceRepo) Update(ctx context.Context, inv *Invoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

type mockAuthorizer struct {
	allow bool
}

func (m *mockAuthorizer) Authorize(ctx context.Context, sub *Principal, act Action, res Resource) (bool, error) {
	return m.allow, nil
}

func TestApproveInvoiceUseCase(t *testing.T) {
	repo := &mockInvoiceRepo{
		invoices: map[string]*Invoice{
			"inv_101": {
				ID:       "inv_101",
				TenantID: "tenant_alpha",
				OwnerID:  "user_bob",
				Status:   "draft",
				Amount:   5000,
			},
		},
	}

	authz := &mockAuthorizer{allow: true}
	useCase := NewApproveInvoiceUseCase(repo, authz)

	principalAlpha := &Principal{
		ID:       "user_alice",
		TenantID: "tenant_alpha",
		Roles:    map[string]struct{}{"manager": {}},
	}
	ctxAlpha := InjectPrincipal(context.Background(), principalAlpha)

	t.Run("Allowed Decision -> Status Approved", func(t *testing.T) {
		authz.allow = true
		err := useCase.Execute(ctxAlpha, "inv_101")
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}

		if repo.invoices["inv_101"].Status != "approved" {
			t.Errorf("expected status 'approved', got '%s'", repo.invoices["inv_101"].Status)
		}
	})

	t.Run("PDP Denial -> ErrPermissionDenied", func(t *testing.T) {
		repo.invoices["inv_101"].Status = "draft" // reset
		authz.allow = false

		err := useCase.Execute(ctxAlpha, "inv_101")
		if !errors.Is(err, ErrPermissionDenied) {
			t.Errorf("expected ErrPermissionDenied, got: %v", err)
		}
	})

	t.Run("Cross-Tenant Isolation -> ErrResourceNotFound", func(t *testing.T) {
		principalBeta := &Principal{
			ID:       "user_charlie",
			TenantID: "tenant_beta", // Different tenant!
		}
		ctxBeta := InjectPrincipal(context.Background(), principalBeta)

		err := useCase.Execute(ctxBeta, "inv_101")
		if !errors.Is(err, ErrResourceNotFound) {
			t.Errorf("expected ErrResourceNotFound due to tenant isolation, got: %v", err)
		}
	})
}
