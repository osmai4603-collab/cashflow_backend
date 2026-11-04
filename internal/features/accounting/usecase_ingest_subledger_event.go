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

type IngestSubledgerEventUseCase struct {
	repo               AccountingRepository
	rates              FXClosingRateProvider
	functionalCurrency string
	logger             *slog.Logger
}

func NewIngestSubledgerEventUseCase(repo AccountingRepository, logger *slog.Logger) *IngestSubledgerEventUseCase {
	return NewIngestSubledgerEventUseCaseWithRates(repo, nil, "SAR", logger)
}

func NewIngestSubledgerEventUseCaseWithRates(
	repo AccountingRepository,
	rates FXClosingRateProvider,
	functionalCurrency string,
	logger *slog.Logger,
) *IngestSubledgerEventUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	currency, err := normalizeCurrency(functionalCurrency, "SAR")
	if err != nil {
		currency = "SAR"
	}
	return &IngestSubledgerEventUseCase{
		repo: repo, rates: rates, functionalCurrency: currency, logger: logger,
	}
}

func (uc *IngestSubledgerEventUseCase) Execute(ctx context.Context, request IngestSubledgerEventRequest) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "ingest_subledger_event")
	if err := request.Validate(); err != nil {
		return nil, err
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

	mapping, err := uc.repo.GetSubledgerEventMapping(ctx, request.EventType)
	if err != nil {
		return nil, fmt.Errorf("%w: event_type %q", ErrSubledgerMappingNotFound, request.EventType)
	}
	if mapping == nil || !mapping.IsActive {
		return nil, fmt.Errorf("%w: event_type %q", ErrSubledgerMappingNotFound, request.EventType)
	}
	debitAccount, err := uc.repo.GetAccountByID(ctx, mapping.DebitAccountID)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped debit account: %w", err)
	}
	if debitAccount == nil || !debitAccount.IsActive {
		return nil, fmt.Errorf("mapped debit account %s must be active", mapping.DebitAccountID)
	}
	creditAccount, err := uc.repo.GetAccountByID(ctx, mapping.CreditAccountID)
	if err != nil {
		return nil, fmt.Errorf("resolve mapped credit account: %w", err)
	}
	if creditAccount == nil || !creditAccount.IsActive {
		return nil, fmt.Errorf("mapped credit account %s must be active", mapping.CreditAccountID)
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
	journal, err := uc.repo.GetJournalByID(ctx, mapping.JournalID)
	if err != nil {
		return nil, fmt.Errorf("get subledger journal: %w", err)
	}
	if journal == nil || !journal.IsActive {
		return nil, errors.New("subledger journal must exist and be active")
	}
	rate := decimal.NewFromInt(1)
	if request.Currency != uc.functionalCurrency {
		if uc.rates == nil {
			return nil, fmt.Errorf("%w: no rate provider configured for currency %s on date %s", ErrExchangeRateMissing, request.Currency, eventDate.Format(dateLayout))
		}
		rate, err = uc.rates.GetClosingRate(ctx, request.Currency, uc.functionalCurrency, eventDate)
		if err != nil {
			return nil, fmt.Errorf("convert subledger event to %s: %w", uc.functionalCurrency, err)
		}
		if !rate.IsPositive() {
			return nil, fmt.Errorf("exchange rate %s/%s on %s must be positive", request.Currency, uc.functionalCurrency, eventDate.Format(dateLayout))
		}
	}
	functionalAmount := request.Amount.Mul(rate).Round(4)

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
				Debit: functionalAmount, Credit: decimal.Zero,
				AmountCurrency: request.Amount, CurrencyID: request.Currency,
				Date: eventDate, CreatedAt: now,
			},
			{
				ID: uuid.New(), MoveID: moveID, AccountID: creditAccount.ID,
				Debit: decimal.Zero, Credit: functionalAmount,
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
