package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var subledgerJournalID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type subledgerAccountMapping struct {
	debitCode  string
	debitType  AccountType
	creditCode string
	creditType AccountType
}

var subledgerMappings = map[string]subledgerAccountMapping{
	"goods_delivered": {
		debitCode: "501000", debitType: AccountTypeExpense,
		creditCode: "140000", creditType: AccountTypeAsset,
	},
	"customer_invoice_issued": {
		debitCode: "105000", debitType: AccountTypeAsset,
		creditCode: "401000", creditType: AccountTypeIncome,
	},
	"payment_received": {
		debitCode: "102000", debitType: AccountTypeAsset,
		creditCode: "105000", creditType: AccountTypeAsset,
	},
}

type IngestSubledgerEventUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewIngestSubledgerEventUseCase(repo AccountingRepository, logger *slog.Logger) *IngestSubledgerEventUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &IngestSubledgerEventUseCase{repo: repo, logger: logger}
}

func (uc *IngestSubledgerEventUseCase) Execute(ctx context.Context, request IngestSubledgerEventRequest) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "ingest_subledger_event")
	if err := request.Validate(); err != nil {
		return nil, err
	}
	mapping, ok := subledgerMappings[request.EventType]
	if !ok {
		return nil, fmt.Errorf("unknown subledger event_type: %s", request.EventType)
	}
	existing, err := uc.repo.GetProcessedEventMove(ctx, request.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("check processed subledger event: %w", err)
	}
	if existing != nil {
		logger.InfoContext(ctx, "subledger event already processed", slog.String("idempotency_key", request.IdempotencyKey))
		response := ToMoveResponse(existing)
		return &response, nil
	}

	debitAccount, err := uc.getMappedAccount(ctx, mapping.debitCode, mapping.debitType)
	if err != nil {
		return nil, fmt.Errorf("resolve debit account: %w", err)
	}
	creditAccount, err := uc.getMappedAccount(ctx, mapping.creditCode, mapping.creditType)
	if err != nil {
		return nil, fmt.Errorf("resolve credit account: %w", err)
	}
	eventDate, err := parseDate(request.Date, "date")
	if err != nil {
		return nil, err
	}
	period, err := uc.repo.GetPeriodByDate(ctx, eventDate)
	if err != nil {
		return nil, fmt.Errorf("get subledger event period: %w", err)
	}
	if period == nil {
		return nil, ErrPeriodNotFound
	}
	if err := period.CanPost(); err != nil {
		return nil, err
	}
	journal, err := uc.repo.GetJournalByID(ctx, subledgerJournalID)
	if err != nil {
		return nil, fmt.Errorf("get subledger journal: %w", err)
	}
	if journal == nil || !journal.IsActive {
		return nil, errors.New("subledger journal must exist and be active")
	}

	now := time.Now().UTC()
	moveID := uuid.New()
	move := &AccountMove{
		ID:        moveID,
		JournalID: journal.ID,
		PeriodID:  period.ID,
		Date:      eventDate,
		State:     MoveStateDraft,
		Ref:       fmt.Sprintf("[%s] %s", request.EventType, request.Ref),
		CreatedAt: now,
		Lines: []AccountMoveLine{
			{
				ID: uuid.New(), MoveID: moveID, AccountID: debitAccount.ID,
				Debit: request.Amount, Credit: decimal.Zero,
				AmountCurrency: request.Amount, CurrencyID: request.Currency,
				Date: eventDate, CreatedAt: now,
			},
			{
				ID: uuid.New(), MoveID: moveID, AccountID: creditAccount.ID,
				Debit: decimal.Zero, Credit: request.Amount,
				AmountCurrency: request.Amount.Neg(), CurrencyID: request.Currency,
				Date: eventDate, CreatedAt: now,
			},
		},
	}
	move.PostedBy = uuidPointer(uuid.Nil)
	audit := newFinancialAudit("account_move", move.ID, "INGEST_POST", uuid.Nil, now)
	if err := uc.repo.PostNewMoveWithEventAtomic(ctx, move, audit, request.IdempotencyKey, request.EventType); err != nil {
		existing, lookupErr := uc.repo.GetProcessedEventMove(ctx, request.IdempotencyKey)
		if lookupErr != nil {
			return nil, errors.Join(
				fmt.Errorf("post subledger event atomically: %w", err),
				fmt.Errorf("look up idempotent result: %w", lookupErr),
			)
		}
		if existing != nil {
			response := ToMoveResponse(existing)
			return &response, nil
		}
		logger.ErrorContext(ctx, "subledger event posting failed", slog.String("idempotency_key", request.IdempotencyKey), slog.Any("error", err))
		return nil, fmt.Errorf("post subledger event atomically: %w", err)
	}

	logger.InfoContext(ctx, "subledger event posted", slog.String("move_id", move.ID.String()), slog.String("idempotency_key", request.IdempotencyKey))
	response := ToMoveResponse(move)
	return &response, nil
}

func (uc *IngestSubledgerEventUseCase) getMappedAccount(ctx context.Context, code string, expectedType AccountType) (*Account, error) {
	account, err := uc.repo.GetAccountByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("account %s is not configured: %w", code, err)
	}
	if account == nil || !account.IsActive || account.Type != expectedType {
		return nil, fmt.Errorf("account %s must be active and have type %s", code, expectedType)
	}
	return account, nil
}
