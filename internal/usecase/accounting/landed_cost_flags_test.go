package accountingusecase_test

import (
	"context"
	"testing"
	"time"

	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

func TestCreateJournalEntryPreservesLandedCostFlags(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUseCase()

	move, err := uc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
		JournalID: 6,
		Date:      time.Now().UTC(),
		Ref:      "LC-001",
		Lines: []accountingusecase.JournalEntryLineInput{
			{AccountID: 5, Name: "Inventory debit", Debit: 100, Credit: 0, IsLandedCostsLine: true},
			{AccountID: 17, Name: "Expense credit", Debit: 0, Credit: 100},
		},
	})
	if err != nil {
		t.Fatalf("CreateJournalEntry failed: %v", err)
	}
	if len(move.Lines) != 2 {
		t.Fatalf("expected 2 move lines, got %d", len(move.Lines))
	}
	if !move.Lines[0].IsLandedCostsLine {
		t.Fatal("expected landed-cost flag to be preserved on generated journal line")
	}
}
