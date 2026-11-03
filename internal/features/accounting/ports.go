package accounting

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// AccountingRepository is the primary persistence port for the accounting domain.
type AccountingRepository interface {
	CreateAccount(ctx context.Context, acc *Account) error
	GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetAccountByCode(ctx context.Context, code string) (*Account, error)
	ListAccounts(ctx context.Context) ([]*Account, error)

	CreateFiscalYear(ctx context.Context, year *FiscalYear, periods []*FiscalPeriod) error
	GetPeriodByDate(ctx context.Context, date time.Time) (*FiscalPeriod, error)
	GetPeriodByID(ctx context.Context, id uuid.UUID) (*FiscalPeriod, error)
	UpdatePeriodState(ctx context.Context, id uuid.UUID, state PeriodState, actorID uuid.UUID) error

	GetJournalByID(ctx context.Context, id uuid.UUID) (*Journal, error)
	GetNextSequenceWithLock(ctx context.Context, tx pgx.Tx, journalID uuid.UUID) (string, error)

	SaveDraftMove(ctx context.Context, move *AccountMove) error
	GetMoveByID(ctx context.Context, id uuid.UUID) (*AccountMove, error)
	PostMoveAtomic(ctx context.Context, move *AccountMove, audit *FinancialAuditLog) error
	PostNewMoveWithEventAtomic(ctx context.Context, move *AccountMove, audit *FinancialAuditLog, idempotencyKey, eventType string) error
	ReverseMoveAtomic(ctx context.Context, originalMoveID uuid.UUID, reversalMove *AccountMove, audit *FinancialAuditLog) error

	GetTrialBalanceData(ctx context.Context, fromDate, toDate time.Time) ([]*TrialBalanceRow, error)
	GetGeneralLedgerLines(ctx context.Context, accountID uuid.UUID, fromDate, toDate time.Time) ([]*GeneralLedgerLineRow, error)
	GetFXMonetaryBalances(ctx context.Context, asOf time.Time) ([]*FXMonetaryBalance, error)

	GetProcessedEventMove(ctx context.Context, idempotencyKey string) (*AccountMove, error)
	SaveProcessedEvent(ctx context.Context, key, eventType string, moveID uuid.UUID) error
}

// FXRevaluationProvider supplies the monetary positions and closing rates used
// to build an unrealized foreign-currency revaluation draft.
type FXRevaluationProvider interface {
	GetFXRevaluationSnapshot(ctx context.Context, asOf time.Time) (*FXRevaluationSnapshot, error)
}

// FXClosingRateProvider returns the functional-currency units per one unit of
// foreign currency for the requested valuation date.
type FXClosingRateProvider interface {
	GetClosingRate(ctx context.Context, foreignCurrency, baseCurrency string, asOf time.Time) (decimal.Decimal, error)
}

// EventPublisher publishes domain events outside the accounting feature boundary.
type EventPublisher interface {
	Publish(ctx context.Context, eventName string, payload any) error
}
