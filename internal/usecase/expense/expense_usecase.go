package expenseusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/expense"
	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// UseCase implements the expense workflow independently from HTTP and storage.
type UseCase struct {
	repo              expense.Repository
	hrRepo            hr.Repository
	logger            *slog.Logger
	accounting        *AccountingIntegration
	activityScheduler interface {
		Schedule(context.Context, *activity.Activity) error
	}
	activityTypeID int64
}

func New(repo expense.Repository, hrRepo hr.Repository, logger *slog.Logger, integrations ...*AccountingIntegration) *UseCase {
	if logger == nil {
		logger = slog.Default()
	}
	var integration *AccountingIntegration
	if len(integrations) > 0 {
		integration = integrations[0]
	}
	return &UseCase{repo: repo, hrRepo: hrRepo, logger: logger, accounting: integration}
}

// ConfigureActivityScheduler enables manager approval activities after submission.
func (u *UseCase) ConfigureActivityScheduler(scheduler interface {
	Schedule(context.Context, *activity.Activity) error
}, activityTypeID int64) {
	u.activityScheduler = scheduler
	u.activityTypeID = activityTypeID
}

func (u *UseCase) CreateExpense(ctx context.Context, value *expense.Expense) error {
	if value == nil {
		return platformerrors.Validation("expense is required", nil)
	}
	if value.Date.IsZero() {
		value.Date = time.Now().UTC()
	}
	employee, err := u.hrRepo.GetEmployeeByID(ctx, value.EmployeeID)
	if err != nil {
		return platformerrors.Validation("employee not found", map[string]string{"employee_id": fmt.Sprint(value.EmployeeID)})
	}
	if value.DepartmentID == nil {
		value.DepartmentID = employee.DepartmentID
	}
	if value.ManagerID == nil {
		value.ManagerID = employee.ExpenseManagerID
	}
	if err := value.Validate(); err != nil {
		return err
	}
	return u.repo.Create(ctx, value)
}

func (u *UseCase) GetExpense(ctx context.Context, id int64) (*expense.Expense, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *UseCase) ListExpenses(ctx context.Context, filter expense.Filter) ([]*expense.Expense, error) {
	return u.repo.List(ctx, filter)
}

func (u *UseCase) UpdateExpense(ctx context.Context, value *expense.Expense) error {
	if value == nil {
		return platformerrors.Validation("expense is required", nil)
	}
	current, err := u.repo.GetByID(ctx, value.ID)
	if err != nil {
		return err
	}
	if !isEditable(current.State) {
		return platformerrors.Conflict("only draft, submitted, and approved expenses can be edited")
	}
	return u.repo.Update(ctx, value)
}

func (u *UseCase) DeleteExpense(ctx context.Context, id int64) error {
	value, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if value.State != expense.StateDraft && value.State != expense.StateRefused {
		return platformerrors.Conflict("only draft or refused expenses can be deleted")
	}
	return u.repo.Delete(ctx, id)
}

func (u *UseCase) SubmitExpense(ctx context.Context, id int64) error {
	value, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if value.State != expense.StateDraft && value.State != expense.StateRefused {
		return platformerrors.Conflict("only draft or refused expenses can be submitted")
	}
	if err := value.Validate(); err != nil {
		return err
	}
	employee, err := u.hrRepo.GetEmployeeByID(ctx, value.EmployeeID)
	if err != nil {
		return err
	}
	if value.ManagerID == nil {
		value.ManagerID = employee.ExpenseManagerID
	}
	if value.ManagerID == nil {
		value.State = expense.StateApproved
		now := time.Now().UTC()
		value.ApprovalDate = &now
	} else {
		value.State = expense.StateSubmitted
	}
	if err := u.repo.Update(ctx, value); err != nil {
		return err
	}
	if value.State == expense.StateSubmitted && u.activityScheduler != nil {
		if value.CompanyID <= 0 || u.activityTypeID <= 0 {
			return platformerrors.Validation("expense approval activity configuration is missing", nil)
		}
		activityValue := &activity.Activity{
			ActivityTypeID: u.activityTypeID,
			Summary:        "Approve expense: " + value.Name,
			DateDeadline:   time.Now().UTC(),
			AssignedUserID: *value.ManagerID,
			ResModel:       "hr.expense",
			ResID:          &value.ID,
			Active:         true,
			CompanyID:      value.CompanyID,
		}
		if err := u.activityScheduler.Schedule(ctx, activityValue); err != nil {
			return platformerrors.Internal("failed to schedule expense approval activity", err)
		}
	}
	return nil
}

