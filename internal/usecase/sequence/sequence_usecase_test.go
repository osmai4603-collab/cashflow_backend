package sequenceusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	sequencestorage "cashflow_backend/internal/adapters/storage/sequence"
	"cashflow_backend/internal/domain/sequence"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"
)

func setupTestUseCase() (*sequenceusecase.SequenceUseCase, *sequencestorage.MemoryRepo) {
	repo := sequencestorage.NewMemoryRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := sequenceusecase.New(repo, logger)
	return uc, repo
}

func TestSequenceUseCase_CreateSequence(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	// 1. Success with defaults
	s, err := uc.CreateSequence(ctx, sequenceusecase.CreateSequenceInput{
		Name:   "Invoice Sequence",
		Code:   "sale.invoice",
		Prefix: "INV/",
	})
	if err != nil {
		t.Fatalf("unexpected error creating sequence: %v", err)
	}
	if s.ID <= 0 {
		t.Errorf("expected positive sequence ID, got %d", s.ID)
	}
	if s.Padding != 5 {
		t.Errorf("expected default padding 5, got %d", s.Padding)
	}
	if s.IncrementBy != 1 {
		t.Errorf("expected default increment_by 1, got %d", s.IncrementBy)
	}
	if s.StartNumber != 1 {
		t.Errorf("expected default start_number 1, got %d", s.StartNumber)
	}
	if s.SequenceType != sequence.SequenceTypeNormal {
		t.Errorf("expected default type normal, got %v", s.SequenceType)
	}

	// 2. Date-range sequence
	sr, err := uc.CreateSequence(ctx, sequenceusecase.CreateSequenceInput{
		Name:         "Sales Order",
		Code:         "sale.order",
		Prefix:       "SO/%(year)s/",
		Padding:      4,
		SequenceType: sequence.SequenceTypeDateRange,
		DateRange:    sequence.DateRangeYear,
	})
	if err != nil {
		t.Fatalf("unexpected error creating date-range sequence: %v", err)
	}
	if !sr.Active {
		t.Errorf("expected sequence to be active")
	}

	// 3. Validation failure (empty name)
	_, err = uc.CreateSequence(ctx, sequenceusecase.CreateSequenceInput{Code: "x"})
	if err == nil {
		t.Fatalf("expected error for empty name, got nil")
	}

	// 4. Validation failure (invalid date range granularity)
	_, err = uc.CreateSequence(ctx, sequenceusecase.CreateSequenceInput{
		Name:         "Bad",
		Code:         "bad",
		SequenceType: sequence.SequenceTypeDateRange,
		DateRange:    "century",
	})
	if err == nil {
		t.Fatalf("expected error for invalid date range, got nil")
	}
}

func TestSequenceUseCase_GenerateNext(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	s, err := uc.CreateSequence(ctx, sequenceusecase.CreateSequenceInput{
		Name:   "Invoice Sequence",
		Code:   "sale.invoice",
		Prefix: "INV/",
		Suffix: "/%(year)s",
	})
	if err != nil {
		t.Fatalf("failed to create sequence: %v", err)
	}

	ref, err := uc.GenerateNextByID(ctx, s.ID, time.Date(2024, 5, 3, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("unexpected error generating next: %v", err)
	}
	if ref != "INV/00001/2024" {
		t.Errorf("expected INV/00001/2024, got %q", ref)
	}

	ref2, err := uc.GenerateNextByID(ctx, s.ID, time.Now())
	if err != nil {
		t.Fatalf("unexpected error generating next: %v", err)
	}
	if ref2 != "INV/00002/2024" && ref2 != "INV/00002/2025" && ref2 != "INV/00002/2026" {
		t.Errorf("expected INV/00002/<year>, got %q", ref2)
	}

	// GenerateNext by code
	ref3, err := uc.GenerateNext(ctx, "sale.invoice", time.Now())
	if err != nil {
		t.Fatalf("unexpected error generating next by code: %v", err)
	}
	if ref3 != "INV/00003/2024" && ref3 != "INV/00003/2025" && ref3 != "INV/00003/2026" {
		t.Errorf("expected INV/00003/<year>, got %q", ref3)
	}

	// Unknown code
	_, err = uc.GenerateNext(ctx, "nope", time.Now())
	if err == nil {
		t.Fatalf("expected error for unknown code, got nil")
	}
}

func TestSequenceUseCase_UpdateAndDelete(t *testing.T) {
	uc, _ := setupTestUseCase()
	ctx := context.Background()

	s, err := uc.CreateSequence(ctx, sequenceusecase.CreateSequenceInput{
		Name:   "Original",
		Code:   "orig",
		Prefix: "O/",
	})
	if err != nil {
		t.Fatalf("failed to create sequence: %v", err)
	}

	// Update
	newPrefix := "NEW/"
	upd, err := uc.UpdateSequence(ctx, s.ID, sequenceusecase.UpdateSequenceInput{
		Name:   stringPtr("Renamed"),
		Prefix: &newPrefix,
	})
	if err != nil {
		t.Fatalf("failed to update sequence: %v", err)
	}
	if upd.Prefix != "NEW/" || upd.Name != "Renamed" {
		t.Errorf("update fields mismatch")
	}

	// Delete (soft)
	if err := uc.DeleteSequence(ctx, s.ID); err != nil {
		t.Fatalf("failed to delete sequence: %v", err)
	}
	_, err = uc.GetSequence(ctx, s.ID)
	if err == nil {
		t.Fatalf("expected not found after soft delete, got nil")
	}

	// Invalid id
	_, err = uc.GetSequence(ctx, 0)
	if err == nil {
		t.Fatalf("expected error for id=0, got nil")
	}

	// List returns nothing
	items, err := uc.ListSequences(ctx)
	if err != nil {
		t.Fatalf("failed to list sequences: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 sequences after delete, got %d", len(items))
	}
}

func stringPtr(s string) *string {
	return &s
}
