package expenseusecase_test

import (
	"context"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	expensestorage "cashflow_backend/internal/adapters/storage/expense"
	hrs "cashflow_backend/internal/adapters/storage/hr"
	"cashflow_backend/internal/domain/expense"
	"cashflow_backend/internal/domain/hr"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	expenseusecase "cashflow_backend/internal/usecase/expense"
)

func TestAccountingIntegrationPostsBalancedEmployeeReceipt(t *testing.T) {
	ctx := context.Background()
	hrRepo := hrs.NewMemoryRepo()
	employee := &hr.Employee{Name: "Employee", Active: true}
	if err := hrRepo.CreateEmployee(ctx, employee); err != nil {
		t.Fatal(err)
	}
	accountingRepo := accountingstorage.NewMemoryRepo()
	accountingUC := accountingusecase.New(accountingRepo, nil)
	integration := expenseusecase.NewAccountingIntegration(accountingUC, hrRepo)
	accountID := int64(13)
	value := &expense.Expense{
		Name: "Hotel", Date: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC), EmployeeID: employee.ID,
		TotalAmount: 115, UnitAmount: 115, Quantity: 1, CurrencyID: 1, PaymentMode: expense.PaymentOwnAccount,
		AccountID: &accountID, TaxIDs: []int64{3}, State: expense.StateApproved, CompanyID: 1,
	}
	move, err := integration.PostExpense(ctx, value, expenseusecase.PostExpenseInput{JournalID: 2})
	if err != nil {
		t.Fatal(err)
	}
	if move.MoveType != "in_receipt" || move.State != "posted" {
		t.Fatalf("move = %+v, want posted in_receipt", move)
	}
	if move.TotalDebit() != 115 || move.TotalCredit() != 115 {
		t.Fatalf("move is not balanced: debit=%v credit=%v", move.TotalDebit(), move.TotalCredit())
	}
}

func TestUseCasePostExpenseLinksMoveAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	hrRepo := hrs.NewMemoryRepo()
	employee := &hr.Employee{Name: "Employee", Active: true}
	if err := hrRepo.CreateEmployee(ctx, employee); err != nil {
		t.Fatal(err)
	}
	accountingRepo := accountingstorage.NewMemoryRepo()
	accountingUC := accountingusecase.New(accountingRepo, nil)
	integration := expenseusecase.NewAccountingIntegration(accountingUC, hrRepo)
	expenseRepo := expensestorage.NewMemoryRepo()
	uc := expenseusecase.New(expenseRepo, hrRepo, nil, integration)
	accountID := int64(13)
	value := &expense.Expense{Name: "Taxi", EmployeeID: employee.ID, TotalAmount: 25, CurrencyID: 1, AccountID: &accountID, CompanyID: 1}
	if err := uc.CreateExpense(ctx, value); err != nil {
		t.Fatal(err)
	}
	if err := uc.SubmitExpense(ctx, value.ID); err != nil {
		t.Fatal(err)
	}
	if err := uc.PostExpenseWithAccounting(ctx, value.ID, expenseusecase.PostExpenseInput{JournalID: 2}); err != nil {
		t.Fatal(err)
	}
	posted, err := uc.GetExpense(ctx, value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if posted.State != expense.StatePosted || posted.AccountMoveID == nil {
		t.Fatalf("expense = %+v, want posted and linked move", posted)
	}
	if err := uc.PostExpenseWithAccounting(ctx, value.ID, expenseusecase.PostExpenseInput{JournalID: 2}); err != nil {
		t.Fatal(err)
	}
}
