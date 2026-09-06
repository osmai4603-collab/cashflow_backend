package audit

import (
	"context"
	"time"
)

type contextKey string

const (
	userIDKey    contextKey = "current_user_id"
	companyIDKey contextKey = "current_company_id"
)

// Fields provides standardized audit tracking embedded in ERP domain entities.
type Fields struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
}

// NewFields initializes audit fields for newly created entities.
func NewFields(ctx context.Context) Fields {
	now := time.Now().UTC()
	uid := UserIDFromContext(ctx)
	return Fields{
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: uid,
		UpdatedBy: uid,
	}
}

// Touch updates the modification timestamp and modifier ID.
func (f *Fields) Touch(ctx context.Context) {
	f.UpdatedAt = time.Now().UTC()
	if uid := UserIDFromContext(ctx); uid != nil {
		f.UpdatedBy = uid
	}
}

// WithUserID injects the current authenticated user ID into the context.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext retrieves the user ID from context if present.
func UserIDFromContext(ctx context.Context) *int64 {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(userIDKey).(int64); ok {
		return &v
	}
	return nil
}

// WithCompanyID injects the current active company/tenant ID into context.
func WithCompanyID(ctx context.Context, companyID int64) context.Context {
	return context.WithValue(ctx, companyIDKey, companyID)
}

// CompanyIDFromContext retrieves the active company ID from context if present.
func CompanyIDFromContext(ctx context.Context) *int64 {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(companyIDKey).(int64); ok {
		return &v
	}
	return nil
}
