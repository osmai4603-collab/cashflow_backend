package bankstatement

import (
	"context"
	"time"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistent storage contract for bank statements and reconciliation.
type Repository interface {
	// ─── Statements ─────────────────────────────────────────────────────
	CreateStatement(ctx context.Context, st *BankStatement) error
	GetStatementByID(ctx context.Context, id int64) (*BankStatement, error)
	GetStatementWithLines(ctx context.Context, id int64) (*BankStatement, error)
	UpdateStatement(ctx context.Context, st *BankStatement) error
	DeleteStatement(ctx context.Context, id int64) error
	ListStatements(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[BankStatement], error)

	// ───Statement Lines ─────────────────────────────────────────────────
	AddStatementLines(ctx context.Context, statementID int64, lines []BankStatementLine) error
	GetStatementLineByID(ctx context.Context, id int64) (*BankStatementLine, error)
	UpdateStatementLine(ctx context.Context, line *BankStatementLine) error
	UpdateStatementLineReconcileState(ctx context.Context, lineID int64, reconciled bool, residual float64, matchingNumber *string) error

	// ─── Reconciles ─────────────────────────────────────────────────────
	CreatePartialReconcile(ctx context.Context, pr *PartialReconcile) error
	ListPartialReconcilesByLine(ctx context.Context, lineID int64) ([]PartialReconcile, error)
	DeletePartialReconcile(ctx context.Context, id int64) error
	CreateFullReconcile(ctx context.Context, fr *FullReconcile) error
	GetFullReconcileByMatchingNumber(ctx context.Context, matchingNumber string) (*FullReconcile, error)

	// ─── Reconcile Models ───────────────────────────────────────────────
	CreateReconcileModel(ctx context.Context, m *ReconcileModel) error
	GetReconcileModelByID(ctx context.Context, id int64) (*ReconcileModel, error)
	UpdateReconcileModel(ctx context.Context, m *ReconcileModel) error
	DeleteReconcileModel(ctx context.Context, id int64) error
	ListReconcileModels(ctx context.Context) ([]ReconcileModel, error)
	ListAutoReconcileModels(ctx context.Context) ([]ReconcileModel, error)

	// ─── Cash Rounding ──────────────────────────────────────────────────
	CreateCashRounding(ctx context.Context, cr *CashRounding) error
	GetCashRoundingByID(ctx context.Context, id int64) (*CashRounding, error)
	UpdateCashRounding(ctx context.Context, cr *CashRounding) error
	DeleteCashRounding(ctx context.Context, id int64) error
	ListCashRoundings(ctx context.Context) ([]CashRounding, error)
}

// StatementSequenceProvider generates the next statement name sequence for a journal.
type StatementSequenceProvider interface {
	NextStatementName(ctx context.Context, journalCode string, year int) (string, error)
}

// TimestampFn standardizes time injection in tests.
type TimestampFn func() time.Time
