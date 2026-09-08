package expenseusecase_test

import (
	"context"
	"testing"
	"time"

	expensestorage "cashflow_backend/internal/adapters/storage/expense"
	hrs "cashflow_backend/internal/adapters/storage/hr"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/expense"
	"cashflow_backend/internal/domain/hr"
	expenseusecase "cashflow_backend/internal/usecase/expense"
)

type activitySchedulerSpy struct{ values []*activity.Activity }

func (s *activitySchedulerSpy) Schedule(_ context.Context, value *activity.Activity) error {
	s.values = append(s.values, value)
	return nil
}

func TestSubmitExpenseSchedulesManagerActivity(t *testing.T) {
	ctx := context.Background()
	hrRepo := hrs.NewMemoryRepo()
	managerID := int64(77)
	employee := &hr.Employee{Name: "Employee", Active: true, ExpenseManagerID: &managerID}
	if err := hrRepo.CreateEmployee(ctx, employee); err != nil {
		t.Fatal(err)
	}
	repo := expensestorage.NewMemoryRepo()
	uc := expenseusecase.New(repo, hrRepo, nil)
	spy := &activitySchedulerSpy{}
	uc.ConfigureActivityScheduler(spy, 4)
	value := &expense.Expense{Name: "Hotel", EmployeeID: employee.ID, CompanyID: 1, TotalAmount: 100}
	if err := uc.CreateExpense(ctx, value); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitExpense(ctx, value.ID); err != nil {
		t.Fatal(err)
	}
	if len(spy.values) != 1 {
		t.Fatalf("scheduled activities = %d, want 1", len(spy.values))
	}
	created := spy.values[0]
	if created.ActivityTypeID != 4 || created.AssignedUserID != managerID || created.ResModel != "hr.expense" || created.ResID == nil || *created.ResID != value.ID {
		t.Fatalf("activity = %+v, want expense approval activity", created)
	}
}

func TestSubmitExpenseAutovalidatesWithoutManager(t *testing.T) {
	ctx := context.Background()
	hrRepo := hrs.NewMemoryRepo()
	employee := &hr.Employee{Name: "Employee", Active: true}
	if err := hrRepo.CreateEmployee(ctx, employee); err != nil {
		t.Fatal(err)
	}
	repo := expensestorage.NewMemoryRepo()
	uc := expenseusecase.New(repo, hrRepo, nil)
	value := &expense.Expense{Name: "Taxi", EmployeeID: employee.ID, TotalAmount: 25}
	if err := uc.CreateExpense(ctx, value); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitExpense(ctx, value.ID); err != nil {
		t.Fatal(err)
	}
	found, err := uc.GetExpense(ctx, value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.State != expense.StateApproved || found.ApprovalDate == nil {
		t.Fatalf("expense = %+v, want auto-approved with approval date", found)
	}
}

func TestApproveExpenseRejectsDuplicateAndRequiresReasonOnRefusal(t *testing.T) {
	ctx := context.Background()
	hrRepo := hrs.NewMemoryRepo()
	managerID := int64(77)
	employee := &hr.Employee{Name: "Employee", Active: true, ExpenseManagerID: &managerID}
	if err := hrRepo.CreateEmployee(ctx, employee); err != nil {
		t.Fatal(err)
	}
	repo := expensestorage.NewMemoryRepo()
	uc := expenseusecase.New(repo, hrRepo, nil)
	date := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	first := &expense.Expense{Name: "Hotel", Date: date, EmployeeID: employee.ID, TotalAmount: 100}
	second := &expense.Expense{Name: "Hotel duplicate", Date: date, EmployeeID: employee.ID, TotalAmount: 100}
	if err := uc.CreateExpense(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := uc.CreateExpense(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitExpense(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitExpense(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := uc.ApproveExpense(ctx, second.ID, managerID); err == nil {
		t.Fatal("ApproveExpense() returned nil for duplicate")
	}
	if err := uc.RefuseExpense(ctx, first.ID, managerID, " "); err == nil {
		t.Fatal("RefuseExpense() returned nil without reason")
	}
	if err := uc.RefuseExpense(ctx, first.ID, managerID, "duplicate receipt"); err != nil {
		t.Fatal(err)
	}
}

func TestSplitExpenseRequiresBalancedPositiveLines(t *testing.T) {
	ctx := context.Background()
	hrRepo := hrs.NewMemoryRepo()
	employee := &hr.Employee{Name: "Employee", Active: true}
	if err := hrRepo.CreateEmployee(ctx, employee); err != nil {
		t.Fatal(err)
	}
	repo := expensestorage.NewMemoryRepo()
	uc := expenseusecase.New(repo, hrRepo, nil)
	value := &expense.Expense{Name: "Hotel", EmployeeID: employee.ID, TotalAmount: 100}
	if err := uc.CreateExpense(ctx, value); err != nil {
		t.Fatal(err)
	}
	ids, err := uc.SplitExpense(ctx, expense.ExpenseSplitRequest{ExpenseID: value.ID, Splits: []expense.ExpenseSplitLine{
		{Name: "Room", TotalAmount: 60}, {Name: "Meals", TotalAmount: 40},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("split IDs = %v, want 2 IDs", ids)
	}
}
