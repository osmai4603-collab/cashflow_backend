package expensestorage_test

import (
	"context"
	"testing"
	"time"

	expensestorage "cashflow_backend/internal/adapters/storage/expense"
	domain "cashflow_backend/internal/domain/expense"
)

func TestMemoryRepoFiltersAndCopiesExpenseCollections(t *testing.T) {
	repo := expensestorage.NewMemoryRepo()
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	value := &domain.Expense{
		Name:                "Hotel",
		Date:                date,
		EmployeeID:          7,
		CompanyID:           2,
		TotalAmount:         100,
		AttachmentIDs:       []int64{4},
		AttachmentChecksums: []string{"abc"},
		TaxIDs:              []int64{3},
	}
	if err := repo.Create(context.Background(), value); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	value.AttachmentIDs[0] = 99
	found, err := repo.GetByID(context.Background(), value.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if found.AttachmentIDs[0] != 4 || found.TaxIDs[0] != 3 {
		t.Fatalf("repository did not protect collection fields: %+v", found)
	}

	companyID := int64(2)
	items, err := repo.List(context.Background(), domain.Filter{CompanyID: &companyID, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != value.ID {
		t.Fatalf("List() = %+v, want one matching expense", items)
	}
}

func TestMemoryRepoFindsDuplicatesByEmployeeDateAndAmount(t *testing.T) {
	repo := expensestorage.NewMemoryRepo()
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	first := &domain.Expense{Name: "Taxi", Date: date, EmployeeID: 7, TotalAmount: 25}
	second := &domain.Expense{Name: "Taxi again", Date: date.Add(12 * time.Hour), EmployeeID: 7, TotalAmount: 25}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	duplicates, err := repo.GetDuplicateExpenses(context.Background(), second)
	if err != nil {
		t.Fatalf("GetDuplicateExpenses() error = %v", err)
	}
	if len(duplicates) != 1 || duplicates[0].ID != first.ID {
		t.Fatalf("duplicates = %+v, want first expense", duplicates)
	}
}
