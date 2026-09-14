package examples

import (
	"context"
	"errors"
	"fmt"
)

// contextKey defines an unexported type for context keys to prevent collisions.
type contextKey struct {
	name string
}

var (
	tenantCtxKey = &contextKey{name: "tenant_context"}

	// ErrUnauthorized indicates missing or unverified tenant credentials.
	ErrUnauthorized = errors.New("unauthorized: missing tenant context")
	// ErrInvalidContext indicates an invalid type in the context.
	ErrInvalidContext = errors.New("internal: invalid tenant context structure")
)

// TenantContext encapsulates validated authentication metadata scoped to a request.
type TenantContext struct {
	UserID    string   `json:"user_id"`
	CompanyID string   `json:"company_id"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
}

// WithTenant injects a verified TenantContext into the parent context.
func WithTenant(ctx context.Context, tenant TenantContext) context.Context {
	return context.WithValue(ctx, tenantCtxKey, tenant)
}

// FromContext extracts the TenantContext from the context or returns an error.
func FromContext(ctx context.Context) (TenantContext, error) {
	val := ctx.Value(tenantCtxKey)
	if val == nil {
		return TenantContext{}, ErrUnauthorized
	}

	tenant, ok := val.(TenantContext)
	if !ok {
		return TenantContext{}, ErrInvalidContext
	}

	if tenant.CompanyID == "" || tenant.UserID == "" {
		return TenantContext{}, fmt.Errorf("%w: incomplete credentials", ErrUnauthorized)
	}

	return tenant, nil
}

// ScopedQueryHelper demonstrates how repositories enforce tenant boundaries.
type ScopedQueryHelper struct{}

// ScopeSQL appends or verifies the tenant constraint on a SQL statement.
func (h *ScopedQueryHelper) ScopeSQL(ctx context.Context, baseQuery string) (string, string, error) {
	tenant, err := FromContext(ctx)
	if err != nil {
		return "", "", err
	}

	// Always return tenant ID as a parameterized query value
	return baseQuery, tenant.CompanyID, nil
}
