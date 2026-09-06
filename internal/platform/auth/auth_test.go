package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/auth"
)

const testSecret = "test-secret-key-1234567890"

func TestToken_Lifecycle(t *testing.T) {
	token, err := auth.GenerateToken(42, 100, []string{"admin", "accountant"}, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := auth.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}
	if claims.CompanyID != 100 {
		t.Errorf("expected CompanyID 100, got %d", claims.CompanyID)
	}
	if !claims.HasRole("admin") {
		t.Errorf("expected claims to have role admin")
	}
	if !claims.HasRole("accountant") {
		t.Errorf("expected claims to have role accountant")
	}
	if claims.HasRole("viewer") {
		t.Errorf("did not expect claims to have role viewer")
	}
}

func TestToken_Expired(t *testing.T) {
	token, err := auth.GenerateToken(1, 1, []string{"user"}, testSecret, -time.Minute)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	_, err = auth.ValidateToken(token, testSecret)
	if err != auth.ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestToken_InvalidSecret(t *testing.T) {
	token, err := auth.GenerateToken(1, 1, []string{"user"}, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	_, err = auth.ValidateToken(token, "different-wrong-secret")
	if err != auth.ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestMiddleware_SuccessAndAuditContext(t *testing.T) {
	token, err := auth.GenerateToken(77, 202, []string{"manager"}, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var capturedUID *int64
	var capturedCID *int64

	handler := auth.Middleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUID = audit.UserIDFromContext(r.Context())
		capturedCID = audit.CompanyIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if capturedUID == nil || *capturedUID != 77 {
		t.Errorf("expected captured UserID 77, got %v", capturedUID)
	}
	if capturedCID == nil || *capturedCID != 202 {
		t.Errorf("expected captured CompanyID 202, got %v", capturedCID)
	}
}

func TestMiddleware_MissingToken(t *testing.T) {
	handler := auth.Middleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireRole(t *testing.T) {
	token, _ := auth.GenerateToken(1, 1, []string{"user"}, testSecret, time.Hour)

	adminRoute := auth.Middleware(testSecret)(
		auth.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	adminRoute.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for non-admin, got %d", rec.Code)
	}
}
