package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type stubAuthorizer struct {
	err error
}

func (s stubAuthorizer) Check(context.Context, Subject, string, Action) error {
	return s.err
}

func TestRequireAccessRejectsUnauthenticatedRequest(t *testing.T) {
	handler := RequireAccess(stubAuthorizer{}, "sale.order", ActionRead)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/", nil))

	if record.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", record.Code, http.StatusUnauthorized)
	}
}

func TestRequireAccessReturnsForbiddenWhenDenied(t *testing.T) {
	handler := RequireAccess(stubAuthorizer{err: platformerrors.Forbidden("denied")}, "sale.order", ActionRead)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(WithClaims(request.Context(), &UserClaims{UserID: 7, CompanyID: 3}))
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, request)

	if record.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", record.Code, http.StatusForbidden)
	}
}

func TestRequireAccessCallsNextWhenAllowed(t *testing.T) {
	called := false
	handler := RequireAccess(stubAuthorizer{}, "sale.order", ActionRead)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(WithClaims(request.Context(), &UserClaims{UserID: 7, CompanyID: 3}))
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, request)

	if !called {
		t.Fatal("expected next handler to be called")
	}
}
