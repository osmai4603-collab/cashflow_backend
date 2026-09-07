package companyusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	companystorage "cashflow_backend/internal/adapters/storage/company"
	companyusecase "cashflow_backend/internal/usecase/company"
)

func setupTestUseCase() (*companyusecase.CompanyUseCase, *companystorage.MemoryRepo) {
	repo := companystorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := companyusecase.New(repo, logger)
	return uc, repo
}

func TestCompanyUseCase_CreateCompany(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	// 1. Success with defaults
	c, err := uc.CreateCompany(ctx, companyusecase.CreateCompanyInput{
		Name:       "Acme Corp",
		CurrencyID: 3,
		City:       "Dubai",
	})
	if err != nil {
		t.Fatalf("unexpected error creating company: %v", err)
	}
	if c.ID <= 0 {
		t.Errorf("expected positive company ID, got %d", c.ID)
	}
	if !c.Active {
		t.Errorf("expected company to be active")
	}
	if c.CurrencyID != 3 {
		t.Errorf("expected currency_id=3, got %d", c.CurrencyID)
	}

	// 2. Validation failure (empty name)
	_, err = uc.CreateCompany(ctx, companyusecase.CreateCompanyInput{
		Name:       "   ",
		CurrencyID: 3,
	})
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}

	// 3. Validation failure (currency required)
	_, err = uc.CreateCompany(ctx, companyusecase.CreateCompanyInput{Name: "No Currency"})
	if err == nil {
		t.Fatalf("expected error for missing currency, got nil")
	}
}

func TestCompanyUseCase_GetAndUpdateCompany(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	c, err := uc.CreateCompany(ctx, companyusecase.CreateCompanyInput{
		Name:       "Original Corp",
		CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Get by ID
	fetched, err := uc.GetCompany(ctx, c.ID)
	if err != nil {
		t.Fatalf("failed to get company: %v", err)
	}
	if fetched.Name != "Original Corp" {
		t.Errorf("expected Original Corp, got %q", fetched.Name)
	}

	// Update
	newName := "Renamed Corp"
	updated, err := uc.UpdateCompany(ctx, c.ID, companyusecase.UpdateCompanyInput{
		Name:    &newName,
		Country: stringPtr("UAE"),
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}
	if updated.Name != newName || updated.Country != "UAE" {
		t.Errorf("update fields mismatch")
	}

	// Invalid IDs
	_, err = uc.GetCompany(ctx, 0)
	if err == nil {
		t.Fatalf("expected error for id=0, got nil")
	}
	_, err = uc.GetCompany(ctx, 99999)
	if err == nil {
		t.Fatalf("expected error for nonexistent id")
	}
}

func TestCompanyUseCase_DeleteAndDefault(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	c, err := uc.CreateCompany(ctx, companyusecase.CreateCompanyInput{
		Name:       "To Be Deleted",
		CurrencyID: 2,
	})
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	if _, err := uc.CreateCompany(ctx, companyusecase.CreateCompanyInput{
		Name:       "Survivor",
		CurrencyID: 1,
	}); err != nil {
		t.Fatalf("failed to create second company: %v", err)
	}

	// Delete (soft)
	if err := uc.DeleteCompany(ctx, c.ID); err != nil {
		t.Fatalf("failed to delete company: %v", err)
	}
	_, err = uc.GetCompany(ctx, c.ID)
	if err == nil {
		t.Fatalf("expected not found after soft delete, got nil")
	}

	// GetDefaultCompany falls back to the remaining active company (id=2)
	def, err := uc.GetDefaultCompany(ctx)
	if err != nil {
		t.Fatalf("failed to get default company: %v", err)
	}
	if def.ID != 2 {
		t.Errorf("expected default company id=2, got %d", def.ID)
	}
}

func stringPtr(s string) *string {
	return &s
}
