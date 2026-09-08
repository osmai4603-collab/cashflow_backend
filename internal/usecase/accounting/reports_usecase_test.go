package accountingusecase_test

import (
	"context"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/usecase/accounting"
)

func setupReportsEnv(t *testing.T) (*accountingusecase.UseCase, *accountingstorage.MemoryRepo) {
	repo := accountingstorage.NewMemoryRepo()
	uc := accountingusecase.New(repo, nil)
	return uc, repo
}

func TestUseCase_FinancialReports(t *testing.T) {
	ctx := context.Background()
	uc, repo := setupReportsEnv(t)

	// 1. Seed some data: A Sales Invoice
	// Debit Receivable (3), Credit Revenue (10)
	move := &accounting.AccountMove{
		Date:     time.Now().UTC(),
		MoveType: accounting.MoveTypeOutInvoice,
		State:    accounting.MoveStatePosted,
		Lines: []accounting.AccountMoveLine{
			{AccountID: 3, Debit: 1150.0, Credit: 0},
			{AccountID: 10, Debit: 0, Credit: 1000.0},
			{AccountID: 7, Debit: 0, Credit: 150.0}, // VAT Output
		},
	}
	repo.CreateMove(ctx, move)

	t.Run("Trial Balance", func(t *testing.T) {
		report, err := uc.GetTrialBalance(ctx, time.Time{}, time.Time{}, true)
		if err != nil {
			t.Fatalf("GetTrialBalance failed: %v", err)
		}
		if !report.IsBalanced {
			t.Errorf("Trial Balance should be balanced, diff: %v", report.Difference)
		}
		if report.TotalDebit != 1150.0 || report.TotalCredit != 1150.0 {
			t.Errorf("unexpected totals: debit=%v, credit=%v", report.TotalDebit, report.TotalCredit)
		}
	})

	t.Run("Profit and Loss", func(t *testing.T) {
		report, err := uc.GetProfitAndLoss(ctx, time.Time{}, time.Time{})
		if err != nil {
			t.Fatalf("GetProfitAndLoss failed: %v", err)
		}
		if report.NetProfit != 1000.0 {
			t.Errorf("expected net profit 1000.0, got %v", report.NetProfit)
		}
	})

	t.Run("Balance Sheet", func(t *testing.T) {
		report, err := uc.GetBalanceSheet(ctx, time.Now().UTC())
		if err != nil {
			t.Fatalf("GetBalanceSheet failed: %v", err)
		}
		// Assets: Receivable (1150)
		// Liabilities: VAT Output (150)
		// Equity: Net Profit (1000)
		if report.TotalAssets != 1150.0 {
			t.Errorf("expected assets 1150.0, got %v", report.TotalAssets)
		}
		if report.TotalLiabilities != 150.0 {
			t.Errorf("expected liabilities 150.0, got %v", report.TotalLiabilities)
		}
		if report.TotalEquity+report.RetainedEarnings != 1000.0 {
			t.Errorf("expected equity 1000.0, got %v", report.TotalEquity+report.RetainedEarnings)
		}
	})
}
