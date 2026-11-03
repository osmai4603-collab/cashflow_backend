package accounting

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountType string

const (
	AccountTypeAsset     AccountType = "asset"
	AccountTypeLiability AccountType = "liability"
	AccountTypeEquity    AccountType = "equity"
	AccountTypeIncome    AccountType = "income"
	AccountTypeExpense   AccountType = "expense"
)

type NormalBalance string

const (
	NormalBalanceDebit  NormalBalance = "debit"
	NormalBalanceCredit NormalBalance = "credit"
)

type PeriodState string

const (
	PeriodStateOpen       PeriodState = "open"
	PeriodStateSoftClosed PeriodState = "soft_closed"
	PeriodStateHardClosed PeriodState = "hard_closed"
)

type MoveState string

const (
	MoveStateDraft    MoveState = "draft"
	MoveStatePosted   MoveState = "posted"
	MoveStateReversed MoveState = "reversed"
)

var (
	ErrAccountNotFound             = errors.New("accounting: account not found")
	ErrDuplicateAccountCode        = errors.New("accounting: duplicate account code")
	ErrPeriodClosed                = errors.New("accounting: fiscal period is closed for posting")
	ErrPeriodNotFound              = errors.New("accounting: no active fiscal period for date")
	ErrInvalidPeriodTransition     = errors.New("accounting: invalid fiscal period state transition")
	ErrJournalNotFound             = errors.New("accounting: journal not found")
	ErrMoveNotFound                = errors.New("accounting: journal entry not found")
	ErrMoveAlreadyReversed         = errors.New("accounting: journal entry has already been reversed")
	ErrMoveRequiresAtLeastTwoLines = errors.New("accounting: a journal entry requires at least two lines")
	ErrNegativeAmountsNotAllowed   = errors.New("accounting: negative debit or credit amounts are prohibited")
	ErrZeroAmountLine              = errors.New("accounting: move line must have non-zero debit or credit")
	ErrUnbalancedMove              = errors.New("accounting: move is unbalanced; debits must strictly equal credits")
	ErrMoveAlreadyPosted           = errors.New("accounting: move is already posted and immutable")
	ErrCannotReverseUnpostedMove   = errors.New("accounting: only posted moves can be reversed")
	ErrMakerCheckerCreatorRequired = errors.New("accounting: creator attribution is required to post a move above the configured threshold")
	ErrMakerCheckerViolation       = errors.New("accounting: move creator cannot post a move above the configured threshold")
	ErrMakerCheckerActorRequired   = errors.New("accounting: authenticated actor is required to post a move above the configured threshold")
)

type Account struct {
	ID             uuid.UUID
	Code           string
	Name           string
	Type           AccountType
	NormalBalance  NormalBalance
	CurrencyID     string
	ParentID       *uuid.UUID
	IsReconcilable bool
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type FiscalYear struct {
	ID        uuid.UUID
	Year      int
	StartDate time.Time
	EndDate   time.Time
	State     string
	CreatedAt time.Time
}

type FiscalPeriod struct {
	ID           uuid.UUID
	FiscalYearID uuid.UUID
	PeriodNumber int
	Name         string
	StartDate    time.Time
	EndDate      time.Time
	State        PeriodState
	ClosedAt     *time.Time
	ClosedBy     *uuid.UUID
	CreatedAt    time.Time
}

// CanPost returns an error unless the fiscal period is open.
func (p *FiscalPeriod) CanPost() error {
	if p.State != PeriodStateOpen {
		return fmt.Errorf("%w: period %s is in %s state", ErrPeriodClosed, p.Name, p.State)
	}
	return nil
}

type Journal struct {
	ID             uuid.UUID
	Code           string
	Name           string
	Type           string
	SequencePrefix string
	NextNumber     int64
	IsActive       bool
	CreatedAt      time.Time
}

type AccountMoveLine struct {
	ID             uuid.UUID
	MoveID         uuid.UUID
	AccountID      uuid.UUID
	PartnerID      *uuid.UUID
	Debit          decimal.Decimal
	Credit         decimal.Decimal
	AmountCurrency decimal.Decimal
	CurrencyID     string
	CostCenterID   *uuid.UUID
	MatchingNumber string
	Date           time.Time
	CreatedAt      time.Time
}

// AccountMove is the aggregate root for a journal entry.
type AccountMove struct {
	ID             uuid.UUID
	Name           string
	JournalID      uuid.UUID
	PeriodID       uuid.UUID
	Date           time.Time
	State          MoveState
	Ref            string
	ReversedMoveID *uuid.UUID
	HashSignature  string
	PostedAt       *time.Time
	PostedBy       *uuid.UUID
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Lines          []AccountMoveLine
}

// ValidateBalance verifies that the move has at least two non-negative,
// non-zero lines and that total debits equal total credits exactly.
func (m *AccountMove) ValidateBalance() error {
	if len(m.Lines) < 2 {
		return ErrMoveRequiresAtLeastTwoLines
	}

	var totalDebit, totalCredit decimal.Decimal

	for _, line := range m.Lines {
		if line.Debit.IsNegative() || line.Credit.IsNegative() {
			return ErrNegativeAmountsNotAllowed
		}
		if line.Debit.IsZero() && line.Credit.IsZero() {
			return ErrZeroAmountLine
		}
		totalDebit = totalDebit.Add(line.Debit)
		totalCredit = totalCredit.Add(line.Credit)
	}

	if !totalDebit.Equal(totalCredit) {
		return fmt.Errorf("%w: debit sum (%s) != credit sum (%s)", ErrUnbalancedMove, totalDebit, totalCredit)
	}

	return nil
}

// ValidateCanModify returns an error when the move is no longer a draft.
func (m *AccountMove) ValidateCanModify() error {
	if m.State != MoveStateDraft {
		return ErrMoveAlreadyPosted
	}
	return nil
}

// ValidateCanReverse allows reversal only after a move has been posted.
func (m *AccountMove) ValidateCanReverse() error {
	if m.State != MoveStatePosted {
		return ErrCannotReverseUnpostedMove
	}
	return nil
}

type FinancialAuditLog struct {
	ID              int64
	EntityType      string
	EntityID        uuid.UUID
	Action          string
	ActorID         uuid.UUID
	PayloadBefore   []byte
	PayloadAfter    []byte
	IPAddress       string
	PreviousLogHash string
	CurrentLogHash  string
	Timestamp       time.Time
}

// TrialBalanceRow is a persistence read model for account balances over a period.
type TrialBalanceRow struct {
	AccountID     uuid.UUID
	AccountCode   string
	AccountName   string
	InitialDebit  decimal.Decimal
	InitialCredit decimal.Decimal
	PeriodDebit   decimal.Decimal
	PeriodCredit  decimal.Decimal
	EndingDebit   decimal.Decimal
	EndingCredit  decimal.Decimal
}

// GeneralLedgerLineRow is a persistence read model for an account ledger entry.
type GeneralLedgerLineRow struct {
	MoveID         uuid.UUID
	MoveName       string
	Date           time.Time
	Ref            string
	Debit          decimal.Decimal
	Credit         decimal.Decimal
	RunningBalance decimal.Decimal
}

// FXMonetaryBalance is the signed foreign-currency and functional-currency
// balance for one account, partner, cost center, and currency at a date.
type FXMonetaryBalance struct {
	AccountID                uuid.UUID
	AccountType              AccountType
	PartnerID                *uuid.UUID
	CostCenterID             *uuid.UUID
	CurrencyID               string
	ForeignAmount            decimal.Decimal
	FunctionalCarryingAmount decimal.Decimal
}
