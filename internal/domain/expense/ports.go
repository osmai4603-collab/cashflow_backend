package expense

import (
	"context"
)

// Repository defines the persistence operations for Expense.
type Repository interface {
	Create(ctx context.Context, e *Expense) error
	Update(ctx context.Context, e *Expense) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Expense, error)
	List(ctx context.Context, filter Filter) ([]*Expense, error)
	GetDuplicateExpenses(ctx context.Context, e *Expense) ([]*Expense, error)
}

// Filter defines search criteria for Expenses.
type Filter struct {
	EmployeeID   *int64
	ManagerID    *int64
	State        *ExpenseState
	DepartmentID *int64
	CompanyID    *int64
	DateFrom     *string
	DateTo       *string
	Limit        int
	Offset       int
}

// Usecase defines the business logic operations for Expense.
type Usecase interface {
	CreateExpense(ctx context.Context, e *Expense) error
	SubmitExpense(ctx context.Context, id int64) error
	ApproveExpense(ctx context.Context, id int64, approverID int64) error
	RefuseExpense(ctx context.Context, id int64, approverID int64, reason string) error
	PostExpense(ctx context.Context, id int64) error
	SplitExpense(ctx context.Context, req ExpenseSplitRequest) ([]int64, error)
	GetExpense(ctx context.Context, id int64) (*Expense, error)
	ListExpenses(ctx context.Context, filter Filter) ([]*Expense, error)
}
