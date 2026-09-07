package accountingusecase_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	"cashflow_backend/internal/domain/accounting"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

func newTestUseCase() (*accountingusecase.UseCase, *accountingstorage.MemoryRepo) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := accountingstorage.NewMemoryRepo()
	uc := accountingusecase.New(repo, logger)
	return uc, repo
}

func TestAccountUseCase(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUseCase()

	t.Run("create and get account", func(t *testing.T) {
		acc, err := uc.CreateAccount(ctx, accountingusecase.CreateAccountInput{
			Code: "105500",
			Name: "Petty Cash HQ",
			Type: accounting.AccountTypeAssetCash,
		})
		if err != nil {
			t.Fatalf("failed to create account: %v", err)
		}
		if acc.ID == 0 {
			t.Fatal("expected non-zero account id")
		}

		fetched, err := uc.GetAccount(ctx, acc.ID)
		if err != nil {
			t.Fatalf("failed to get account: %v", err)
		}
		if fetched.Name != "Petty Cash HQ" {
			t.Errorf("expected name Petty Cash HQ, got %s", fetched.Name)
		}
	})

	t.Run("duplicate account code fails", func(t *testing.T) {
		_, err := uc.CreateAccount(ctx, accountingusecase.CreateAccountInput{
			Code: "101000", // Already seeded
			Name: "Duplicate",
			Type: accounting.AccountTypeAssetCash,
		})
		if err == nil {
			t.Fatal("expected error on duplicate account code, got nil")
		}
	})
}

func TestInvoiceCreationAndPosting(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUseCase()

	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	prodID := int64(42)

	// Create customer invoice: 2 items @ 100 with 15% VAT (Tax ID 1)
	// Untaxed: 200, Tax: 30, Total: 230
	in := accountingusecase.CreateInvoiceInput{
		MoveType:    accounting.MoveTypeOutInvoice,
		PartnerID:   1, // Customer
		JournalID:   1, // Customer Invoices
		Date:        now,
		InvoiceDate: &now,
		Ref:         "SO001-INV",
		Items: []accountingusecase.InvoiceLineItemInput{
			{
				ProductID: &prodID,
				Name:      "Service Package",
				Quantity:  2.0,
				PriceUnit: 100.0,
				Discount:  0.0,
				TaxIDs:    []int64{1}, // 15% Sales VAT
			},
		},
	}

	invoice, err := uc.CreateInvoice(ctx, in)
	if err != nil {
		t.Fatalf("failed to create invoice: %v", err)
	}

	if invoice.AmountUntaxed != 200.0 {
		t.Errorf("expected untaxed 200.0, got %.2f", invoice.AmountUntaxed)
	}
	if invoice.AmountTax != 30.0 {
		t.Errorf("expected tax 30.0, got %.2f", invoice.AmountTax)
	}
	if invoice.AmountTotal != 230.0 {
		t.Errorf("expected total 230.0, got %.2f", invoice.AmountTotal)
	}

	// Verify balanced lines
	if err := invoice.ValidateBalance(); err != nil {
		t.Fatalf("generated invoice must be balanced: %v", err)
	}

	// Post the invoice
	posted, err := uc.PostMove(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("failed to post invoice: %v", err)
	}

	if posted.State != accounting.MoveStatePosted {
		t.Errorf("expected state posted, got %s", posted.State)
	}
	if posted.Name != "INV/2026/00001" {
		t.Errorf("expected sequence INV/2026/00001, got %s", posted.Name)
	}

	// Reverse the invoice (Credit Note)
	reversed, err := uc.ReverseMove(ctx, posted.ID, now, "Customer Refund")
	if err != nil {
		t.Fatalf("failed to reverse invoice: %v", err)
	}

	if reversed.MoveType != accounting.MoveTypeOutRefund {
		t.Errorf("expected reversal move_type out_refund, got %s", reversed.MoveType)
	}
	if reversed.AmountTotal != 230.0 {
		t.Errorf("expected reversal total 230.0, got %.2f", reversed.AmountTotal)
	}
	if err := reversed.ValidateBalance(); err != nil {
		t.Fatalf("reversal must be balanced: %v", err)
	}
}

