package partnerusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/platform/audit"
	partnerusecase "cashflow_backend/internal/usecase/partner"
)

func setupTestUseCase() (*partnerusecase.PartnerUseCase, *partnerstorage.MemoryRepo) {
	repo := partnerstorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := partnerusecase.New(repo, logger)
	return uc, repo
}

func TestPartnerUseCase_CreatePartner(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := audit.WithUserID(context.Background(), 42)

	// 1. Success with defaults
	in := partnerusecase.CreatePartnerInput{
		Name:  "Ahmed Al-Otaibi",
		Email: "ahmed@example.com",
	}

	p, err := uc.CreatePartner(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error creating partner: %v", err)
	}

	if p.ID <= 0 {
		t.Errorf("expected positive partner ID, got %d", p.ID)
	}
	if p.Type != partner.PartnerTypeIndividual {
		t.Errorf("expected default type individual, got %v", p.Type)
	}
	if !p.IsCustomer {
		t.Errorf("expected default is_customer=true")
	}
	if p.IsSupplier {
		t.Errorf("expected default is_supplier=false")
	}
	if p.Audit.CreatedBy == nil || *p.Audit.CreatedBy != 42 {
		t.Errorf("expected created_by=42 from context")
	}

	// 2. Validation failure (empty name)
	_, err = uc.CreatePartner(ctx, partnerusecase.CreatePartnerInput{Name: "   "})
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}

	// 3. Non-existent parent_id
	fakeParentID := int64(9999)
	_, err = uc.CreatePartner(ctx, partnerusecase.CreatePartnerInput{
		Name:     "Child Partner",
		ParentID: &fakeParentID,
	})
	if err == nil {
		t.Fatalf("expected error for non-existent parent_id, got nil")
	}
}

func TestPartnerUseCase_GetAndUpdatePartner(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := audit.WithUserID(context.Background(), 100)

	// Create company partner
	p, err := uc.CreatePartner(ctx, partnerusecase.CreatePartnerInput{
		Name: "Original Tech Corp",
		Type: partner.PartnerTypeCompany,
	})
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Get by ID
	fetched, err := uc.GetPartner(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get partner: %v", err)
	}
	if fetched.Name != "Original Tech Corp" {
		t.Errorf("expected Original Tech Corp, got %q", fetched.Name)
	}

	// Update partner
	newName := "Updated Tech Corp"
	newCity := "Riyadh"
	updated, err := uc.UpdatePartner(ctx, p.ID, partnerusecase.UpdatePartnerInput{
		Name: &newName,
		City: &newCity,
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}
	if updated.Name != newName || updated.City != newCity {
		t.Errorf("update fields mismatch")
	}
	if updated.Audit.UpdatedBy == nil || *updated.Audit.UpdatedBy != 100 {
		t.Errorf("expected updated_by=100 from context")
	}

	// Self-parent circular validation error
	selfParent := p.ID
	_, err = uc.UpdatePartner(ctx, p.ID, partnerusecase.UpdatePartnerInput{
		ParentID: &selfParent,
	})
	if err == nil {
		t.Fatalf("expected circular parent error, got nil")
	}

	// Invalid ID
	_, err = uc.GetPartner(ctx, 0)
	if err == nil {
		t.Fatalf("expected error for id=0, got nil")
	}
}

func TestPartnerUseCase_DeletePartner(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	p, err := uc.CreatePartner(ctx, partnerusecase.CreatePartnerInput{Name: "To Be Deleted"})
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	if err := uc.DeletePartner(ctx, p.ID); err != nil {
		t.Fatalf("failed to delete partner: %v", err)
	}

	// Should not be found after soft delete
	_, err = uc.GetPartner(ctx, p.ID)
	if err == nil {
		t.Fatalf("expected not found error after soft delete, got nil")
	}
}
