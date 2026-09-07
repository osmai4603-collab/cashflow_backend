package accounting

import (
	"context"
	"time"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistent storage contract for all Core Accounting operations.
type Repository interface {
	// ─── Chart of Accounts ──────────────────────────────────────────────
	CreateAccount(ctx context.Context, a *Account) error
	GetAccountByID(ctx context.Context, id int64) (*Account, error)
	GetAccountByCode(ctx context.Context, code string) (*Account, error)
	UpdateAccount(ctx context.Context, a *Account) error
	DeleteAccount(ctx context.Context, id int64) error
	ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Account], error)

	// ─── Journals ───────────────────────────────────────────────────────
	CreateJournal(ctx context.Context, j *Journal) error
	GetJournalByID(ctx context.Context, id int64) (*Journal, error)
	GetJournalByCode(ctx context.Context, code string) (*Journal, error)
	UpdateJournal(ctx context.Context, j *Journal) error
	DeleteJournal(ctx context.Context, id int64) error
	ListJournals(ctx context.Context) ([]Journal, error)
	GetNextSequence(ctx context.Context, journalID int64, year int) (string, error)

	// ─── Taxes ──────────────────────────────────────────────────────────
	CreateTax(ctx context.Context, t *Tax) error
	GetTaxByID(ctx context.Context, id int64) (*Tax, error)
	UpdateTax(ctx context.Context, t *Tax) error
	DeleteTax(ctx context.Context, id int64) error
	ListTaxes(ctx context.Context, scope *TaxScope) ([]Tax, error)

	// ─── Payment Terms ──────────────────────────────────────────────────
	CreatePaymentTerm(ctx context.Context, pt *PaymentTerm) error
	GetPaymentTermByID(ctx context.Context, id int64) (*PaymentTerm, error)
	UpdatePaymentTerm(ctx context.Context, pt *PaymentTerm) error
	DeletePaymentTerm(ctx context.Context, id int64) error
	ListPaymentTerms(ctx context.Context) ([]PaymentTerm, error)

	// ─── Account Moves & Invoices ───────────────────────────────────────
	CreateMove(ctx context.Context, m *AccountMove) error
	GetMoveByID(ctx context.Context, id int64) (*AccountMove, error)
	GetMoveWithLines(ctx context.Context, id int64) (*AccountMove, error)
	UpdateMove(ctx context.Context, m *AccountMove) error
	DeleteMove(ctx context.Context, id int64) error
	ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[AccountMove], error)

	// ─── Move Lines & Reconciliation ────────────────────────────────────
	GetMoveLineByID(ctx context.Context, id int64) (*AccountMoveLine, error)
	UpdateMoveLine(ctx context.Context, l *AccountMoveLine) error
	UpdateMoveLineReconcile(ctx context.Context, id int64, reconciled bool, residual float64, matchingNumber *string) error
	// ListReconcilableMoveLines returns posted, reconcilable, unmatched lines on the given partner,
	// excluding the supplied line IDs (used to surface matching candidates for a statement line).
	ListReconcilableMoveLines(ctx context.Context, partnerID *int64, excludeLineIDs []int64, limit int) ([]AccountMoveLine, error)

	// ─── Financial Reports ──────────────────────────────────────────────
	GetTrialBalance(ctx context.Context, fromDate, toDate time.Time, onlyPosted bool) (*TrialBalanceReport, error)
	GetProfitAndLoss(ctx context.Context, fromDate, toDate time.Time) (*ProfitAndLossReport, error)
	GetBalanceSheet(ctx context.Context, asOfDate time.Time) (*BalanceSheetReport, error)
	GetGeneralLedger(ctx context.Context, accountID *int64, partnerID *int64, fromDate, toDate *time.Time) ([]GeneralLedgerItem, error)

	// ─── EDI (Electronic Data Interchange) ──────────────────────────────
	CreateEDIDocument(ctx context.Context, doc *EDIDocument) error
	GetEDIDocumentByID(ctx context.Context, id int64) (*EDIDocument, error)
	GetEDIDocumentsByMoveID(ctx context.Context, moveID int64) ([]EDIDocument, error)
	UpdateEDIDocument(ctx context.Context, doc *EDIDocument) error

	CreateEDICertificate(ctx context.Context, cert *EDICertificate) error
	GetEDICertificateByID(ctx context.Context, id int64) (*EDICertificate, error)
	GetActiveCertificate(ctx context.Context, companyID int64) (*EDICertificate, error)
	UpdateEDICertificate(ctx context.Context, cert *EDICertificate) error
}
