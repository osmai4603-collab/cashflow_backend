package accounting

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGenerateCashFlowUseCaseReconcilesIndirectCashFlow(t *testing.T) {
	cashID, incomeID, expenseID := uuid.New(), uuid.New(), uuid.New()
	receivableID, payableID, investmentID, loanID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repo := newMockAccountingRepository()
	repo.accounts = []*Account{
		{ID: cashID, Type: AccountTypeAsset, IsCashEquivalent: true},
		{ID: incomeID, Type: AccountTypeIncome},
		{ID: expenseID, Type: AccountTypeExpense, IsNonCashAdjustment: true},
		{ID: receivableID, Type: AccountTypeAsset, CashFlowActivity: "operating"},
		{ID: payableID, Type: AccountTypeLiability, CashFlowActivity: "operating"},
		{ID: investmentID, Type: AccountTypeAsset, CashFlowActivity: "investing"},
		{ID: loanID, Type: AccountTypeLiability, CashFlowActivity: "financing"},
	}
	repo.trialBalance = []*TrialBalanceRow{
		{AccountID: cashID, PeriodDebit: decimal.NewFromInt(130)},
		{AccountID: incomeID, PeriodCredit: decimal.NewFromInt(100)},
		{AccountID: expenseID, PeriodDebit: decimal.NewFromInt(30)},
		{AccountID: receivableID, PeriodDebit: decimal.NewFromInt(20)},
		{AccountID: payableID, PeriodCredit: decimal.NewFromInt(10)},
		{AccountID: investmentID, PeriodDebit: decimal.NewFromInt(25)},
		{AccountID: loanID, PeriodCredit: decimal.NewFromInt(15)},
	}
	repo.cashFlowMovements = []*CashFlowMovement{
		{MoveID: uuid.New(), Amount: decimal.NewFromInt(25), Activities: []string{"investing"}},
		{MoveID: uuid.New(), Amount: decimal.NewFromInt(15), Activities: []string{"financing"}},
	}
	fromDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	response, err := NewGenerateCashFlowUseCase(repo).Execute(context.Background(), fromDate, toDate)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !response.NetIncome.Equal(decimal.NewFromInt(70)) ||
		!response.NonCashAdjustments.Equal(decimal.NewFromInt(30)) ||
		!response.WorkingCapitalChanges.Equal(decimal.NewFromInt(-10)) ||
		!response.OperatingCashFlow.Equal(decimal.NewFromInt(90)) ||
		!response.InvestingCashFlow.Equal(decimal.NewFromInt(25)) ||
		!response.FinancingCashFlow.Equal(decimal.NewFromInt(15)) ||
		!response.NetChangeInCash.Equal(decimal.NewFromInt(130)) ||
		!response.ReconciliationDifference.IsZero() || !response.IsReconciled {
		t.Fatalf("unexpected cash flow statement: %+v", response)
	}
}

func TestGenerateCashFlowUseCaseRejectsAmbiguousCashMovement(t *testing.T) {
	repo := newMockAccountingRepository()
	repo.cashFlowMovements = []*CashFlowMovement{{
		MoveID: uuid.New(), Amount: decimal.NewFromInt(1),
		Activities: []string{"operating", "investing"},
	}}

	_, err := NewGenerateCashFlowUseCase(repo).Execute(
		context.Background(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("Execute() expected an error for a cash movement classified to multiple activities")
	}
}
