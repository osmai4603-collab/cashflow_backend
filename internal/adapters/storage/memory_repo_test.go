package storage_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cashflow_backend/internal/adapters/storage"
	"cashflow_backend/internal/domain"
)

func TestMemoryTransactionRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := storage.NewMemoryTransactionRepo()

	now := time.Now().UTC()
	tx, err := domain.NewTransaction("tx-100", 250.0, domain.TypeIncome, "Test Deposit", now)
	if err != nil {
		t.Fatalf("unexpected error creating tx: %v", err)
	}

	// Save
	if err := repo.Save(ctx, tx); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// FindByID
	found, err := repo.FindByID(ctx, "tx-100")
	if err != nil {
		t.Fatalf("failed to find tx: %v", err)
	}
	if found.Amount != 250.0 || found.Description != "Test Deposit" {
		t.Errorf("unexpected found tx: %+v", found)
	}

	// FindAll
	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("failed to find all: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 item, got %d", len(all))
	}

	// Not found
	_, err = repo.FindByID(ctx, "non-existent")
	if !errors.Is(err, domain.ErrTransactionNotFound) {
		t.Errorf("expected ErrTransactionNotFound, got %v", err)
	}
}

func TestMemoryTransactionRepo_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := storage.NewMemoryTransactionRepo()
	var wg sync.WaitGroup

	// Concurrently write 50 items
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tx, _ := domain.NewTransaction(
				string(rune('a'+id)),
				float64(id+10),
				domain.TypeIncome,
				"concurrent",
				time.Now().UTC(),
			)
			_ = repo.Save(ctx, tx)
		}(i)
	}

	// Concurrently read all
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = repo.FindAll(ctx)
		}()
	}

	wg.Wait()

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("expected items in repo, got 0")
	}
}

func TestMemoryTransactionRepo_Close(t *testing.T) {
	ctx := context.Background()
	repo := storage.NewMemoryTransactionRepo()

	if err := repo.Ping(ctx); err != nil {
		t.Fatalf("expected ping to succeed, got: %v", err)
	}

	if err := repo.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	if err := repo.Ping(ctx); !errors.Is(err, storage.ErrStorageClosed) {
		t.Errorf("expected ErrStorageClosed on ping, got %v", err)
	}

	tx, _ := domain.NewTransaction("tx-1", 10, domain.TypeExpense, "test", time.Now().UTC())
	if err := repo.Save(ctx, tx); !errors.Is(err, storage.ErrStorageClosed) {
		t.Errorf("expected ErrStorageClosed on save, got %v", err)
	}
}
