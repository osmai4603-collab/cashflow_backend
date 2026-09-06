package audit_test

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/platform/audit"
)

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	if audit.UserIDFromContext(ctx) != nil {
		t.Errorf("expected nil user ID on empty context")
	}
	if audit.CompanyIDFromContext(ctx) != nil {
		t.Errorf("expected nil company ID on empty context")
	}

	ctx = audit.WithUserID(ctx, 42)
	ctx = audit.WithCompanyID(ctx, 1)

	uid := audit.UserIDFromContext(ctx)
	if uid == nil || *uid != 42 {
		t.Errorf("expected user ID 42, got %v", uid)
	}

	cid := audit.CompanyIDFromContext(ctx)
	if cid == nil || *cid != 1 {
		t.Errorf("expected company ID 1, got %v", cid)
	}
}

func TestFields_Touch(t *testing.T) {
	ctx := audit.WithUserID(context.Background(), 100)
	f := audit.NewFields(ctx)

	if f.CreatedBy == nil || *f.CreatedBy != 100 {
		t.Errorf("expected CreatedBy 100")
	}
	if f.UpdatedBy == nil || *f.UpdatedBy != 100 {
		t.Errorf("expected UpdatedBy 100")
	}

	origUpdated := f.UpdatedAt
	time.Sleep(2 * time.Millisecond)

	ctx2 := audit.WithUserID(context.Background(), 200)
	f.Touch(ctx2)

	if !f.UpdatedAt.After(origUpdated) {
		t.Errorf("expected UpdatedAt to be after original timestamp")
	}
	if f.UpdatedBy == nil || *f.UpdatedBy != 200 {
		t.Errorf("expected UpdatedBy 200 after Touch")
	}
}
