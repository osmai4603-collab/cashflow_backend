package accounting_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
)

func TestAccountValidation(t *testing.T) {
	t.Run("valid account", func(t *testing.T) {
		acc := &accounting.Account{
			Code: "101000",
			Name: "Cash on Hand",
			Type: accounting.AccountTypeAssetCash,
		}
		if err := acc.Validate(); err != nil {
			t.Fatalf("expected valid account, got error: %v", err)
		}
	})

	t.Run("missing code", func(t *testing.T) {
		acc := &accounting.Account{
			Name: "Cash on Hand",
			Type: accounting.AccountTypeAssetCash,
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for missing code, got nil")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		acc := &accounting.Account{
			Code: "101000",
			Type: accounting.AccountTypeAssetCash,
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for missing name, got nil")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		acc := &accounting.Account{
			Code: "101000",
			Name: "Test",
			Type: "invalid_type",
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for invalid type, got nil")
		}
	})

	t.Run("circular parent reference", func(t *testing.T) {
		id := int64(10)
		acc := &accounting.Account{
			ID:       id,
			Code:     "101000",
			Name:     "Test",
			Type:     accounting.AccountTypeAssetCash,
			ParentID: &id,
		}
		if err := acc.Validate(); err == nil {
			t.Fatal("expected error for circular parent reference, got nil")
		}
	})
}

func TestJournalSequenceFormat(t *testing.T) {
	j := &accounting.Journal{
		Code:           "INV",
		Name:           "Customer Invoices",
		Type:           accounting.JournalTypeSale,
		SequencePrefix: "INV/%Y/",
	}

	seq := j.FormatSequence(2026, 42)
	expected := "INV/2026/00042"
	if seq != expected {
		t.Fatalf("expected %s, got %s", expected, seq)
	}
}

func TestTaxCalculations(t *testing.T) {
	t.Run("tax excluded from price (15%)", func(t *testing.T) {
		tax := &accounting.Tax{
			Name:         "15% Sales VAT",
			Type:         accounting.TaxTypePercent,
			TypeTaxUse:   accounting.TaxScopeSale,
			Amount:       15.0,
			PriceInclude: false,
			AccountID:    1,
		}

		res := tax.Compute(100.0)
		if res.UntaxedAmount != 100.0 {
			t.Errorf("expected untaxed 100.0, got %.4f", res.UntaxedAmount)
		}
		if res.TaxAmount != 15.0 {
			t.Errorf("expected tax 15.0, got %.4f", res.TaxAmount)
		}
		if res.TotalAmount != 115.0 {
			t.Errorf("expected total 115.0, got %.4f", res.TotalAmount)
		}
	})

	t.Run("tax included in price (15%)", func(t *testing.T) {
		tax := &accounting.Tax{
			Name:         "15% VAT Included",
			Type:         accounting.TaxTypePercent,
			TypeTaxUse:   accounting.TaxScopeSale,
			Amount:       15.0,
			PriceInclude: true,
			AccountID:    1,
		}

		res := tax.Compute(115.0)
		if res.UntaxedAmount != 100.0 {
			t.Errorf("expected untaxed 100.0, got %.4f", res.UntaxedAmount)
		}
		if res.TaxAmount != 15.0 {
			t.Errorf("expected tax 15.0, got %.4f", res.TaxAmount)
		}
		if res.TotalAmount != 115.0 {
			t.Errorf("expected total 115.0, got %.4f", res.TotalAmount)
		}
	})
}

func TestPaymentTermDueDate(t *testing.T) {
	term := &accounting.PaymentTerm{
		Name: "30 Days",
		Lines: []accounting.PaymentTermLine{
			{ValueType: accounting.PaymentTermValueBalance, Days: 30},
		},
	}

	baseDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dueDate := term.ComputeDueDate(baseDate)
	expected := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	if !dueDate.Equal(expected) {
		t.Fatalf("expected due date %v, got %v", expected, dueDate)
	}
}

func TestMoveBalanceAndPosting(t *testing.T) {
	t.Run("balanced entry posts successfully", func(t *testing.T) {
		move := &accounting.AccountMove{
			JournalID: 1,
			Date:      time.Now(),
			MoveType:  accounting.MoveTypeEntry,
			Lines: []accounting.AccountMoveLine{
				{AccountID: 1, Name: "Debit line", Debit: 250.0, Credit: 0.0},
				{AccountID: 2, Name: "Credit line", Debit: 0.0, Credit: 250.0},
			},
		}

		if err := move.ValidateBalance(); err != nil {
			t.Fatalf("expected balanced entry, got error: %v", err)
		}

		if err := move.Post("MISC/2026/00001"); err != nil {
			t.Fatalf("expected successful post, got error: %v", err)
		}

		if move.State != accounting.MoveStatePosted {
			t.Errorf("expected state posted, got %s", move.State)
		}
		if move.Name != "MISC/2026/00001" {
			t.Errorf("expected name MISC/2026/00001, got %s", move.Name)
		}
	})

	t.Run("unbalanced entry post is rejected", func(t *testing.T) {
		move := &accounting.AccountMove{
			JournalID: 1,
			Date:      time.Now(),
			MoveType:  accounting.MoveTypeEntry,
			Lines: []accounting.AccountMoveLine{
				{AccountID: 1, Name: "Debit line", Debit: 250.0, Credit: 0.0},
				{AccountID: 2, Name: "Credit line", Debit: 0.0, Credit: 200.0},
			},
		}

		if err := move.ValidateBalance(); err == nil {
			t.Fatal("expected error on unbalanced entry, got nil")
		}

		if err := move.Post("MISC/2026/00001"); err == nil {
			t.Fatal("expected post to fail on unbalanced entry, got nil")
		}
	})

	t.Run("reversal swaps debits and credits", func(t *testing.T) {
		id := int64(100)
		move := &accounting.AccountMove{
			ID:        id,
			Name:      "INV/2026/00005",
			JournalID: 1,
			Date:      time.Now(),
			MoveType:  accounting.MoveTypeOutInvoice,
			State:     accounting.MoveStatePosted,
			Lines: []accounting.AccountMoveLine{
				{AccountID: 3, Name: "Receivable", Debit: 115.0, Credit: 0.0},
				{AccountID: 10, Name: "Sales", Debit: 0.0, Credit: 100.0},
				{AccountID: 7, Name: "VAT", Debit: 0.0, Credit: 15.0},
			},
		}

		rev := move.CreateReverseMove(time.Now(), "Cancellation")
		if rev.MoveType != accounting.MoveTypeOutRefund {
			t.Errorf("expected reversal move_type out_refund, got %s", rev.MoveType)
		}
		if rev.ReversedEntryID == nil || *rev.ReversedEntryID != id {
			t.Errorf("expected reversed_entry_id %d, got %v", id, rev.ReversedEntryID)
		}
		if len(rev.Lines) != 3 {
			t.Fatalf("expected 3 reversal lines, got %d", len(rev.Lines))
		}

		// First line was Debit 115, reversal should be Credit 115
		if rev.Lines[0].Credit != 115.0 || rev.Lines[0].Debit != 0.0 {
			t.Errorf("expected line 0 to be Credit 115, got Debit: %.2f Credit: %.2f", rev.Lines[0].Debit, rev.Lines[0].Credit)
		}
		// Reversal must be balanced
		if err := rev.ValidateBalance(); err != nil {
			t.Fatalf("reversal move must be balanced: %v", err)
		}
	})
}
