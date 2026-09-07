package auth

import (
	"context"
	"testing"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type countingAuthorizer struct {
	calls int
	err   error
}

func (a *countingAuthorizer) Check(context.Context, Subject, string, Action) error {
	a.calls++
	return a.err
}

func TestCachedAuthorizerCachesAndInvalidatesDecision(t *testing.T) {
	delegate := &countingAuthorizer{}
	cached := NewCachedAuthorizer(delegate)
	subject := Subject{UserID: 1, CompanyID: 2}

	if err := cached.Check(context.Background(), subject, "res.partner", ActionRead); err != nil {
		t.Fatalf("first check: %v", err)
	}
	if err := cached.Check(context.Background(), subject, "res.partner", ActionRead); err != nil {
		t.Fatalf("cached check: %v", err)
	}
	if delegate.calls != 1 {
		t.Fatalf("delegate calls = %d, want 1", delegate.calls)
	}

	cached.Invalidate()
	if err := cached.Check(context.Background(), subject, "res.partner", ActionRead); err != nil {
		t.Fatalf("check after invalidation: %v", err)
	}
	if delegate.calls != 2 {
		t.Fatalf("delegate calls after invalidation = %d, want 2", delegate.calls)
	}
}

func TestCachedAuthorizerCachesDenialsAsForbidden(t *testing.T) {
	delegate := &countingAuthorizer{err: platformerrors.Forbidden("insufficient permissions")}
	cached := NewCachedAuthorizer(delegate)
	subject := Subject{UserID: 1, CompanyID: 2}

	if err := cached.Check(context.Background(), subject, "res.partner", ActionWrite); err == nil {
		t.Fatal("expected first denial")
	}
	if err := cached.Check(context.Background(), subject, "res.partner", ActionWrite); err == nil || platformerrors.HTTPStatus(err) != 403 {
		t.Fatalf("expected cached 403, got %v", err)
	}
	if delegate.calls != 1 {
		t.Fatalf("delegate calls = %d, want 1", delegate.calls)
	}
}
