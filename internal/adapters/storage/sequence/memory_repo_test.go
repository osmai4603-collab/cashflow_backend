package sequencestorage_test

import (
	"context"
	"sync"
	"testing"

	sequencestorage "cashflow_backend/internal/adapters/storage/sequence"
	"cashflow_backend/internal/domain/sequence"
)

func TestSequenceMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := sequencestorage.NewMemoryRepo()

	s := &sequence.Sequence{
		Name:         "Invoice",
		Code:         "sale.invoice",
		Prefix:       "INV/",
		Padding:      5,
		IncrementBy:  1,
		StartNumber:  1,
		SequenceType: sequence.SequenceTypeNormal,
	}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("unexpected error creating sequence: %v", err)
	}
	if s.ID <= 0 || !s.Active {
		t.Errorf("expected positive id and active=true")
	}

	fetched, err := repo.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching sequence: %v", err)
	}
	if fetched.Code != "sale.invoice" {
		t.Errorf("expected code sale.invoice, got %q", fetched.Code)
	}

	// By code (case-insensitive)
	byCode, err := repo.GetByCode(ctx, "SALE.INVOICE")
	if err != nil {
		t.Fatalf("unexpected error fetching by code: %v", err)
	}
	if byCode.ID != s.ID {
		t.Errorf("expected ID %d, got %d", s.ID, byCode.ID)
	}

	// Update
	newName := "Customer Invoice"
	if err := repo.Update(ctx, &sequence.Sequence{ID: s.ID, Name: newName, Code: fetched.Code, Padding: 5, IncrementBy: 1, StartNumber: 1, SequenceType: sequence.SequenceTypeNormal, Active: true}); err != nil {
		t.Fatalf("unexpected error updating sequence: %v", err)
	}
	updated, err := repo.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error re-fetching: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("expected updated name %q, got %q", newName, updated.Name)
	}

	// Delete
	if err := repo.Delete(ctx, s.ID); err != nil {
		t.Fatalf("unexpected error deleting sequence: %v", err)
	}
	_, err = repo.GetByID(ctx, s.ID)
	if err == nil {
		t.Errorf("expected error getting soft-deleted sequence, got nil")
	}
}

func TestSequenceMemoryRepo_NextValue(t *testing.T) {
	ctx := context.Background()
	repo := sequencestorage.NewMemoryRepo()

	s := &sequence.Sequence{
		Name:         "Sales Order",
		Code:         "sale.order",
		Prefix:       "SO/",
		Padding:      5,
		IncrementBy:  2,
		StartNumber:  10,
		SequenceType: sequence.SequenceTypeNormal,
	}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("failed to create sequence: %v", err)
	}

	// First value uses StartNumber
	next, err := repo.NextValue(ctx, s.ID)
	if err != nil {
		t.Fatalf("unexpected error getting next value: %v", err)
	}
	if next != 10 {
		t.Errorf("expected first value 10, got %d", next)
	}

	// Subsequent values increment by IncrementBy
	next, _ = repo.NextValue(ctx, s.ID)
	if next != 12 {
		t.Errorf("expected second value 12, got %d", next)
	}

	// Reset returns to start - increment
	if err := repo.Reset(ctx, s.ID); err != nil {
		t.Fatalf("unexpected error resetting sequence: %v", err)
	}
	next, _ = repo.NextValue(ctx, s.ID)
	if next != 10 {
		t.Errorf("expected value 10 after reset, got %d", next)
	}

	// Unknown id
	_, err = repo.NextValue(ctx, 9999)
	if err == nil {
		t.Errorf("expected error for unknown sequence id, got nil")
	}
}

func TestSequenceMemoryRepo_NextValueConcurrentNoDuplicates(t *testing.T) {
	ctx := context.Background()
	repo := sequencestorage.NewMemoryRepo()

	s := &sequence.Sequence{
		Name:         "Concurrent Invoice",
		Code:         "conc.invoice",
		Padding:      5,
		IncrementBy:  1,
		StartNumber:  1,
		SequenceType: sequence.SequenceTypeNormal,
	}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("failed to create sequence: %v", err)
	}

	const numGoroutines = 50
	results := make([]int, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			next, err := repo.NextValue(ctx, s.ID)
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", i, err)
				return
			}
			results[i] = next
		}(i)
	}
	wg.Wait()

	// Every issued number must be unique and within 1..numGoroutines
	seen := make(map[int]bool)
	for i, got := range results {
		if got < 1 || got > numGoroutines {
			t.Errorf("goroutine %d: value %d out of range", i, got)
		}
		if seen[got] {
			t.Errorf("duplicate sequence value %d issued", got)
		}
		seen[got] = true
	}

	if len(seen) != numGoroutines {
		t.Errorf("expected %d unique values, got %d", numGoroutines, len(seen))
	}
}

func TestSequenceMemoryRepo_List(t *testing.T) {
	ctx := context.Background()
	repo := sequencestorage.NewMemoryRepo()

	for i := 0; i < 3; i++ {
		s := &sequence.Sequence{Name: "S", Code: "c", Padding: 5, IncrementBy: 1, StartNumber: 1}
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("failed to create sequence: %v", err)
		}
	}

	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing sequences: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 sequences, got %d", len(items))
	}
}
