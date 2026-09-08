package expensestorage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cashflow_backend/internal/domain/expense"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MemoryRepo is a deterministic repository for unit tests and local development.
type MemoryRepo struct {
	mu       sync.RWMutex
	expenses map[int64]*expense.Expense
	nextID   int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{expenses: make(map[int64]*expense.Expense)}
}

func cloneExpense(value *expense.Expense) *expense.Expense {
	clone := *value
	clone.AttachmentIDs = append([]int64(nil), value.AttachmentIDs...)
	clone.AttachmentChecksums = append([]string(nil), value.AttachmentChecksums...)
	clone.TaxIDs = append([]int64(nil), value.TaxIDs...)
	return &clone
}

func (r *MemoryRepo) Create(_ context.Context, value *expense.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := value.Validate(); err != nil {
		return err
	}
	r.nextID++
	value.ID = r.nextID
	value.CreatedAt = time.Now().UTC()
	value.UpdatedAt = value.CreatedAt
	r.expenses[value.ID] = cloneExpense(value)
	return nil
}

func (r *MemoryRepo) Update(_ context.Context, value *expense.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.expenses[value.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("expense with ID %d not found", value.ID))
	}
	if err := value.Validate(); err != nil {
		return err
	}
	value.CreatedAt = existing.CreatedAt
	value.UpdatedAt = time.Now().UTC()
	r.expenses[value.ID] = cloneExpense(value)
	return nil
}

func (r *MemoryRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.expenses[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("expense with ID %d not found", id))
	}
	delete(r.expenses, id)
	return nil
}

func (r *MemoryRepo) GetByID(_ context.Context, id int64) (*expense.Expense, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.expenses[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("expense with ID %d not found", id))
	}
	return cloneExpense(value), nil
}

func (r *MemoryRepo) List(_ context.Context, filter expense.Filter) ([]*expense.Expense, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*expense.Expense, 0)
	for _, value := range r.expenses {
		if !matchesFilter(value, filter) {
			continue
		}
		result = append(result, cloneExpense(value))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date.Equal(result[j].Date) {
			return result[i].ID > result[j].ID
		}
		return result[i].Date.After(result[j].Date)
	})
	start := filter.Offset
	if start < 0 {
		start = 0
	}
	if start >= len(result) {
		return []*expense.Expense{}, nil
	}
	end := len(result)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}
	return result[start:end], nil
}

func (r *MemoryRepo) GetDuplicateExpenses(_ context.Context, value *expense.Expense) ([]*expense.Expense, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*expense.Expense, 0)
	for _, candidate := range r.expenses {
		if candidate.ID == value.ID || candidate.EmployeeID != value.EmployeeID || !sameDate(candidate.Date, value.Date) || candidate.TotalAmount != value.TotalAmount || candidate.State == expense.StateRefused {
			continue
		}
		result = append(result, cloneExpense(candidate))
	}
	return result, nil
}

func matchesFilter(value *expense.Expense, filter expense.Filter) bool {
	if filter.EmployeeID != nil && value.EmployeeID != *filter.EmployeeID || filter.ManagerID != nil && !sameID(value.ManagerID, *filter.ManagerID) || filter.State != nil && value.State != *filter.State || filter.DepartmentID != nil && !sameID(value.DepartmentID, *filter.DepartmentID) || filter.CompanyID != nil && value.CompanyID != *filter.CompanyID {
		return false
	}
	if filter.DateFrom != nil && value.Date.Format("2006-01-02") < *filter.DateFrom || filter.DateTo != nil && value.Date.Format("2006-01-02") > *filter.DateTo {
		return false
	}
	return true
}

func sameID(value *int64, expected int64) bool { return value != nil && *value == expected }
func sameDate(a, b time.Time) bool             { return a.Format("2006-01-02") == b.Format("2006-01-02") }
