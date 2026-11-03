package accounting_test

import (
	"errors"
	"testing"

	"cashflow_backend/internal/features/accounting"

	"github.com/shopspring/decimal"
)

func TestAccountMove_ValidateBalance(t *testing.T) {
	tests := []struct {
		name    string
		lines   []accounting.AccountMoveLine
		wantErr error
	}{
		{
			name: "balanced move passes validation",
			lines: []accounting.AccountMoveLine{
				{Debit: decimal.RequireFromString("500.50")},
				{Credit: decimal.RequireFromString("500.50")},
			},
		},
		{
			name: "unbalanced move fails validation",
			lines: []accounting.AccountMoveLine{
				{Debit: decimal.RequireFromString("500.50")},
				{Credit: decimal.RequireFromString("400.00")},
			},
			wantErr: accounting.ErrUnbalancedMove,
		},
		{
			name: "single line move fails validation",
			lines: []accounting.AccountMoveLine{
				{Debit: decimal.NewFromInt(100)},
			},
			wantErr: accounting.ErrMoveRequiresAtLeastTwoLines,
		},
		{
			name: "negative amount fails validation",
			lines: []accounting.AccountMoveLine{
				{Debit: decimal.NewFromInt(-100)},
				{Credit: decimal.NewFromInt(100)},
			},
			wantErr: accounting.ErrNegativeAmountsNotAllowed,
		},
		{
			name: "zero amount line fails validation",
			lines: []accounting.AccountMoveLine{
				{Debit: decimal.NewFromInt(100)},
				{Credit: decimal.NewFromInt(100)},
				{},
			},
			wantErr: accounting.ErrZeroAmountLine,
		},
		{
			name: "fractional totals are compared exactly",
			lines: []accounting.AccountMoveLine{
				{Debit: decimal.RequireFromString("0.1")},
				{Debit: decimal.RequireFromString("0.2")},
				{Credit: decimal.RequireFromString("0.3")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			move := accounting.AccountMove{Lines: tt.lines}
			err := move.ValidateBalance()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateBalance() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFiscalPeriod_CanPost(t *testing.T) {
	tests := []struct {
		name  string
		state accounting.PeriodState
		want  error
	}{
		{name: "open period permits posting", state: accounting.PeriodStateOpen},
		{name: "soft-closed period rejects posting", state: accounting.PeriodStateSoftClosed, want: accounting.ErrPeriodClosed},
		{name: "hard-closed period rejects posting", state: accounting.PeriodStateHardClosed, want: accounting.ErrPeriodClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period := accounting.FiscalPeriod{State: tt.state}
			if err := period.CanPost(); !errors.Is(err, tt.want) {
				t.Fatalf("CanPost() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestAccountMove_StateGuards(t *testing.T) {
	t.Run("only drafts can be modified", func(t *testing.T) {
		move := accounting.AccountMove{State: accounting.MoveStateDraft}
		if err := move.ValidateCanModify(); err != nil {
			t.Fatalf("draft ValidateCanModify() error = %v", err)
		}

		for _, state := range []accounting.MoveState{accounting.MoveStatePosted, accounting.MoveStateReversed} {
			move.State = state
			if err := move.ValidateCanModify(); !errors.Is(err, accounting.ErrMoveAlreadyPosted) {
				t.Errorf("%s ValidateCanModify() error = %v, want %v", state, err, accounting.ErrMoveAlreadyPosted)
			}
		}
	})

	t.Run("only posted moves can be reversed", func(t *testing.T) {
		move := accounting.AccountMove{State: accounting.MoveStatePosted}
		if err := move.ValidateCanReverse(); err != nil {
			t.Fatalf("posted ValidateCanReverse() error = %v", err)
		}

		for _, state := range []accounting.MoveState{accounting.MoveStateDraft, accounting.MoveStateReversed} {
			move.State = state
			if err := move.ValidateCanReverse(); !errors.Is(err, accounting.ErrCannotReverseUnpostedMove) {
				t.Errorf("%s ValidateCanReverse() error = %v, want %v", state, err, accounting.ErrCannotReverseUnpostedMove)
			}
		}
	})
}