func (u *UseCase) ApproveExpense(ctx context.Context, id int64, approverID int64) error {
	value, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if value.State != expense.StateSubmitted {
		return platformerrors.Conflict("only submitted expenses can be approved")
	}
	if approverID <= 0 || value.ManagerID == nil || *value.ManagerID != approverID {
		return platformerrors.Forbidden("only the assigned expense manager can approve this expense")
	}
	duplicates, err := u.repo.GetDuplicateExpenses(ctx, value)
	if err != nil {
		return err
	}
	if len(duplicates) > 0 {
		return platformerrors.Conflict("a duplicate expense or receipt was found")
	}
	now := time.Now().UTC()
	value.State = expense.StateApproved
	value.ApprovalDate = &now
	return u.repo.Update(ctx, value)
}

func (u *UseCase) RefuseExpense(ctx context.Context, id int64, approverID int64, reason string) error {
	value, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if value.State != expense.StateSubmitted && value.State != expense.StateApproved {
		return platformerrors.Conflict("only submitted or approved expenses can be refused")
	}
	if approverID <= 0 || value.ManagerID == nil || *value.ManagerID != approverID {
		return platformerrors.Forbidden("only the assigned expense manager can refuse this expense")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return platformerrors.Validation("refusal reason is required", map[string]string{"reason": "cannot be empty"})
	}
	value.State = expense.StateRefused
	value.RefuseReason = reason
	return u.repo.Update(ctx, value)
}

func (u *UseCase) ResetExpense(ctx context.Context, id int64) error {
	value, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if value.State != expense.StateRefused {
		return platformerrors.Conflict("only refused expenses can be reset")
	}
	value.State = expense.StateDraft
	value.RefuseReason = ""
	return u.repo.Update(ctx, value)
}

func (u *UseCase) PostExpense(_ context.Context, _ int64) error {
	return platformerrors.Conflict("expense accounting integration is not available until the accounting phase")
}

func (u *UseCase) PostExpenseWithAccounting(ctx context.Context, id int64, input PostExpenseInput) error {
	if u.accounting == nil {
		return platformerrors.Conflict("expense accounting integration is not configured")
	}
	value, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if value.AccountMoveID != nil {
		return nil
	}
	move, err := u.accounting.PostExpense(ctx, value, input)
	if err != nil {
		return err
	}
	value.AccountMoveID = &move.ID
	if value.PaymentMode == expense.PaymentCompanyAccount {
		value.State = expense.StatePaid
	} else {
		value.State = expense.StatePosted
	}
	return u.repo.Update(ctx, value)
}

func (u *UseCase) SplitExpense(ctx context.Context, request expense.ExpenseSplitRequest) ([]int64, error) {
	value, err := u.repo.GetByID(ctx, request.ExpenseID)
	if err != nil {
		return nil, err
	}
	if value.State != expense.StateDraft && value.State != expense.StateRefused {
		return nil, platformerrors.Conflict("only draft or refused expenses can be split")
	}
	if len(request.Splits) < 2 {
		return nil, platformerrors.Validation("at least two split lines are required", nil)
	}
	var total float64
	for i := range request.Splits {
		line := &request.Splits[i]
		line.Name = strings.TrimSpace(line.Name)
		if line.Name == "" || line.TotalAmount <= 0 {
			return nil, platformerrors.Validation("split lines must have a name and positive amount", nil)
		}
		total += line.TotalAmount
	}
	if abs(total-value.TotalAmount) > 0.00005 {
		return nil, platformerrors.Validation("split lines must equal the original total", nil)
	}
	ids := make([]int64, 0, len(request.Splits))
	for _, line := range request.Splits {
		child := &expense.Expense{
			Name: line.Name, Date: value.Date, EmployeeID: value.EmployeeID, ManagerID: value.ManagerID,
			DepartmentID: value.DepartmentID, ProductID: line.ProductID, UnitAmount: line.TotalAmount,
			Quantity: 1, TotalAmount: line.TotalAmount, UntaxedAmount: line.TotalAmount,
			CurrencyID: value.CurrencyID, PaymentMode: value.PaymentMode, AccountID: line.AccountID,
			AnalyticAccountID: line.AnalyticAccountID, TaxIDs: append([]int64(nil), line.TaxIDs...),
			CompanyID: value.CompanyID, SplitOriginID: &value.ID,
		}
		if err := u.repo.Create(ctx, child); err != nil {
			return nil, err
		}
		ids = append(ids, child.ID)
	}
	return ids, nil
}

func isEditable(state expense.ExpenseState) bool {
	return state == expense.StateDraft || state == expense.StateSubmitted || state == expense.StateApproved
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

var _ expense.Usecase = (*UseCase)(nil)