func TestInvoiceCOGSBooking(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUseCase()

	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	prodID := int64(7)
	cogsAcc := int64(12) // 500000 COGS
	stockAcc := int64(5) // 140000 Inventory

	// Invoice 10 units @ 50 with a COGS of 10 * 20 = 200 booked on the delivered cost.
	in := accountingusecase.CreateInvoiceInput{
		MoveType:    accounting.MoveTypeOutInvoice,
		PartnerID:   1, // Customer
		JournalID:   1, // Customer Invoices
		Date:        now,
		InvoiceDate: &now,
		Ref:         "SO-COGS-INV",
		Items: []accountingusecase.InvoiceLineItemInput{
			{
				ProductID:    &prodID,
				Name:         "Widget Delivered",
				Quantity:     10.0,
				PriceUnit:    50.0,
				Discount:     0.0,
				TaxIDs:       []int64{1},
				CogsAmount:   200.0,
				CogsAccountID: &cogsAcc,
				StockAccountID: &stockAcc,
			},
		},
	}

	invoice, err := uc.CreateInvoice(ctx, in)
	if err != nil {
		t.Fatalf("failed to create invoice with COGS: %v", err)
	}
	if err := invoice.ValidateBalance(); err != nil {
		t.Fatalf("invoice with COGS lines must stay balanced: %v", err)
	}

	// Expect: receivable 575 (500 + 75 tax) + COGS 200 debit / Inventory 200 credit.
	var cogsLines, invOutLines int
	for _, l := range invoice.Lines {
		if l.DisplayType == "cogs" {
			if l.CogsOriginID == nil {
				t.Errorf("cogs line %d must carry cogs_origin_id", l.ID)
			}
			if strings.Contains(l.Name, "COGS") {
				cogsLines++
				if l.Debit != 200.0 || l.Credit != 0.0 {
					t.Errorf("expected COGS debit 200.0, got debit %.2f credit %.2f", l.Debit, l.Credit)
				}
				if l.AccountID != cogsAcc {
					t.Errorf("expected cogs account %d, got %d", cogsAcc, l.AccountID)
				}
			} else {
				invOutLines++
				if l.Credit != 200.0 || l.Debit != 0.0 {
					t.Errorf("expected Inventory credit 200.0, got debit %.2f credit %.2f", l.Debit, l.Credit)
				}
				if l.AccountID != stockAcc {
					t.Errorf("expected stock account %d, got %d", stockAcc, l.AccountID)
				}
			}
		}
	}
	if cogsLines != 1 || invOutLines != 1 {
		t.Fatalf("expected 1 cogs + 1 inventory line, got %d + %d", cogsLines, invOutLines)
	}

	// The persisted move should round-trip the cogs metadata.
	fetched, err := uc.GetMove(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("failed to reload invoice: %v", err)
	}
	roundTripCogs := 0
	for _, l := range fetched.Lines {
		if l.DisplayType == "cogs" {
			roundTripCogs++
		}
	}
	if roundTripCogs != 2 {
		t.Fatalf("expected 2 cogs lines after reload, got %d", roundTripCogs)
	}

	// No COGS when CogsAmount is zero.
	plain := accountingusecase.CreateInvoiceInput{
		MoveType:    accounting.MoveTypeOutInvoice,
		PartnerID:   1,
		JournalID:   1,
		Date:        now,
		InvoiceDate: &now,
		Ref:         "SO-PLAIN-INV",
		Items: []accountingusecase.InvoiceLineItemInput{
			{ProductID: &prodID, Name: "Consulting", Quantity: 1.0, PriceUnit: 100.0, TaxIDs: []int64{1}},
		},
	}
	plainInv, err := uc.CreateInvoice(ctx, plain)
	if err != nil {
		t.Fatalf("failed to create invoice without COGS: %v", err)
	}
	for _, l := range plainInv.Lines {
		if l.DisplayType == "cogs" {
			t.Fatalf("unexpected cogs line on plain invoice: %+v", l)
		}
	}
}

func TestFinancialReportsUseCase(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUseCase()

	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// Post manual entry: 500 Bank (Debit) / 500 Capital (Credit)
	entry, err := uc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
		JournalID: 5, // MISC
		Date:      now,
		Ref:       "Capital Contribution",
		Lines: []accountingusecase.JournalEntryLineInput{
			{AccountID: 2, Name: "Bank Deposit", Debit: 500.0, Credit: 0.0},
			{AccountID: 8, Name: "Owner Equity", Debit: 0.0, Credit: 500.0},
		},
	})
	if err != nil {
		t.Fatalf("failed to create journal entry: %v", err)
	}

	if _, err := uc.PostMove(ctx, entry.ID); err != nil {
		t.Fatalf("failed to post entry: %v", err)
	}

	// Check Trial Balance
	tb, err := uc.GetTrialBalance(ctx, time.Time{}, time.Time{}, true)
	if err != nil {
		t.Fatalf("failed to get trial balance: %v", err)
	}
	if !tb.IsBalanced {
		t.Errorf("expected trial balance to be balanced, difference: %.4f", tb.Difference)
	}

	// Check Balance Sheet
	bs, err := uc.GetBalanceSheet(ctx, now)
	if err != nil {
		t.Fatalf("failed to get balance sheet: %v", err)
	}
	if !bs.IsBalanced {
		t.Errorf("expected balance sheet to be balanced, difference: %.4f", bs.Difference)
	}
	if bs.TotalAssets != 500.0 {
		t.Errorf("expected assets 500.0, got %.2f", bs.TotalAssets)
	}
	if bs.TotalEquity != 500.0 {
		t.Errorf("expected equity 500.0, got %.2f", bs.TotalEquity)
	}
}
