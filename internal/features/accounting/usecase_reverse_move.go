package accounting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ReverseMoveUseCase struct {
	repo      AccountingRepository
	publisher EventPublisher
	logger    *slog.Logger
}

func NewReverseMoveUseCase(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *ReverseMoveUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReverseMoveUseCase{repo: repo, publisher: publisher, logger: logger}
}

func (uc *ReverseMoveUseCase) Execute(
	ctx context.Context,
	originalMoveID, actorID uuid.UUID,
	reversalDate time.Time,
	reason string,
) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "reverse_move")
	original, err := uc.repo.GetMoveByID(ctx, originalMoveID)
	if err != nil {
		return nil, fmt.Errorf("get original move: %w", err)
	}
	if original == nil {
		return nil, ErrMoveNotFound
	}
	if err := original.ValidateCanReverse(); err != nil {
		return nil, err
	}
	if len(original.Lines) < 2 {
		return nil, ErrMoveRequiresAtLeastTwoLines
	}
	if reversalDate.IsZero() {
		return nil, fmt.Errorf("reversal date is required")
	}
	reversalDate = time.Date(reversalDate.Year(), reversalDate.Month(), reversalDate.Day(), 0, 0, 0, 0, time.UTC)
	period, err := uc.repo.GetPeriodByDate(ctx, reversalDate)
	if err != nil {
		return nil, fmt.Errorf("get fiscal period for reversal date: %w", err)
	}
	if period == nil {
		return nil, ErrPeriodNotFound
	}
	if err := period.CanPost(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	reversalID := uuid.New()
	reversal := &AccountMove{
		ID:             reversalID,
		JournalID:      original.JournalID,
		PeriodID:       period.ID,
		Date:           reversalDate,
		State:          MoveStateDraft,
		Ref:            fmt.Sprintf("Reversal of %s - %s", original.Name, strings.TrimSpace(reason)),
		ReversedMoveID: uuidPointer(originalMoveID),
		CreatedAt:      now,
		Lines:          make([]AccountMoveLine, len(original.Lines)),
	}
	for i, line := range original.Lines {
		reversal.Lines[i] = AccountMoveLine{
			ID:             uuid.New(),
			MoveID:         reversalID,
			AccountID:      line.AccountID,
			PartnerID:      cloneUUID(line.PartnerID),
			Debit:          line.Credit,
			Credit:         line.Debit,
			AmountCurrency: line.AmountCurrency.Neg(),
			CurrencyID:     line.CurrencyID,
			CostCenterID:   cloneUUID(line.CostCenterID),
			MatchingNumber: line.MatchingNumber,
			Date:           reversalDate,
			CreatedAt:      now,
		}
	}
	audit := newFinancialAudit("account_move", originalMoveID, "REVERSE", actorID, now)
	if err := uc.repo.ReverseMoveAtomic(ctx, originalMoveID, reversal, audit); err != nil {
		logger.ErrorContext(ctx, "atomic move reversal failed", slog.String("move_id", originalMoveID.String()), slog.Any("error", err))
		return nil, fmt.Errorf("reverse move atomically: %w", err)
	}

	logger.InfoContext(ctx, "move reversed", slog.String("original_move_id", originalMoveID.String()), slog.String("reversal_move_id", reversal.ID.String()))
	if uc.publisher != nil {
		err := uc.publisher.Publish(ctx, "accounting.move.reversed", map[string]any{
			"move_id": reversal.ID, "original_move_id": originalMoveID, "date": reversal.Date,
		})
		if err != nil {
			logger.ErrorContext(ctx, "move reversed but event publication failed", slog.String("move_id", reversal.ID.String()), slog.Any("error", err))
		}
	}
	response := ToMoveResponse(reversal)
	return &response, nil
}
