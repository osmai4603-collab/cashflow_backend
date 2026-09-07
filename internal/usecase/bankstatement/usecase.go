package bankstatementusecase

import (
	"context"
	"log/slog"
	"math"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/bankstatement"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// AccountingService abstracts the accounting operations needed by the bank statement module.
type AccountingService interface {
	CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error)
	PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
	GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
	GetJournal(ctx context.Context, id int64) (*accounting.Journal, error)
	GetAccount(ctx context.Context, id int64) (*accounting.Account, error)
	GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error)
	UpdatePaymentStatus(ctx context.Context, id int64, state accounting.PaymentState, residual float64) error
	GetMoveLine(ctx context.Context, id int64) (*accounting.AccountMoveLine, error)
	UpdateMoveLineReconcile(ctx context.Context, id int64, reconciled bool, residual float64, matchingNumber *string) error
	ListReconcilableMoveLines(ctx context.Context, partnerID *int64, excludeLineIDs []int64, limit int) ([]accounting.AccountMoveLine, error)
}

// UseCase orchestrates bank statement and reconciliation operations.
type UseCase struct {
	repo          bankstatement.Repository
	accountingSvc AccountingService
	sequences     bankstatement.StatementSequenceProvider
	now           func() time.Time
	logger        *slog.Logger
}

// New constructs a bank statement UseCase.
func New(
	repo bankstatement.Repository,
	accountingSvc AccountingService,
	sequences bankstatement.StatementSequenceProvider,
	logger *slog.Logger,
) *UseCase {
	return &UseCase{
		repo:          repo,
		accountingSvc: accountingSvc,
		sequences:     sequences,
		now:           time.Now,
		logger:        logger,
	}
}

func roundTo4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

const residualTolerance = 0.004
