package accountingstorage_test

import (
	"context"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepoAccountOperations(t *testing.T) {
	ctx := context.Background()
	repo := accountingstorage.NewMemoryRepo()

	t.Run("list seeded accounts", func(t *testing.T) {
		res, err := repo.ListAccounts(ctx, nil, pagination.PageRequest{Page: 1, Limit: 50})
		if err != nil {
			t.Fatalf("unexpected error listing accounts: %v", err)
		}
		if res.TotalItems < 15 {
			t.Fatalf("expected at least 15 seeded accounts, got %d", res.TotalItems)
		}
	})

	t.Run("create account", func(t *testing.T) {
		acc := &accounting.Account{
			Code: "105000",
			Name: "Petty Cash",
			Type: accounting.AccountTypeAssetCash,
		}
		if err := repo.CreateAccount(ctx, acc); err != nil {
			t.Fatalf("failed to create account: %v", err)
		}
		if acc.ID == 0 {
			t.Fatal("expected assigned account ID")
		}

		fetched, err := repo.GetAccountByID(ctx, acc.ID)
		if err != nil {
			t.Fatalf("failed to get account by id: %v", err)
		}
		if fetched.Code != "105000" {
			t.Errorf("expected code 105000, got %s", fetched.Code)
		}
	})

	t.Run("duplicate code conflict", func(t *testing.T) {
		acc := &accounting.Account{
			Code: "101000", // Already seeded Cash on Hand
			Name: "Duplicate",
			Type: accounting.AccountTypeAssetCash,
		}
		if err := repo.CreateAccount(ctx, acc); err == nil {
			t.Fatal("expected error on duplicate account code, got nil")
		}
	})
}

func TestMemoryRepoJournalAndSequence(t *testing.T) {
	ctx := context.Background()
	repo := accountingstorage.NewMemoryRepo()

	t.Run("advance sequence", func(t *testing.T) {
		seq1, err := repo.GetNextSequence(ctx, 1, 2026)
		if err != nil {
			t.Fatalf("unexpected error generating sequence: %v", err)
		}
		if seq1 != "INV/2026/00001" {
			t.Fatalf("expected INV/2026/00001, got %s", seq1)
		}

		seq2, err := repo.GetNextSequence(ctx, 1, 2026)
		if err != nil {
			t.Fatalf("unexpected error generating second sequence: %v", err)
		}
		if seq2 != "INV/2026/00002" {
			t.Fatalf("expected INV/2026/00002, got %s", seq2)
		}
	})
}

func TestMemoryRepoMoveAndReports(t *testing.T) {
	ctx := context.Background()
	repo := accountingstorage.NewMemoryRepo()

	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	// Create and post a balanced move: Sales invoice
	// Debit Receivable (Acc 3): 115
	// Credit Sales Revenue (Acc 10): 100
	// Credit Tax Payable (Acc 7): 15
	move := &accounting.AccountMove{
		Name:          "/",
		MoveType:      accounting.MoveTypeOutInvoice,
		JournalID:     1,
		Date:          now,
		State:         accounting.MoveStateDraft,
		AmountUntaxed: 100,
		AmountTax:     15,
		AmountTotal:   115,
		Lines: []accounting.AccountMoveLine{
			{AccountID: 3, Name: "Customer Receivable", Debit: 115.0, Credit: 0.0},
			{AccountID: 10, Name: "Product Sales", Debit: 0.0, Credit: 100.0},
			{AccountID: 7, Name: "Sales Tax", Debit: 0.0, Credit: 15.0},
		},
	}

	if err := repo.CreateMove(ctx, move); err != nil {
		t.Fatalf("failed to create move: %v", err)
	}

	seq, err := repo.GetNextSequence(ctx, 1, 2026)
	if err != nil {
		t.Fatalf("failed to get sequence: %v", err)
	}
	if err := move.Post(seq); err != nil {
		t.Fatalf("failed to post move: %v", err)
	}
	if err := repo.UpdateMove(ctx, move); err != nil {
		t.Fatalf("failed to update posted move: %v", err)
	}

	// 1. Trial Balance Test
	t.Run("trial balance is balanced", func(t *testing.T) {
		tb, err := repo.GetTrialBalance(ctx, time.Time{}, time.Time{}, true)
		if err != nil {
			t.Fatalf("failed to get trial balance: %v", err)
		}
		if !tb.IsBalanced {
			t.Errorf("trial balance must be balanced, difference: %.4f", tb.Difference)
		}
		if tb.TotalDebit != 115.0 || tb.TotalCredit != 115.0 {
			t.Errorf("expected 115.0 total debit and credit, got Debit: %.2f Credit: %.2f", tb.TotalDebit, tb.TotalCredit)
		}
	})

	// 2. Profit & Loss Test
	t.Run("profit and loss report", func(t *testing.T) {
		pnl, err := repo.GetProfitAndLoss(ctx, time.Time{}, time.Time{})
		if err != nil {
			t.Fatalf("failed to get p&l report: %v", err)
		}
		if pnl.TotalIncome != 100.0 {
			t.Errorf("expected total income 100.0, got %.2f", pnl.TotalIncome)
		}
		if pnl.NetProfit != 100.0 {
			t.Errorf("expected net profit 100.0, got %.2f", pnl.NetProfit)
		}
	})

	// 3. Balance Sheet Test
	t.Run("balance sheet is balanced", func(t *testing.T) {
		bs, err := repo.GetBalanceSheet(ctx, now)
		if err != nil {
			t.Fatalf("failed to get balance sheet: %v", err)
		}
		if !bs.IsBalanced {
			t.Errorf("balance sheet must be balanced, difference: %.4f", bs.Difference)
		}
		if bs.TotalAssets != 115.0 {
			t.Errorf("expected 115.0 assets, got %.2f", bs.TotalAssets)
		}
		if bs.TotalLiabilities != 15.0 {
			t.Errorf("expected 15.0 liabilities, got %.2f", bs.TotalLiabilities)
		}
		if bs.RetainedEarnings != 100.0 {
			t.Errorf("expected 100.0 retained earnings, got %.2f", bs.RetainedEarnings)
		}
	})

	// 4. General Ledger Test
	t.Run("general ledger", func(t *testing.T) {
		accID := int64(3)
		gl, err := repo.GetGeneralLedger(ctx, &accID, nil, nil, nil)
		if err != nil {
			t.Fatalf("failed to get general ledger: %v", err)
		}
		if len(gl) != 1 {
			t.Fatalf("expected 1 ledger item, got %d", len(gl))
		}
		if gl[0].Debit != 115.0 {
			t.Errorf("expected debit 115.0, got %.2f", gl[0].Debit)
		}
	})
}
