package bankstatement

import (
	"math"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// StatementState represents the lifecycle of a bank statement (account.bank.statement in Odoo v19).
type StatementState string

const (
	StatementStateOpen    StatementState = "open"    // Active statement, lines can still be added/matched
	StatementStateConfirm StatementState = "confirm" // Confirmed by the user; lines are locked
)

// BankStatement is a bank/cash journal statement grouping imported or manually-entered lines
// (account.bank.statement in Odoo).
type BankStatement struct {
	ID                 int64               `json:"id"`
	Name               string              `json:"name"`
	JournalID          int64               `json:"journal_id"`
	PartnerID          *int64              `json:"partner_id,omitempty"`
	Date               time.Time           `json:"date"`
	BalanceStart       float64             `json:"balance_start"`
	BalanceEnd         float64             `json:"balance_end"`
	BalanceEndReal     *float64            `json:"balance_end_real,omitempty"`
	Currency           string              `json:"currency"`
	State              StatementState      `json:"state"`
	IsComplete         bool                `json:"is_complete"`
	IsValid            bool                `json:"is_valid"`
	ProblemDescription string              `json:"problem_description"`
	Active             bool                `json:"active"`
	Audit              audit.Fields        `json:"audit"`
	Lines              []BankStatementLine `json:"lines,omitempty"`
}

// BankStatementLine is a single transaction of a bank statement
// (account.bank.statement.line in Odoo).
type BankStatementLine struct {
	ID             int64     `json:"id"`
	StatementID    int64     `json:"statement_id"`
	Name           string    `json:"name"`
	Ref            string    `json:"ref,omitempty"`
	Sequence       int       `json:"sequence"`
	Date           time.Time `json:"date"`
	Amount         float64   `json:"amount"`
	AmountCurrency float64   `json:"amount_currency"`
	Currency       string    `json:"currency"`
	PartnerID      *int64    `json:"partner_id,omitempty"`
	AccountID      *int64    `json:"account_id,omitempty"`
	MoveID         *int64    `json:"move_id,omitempty"`
	JournalID      *int64    `json:"journal_id,omitempty"`
	Checked        bool      `json:"checked"`
	RunningBalance float64   `json:"running_balance"`
	AmountResidual float64   `json:"amount_residual"`
	Reconciled     bool      `json:"reconciled"`
	MatchingNumber *string   `json:"matching_number,omitempty"`
	InternalIndex  string    `json:"internal_index,omitempty"`
	ImportBatchID  string    `json:"import_batch_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IsMoneyIn reports whether the line represents a cash inflow.
func (l *BankStatementLine) IsMoneyIn() bool {
	return l.Amount >= 0
}

// IsComplete reports whether the line has been booked through a journal entry.
func (l *BankStatementLine) IsComplete() bool {
	return l.MoveID != nil && *l.MoveID > 0
}

// AbsAmount returns the absolute transaction amount.
func (l *BankStatementLine) AbsAmount() float64 {
	return math.Abs(l.Amount)
}

// Validate checks the required constraints of a statement line.
func (l *BankStatementLine) Validate() error {
	if l.StatementID <= 0 {
		return platformerrors.Validation("statement is required", map[string]string{"statement_id": "must reference a valid statement"})
	}
	if l.Date.IsZero() {
		return platformerrors.Validation("transaction date is required", map[string]string{"date": "cannot be empty"})
	}
	if l.Name == "" {
		l.Name = "/"
	}
	if l.Currency == "" {
		l.Currency = "USD"
	}
	return nil
}

// ComputeTotals derives the running balance of each line and the expected closing balance.
func (s *BankStatement) ComputeTotals() {
	var running float64
	totalAmount := 0.0
	seq := 0
	for i := range s.Lines {
		seq++
		s.Lines[i].Sequence = seq
		totalAmount += s.Lines[i].Amount
		running += s.Lines[i].Amount
		s.Lines[i].RunningBalance = roundAmount(s.BalanceStart + running)
	}
	s.BalanceEnd = roundAmount(s.BalanceStart + totalAmount)
}

// ComputeCompleteness derives is_complete / is_valid / problem_description
// following the Odoo v19 computed semantics over lines.
func (s *BankStatement) ComputeCompleteness() {
	// A statement is complete when every line has been booked through a posted move
	// (our flow books every line immediately, so this reduces to move presence).
	complete := true
	for i := range s.Lines {
		if !s.Lines[i].IsComplete() {
			complete = false
			break
		}
	}
	s.IsComplete = complete

	problems := make([]string, 0, 2)
	if !s.IsComplete {
		problems = append(problems, "Some statement lines are not yet booked.")
	}
	if s.BalanceEndReal != nil && math.Abs(*s.BalanceEndReal-s.BalanceEnd) > 0.01 {
		problems = append(problems, "The actual closing balance differs from the computed closing balance.")
	}
	s.IsValid = len(problems) == 0
	s.ProblemDescription = ""
	if len(problems) > 0 {
		s.ProblemDescription = problems[0]
	}
}

func roundAmount(v float64) float64 {
	return math.Round(v*10000) / 10000
}
