package accounting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostMoveUseCase struct {
	repo       AccountingRepository
	publisher  EventPublisher
	logger     *slog.Logger
	governance MakerCheckerSettings
}

func NewPostMoveUseCase(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *PostMoveUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostMoveUseCase{repo: repo, publisher: publisher, logger: logger}
}

func newPostMoveUseCase(repo AccountingRepository, publisher EventPublisher, governance MakerCheckerSettings, logger *slog.Logger) *PostMoveUseCase {
	useCase := NewPostMoveUseCase(repo, publisher, logger)
	useCase.governance = governance
	return useCase
}

func (uc *PostMoveUseCase) Execute(ctx context.Context, moveID, actorID uuid.UUID) (*MoveResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "post_move")
	move, err := uc.repo.GetMoveByID(ctx, moveID)
	if err != nil {
		return nil, fmt.Errorf("get move: %w", err)
	}
	if move == nil {
		return nil, ErrMoveNotFound
	}
	if err := move.ValidateCanModify(); err != nil {
		return nil, err
	}
	if err := move.ValidateBalance(); err != nil {
		logger.WarnContext(ctx, "unbalanced move rejected", slog.String("move_id", moveID.String()), slog.Any("error", err))
		return nil, err
	}
	if uc.governance.enabled() {
		var totalDebit decimal.Decimal
		for _, line := range move.Lines {
			totalDebit = totalDebit.Add(line.Debit)
		}
		if totalDebit.GreaterThan(uc.governance.Threshold) {
			if actorID == uuid.Nil {
				return nil, ErrMakerCheckerActorRequired
			}
			if move.CreatedBy == nil || *move.CreatedBy == uuid.Nil {
				return nil, ErrMakerCheckerCreatorRequired
			}
			if actorID == *move.CreatedBy {
				return nil, ErrMakerCheckerViolation
			}
		}
	}
	period, err := uc.repo.GetPeriodByDate(ctx, move.Date)
	if err != nil {
		return nil, fmt.Errorf("get fiscal period for %s: %w", move.Date.Format(dateLayout), err)
	}
	if period == nil {
		return nil, ErrPeriodNotFound
	}
	if err := period.CanPost(); err != nil {
		return nil, err
	}
	move.PeriodID = period.ID

	now := time.Now().UTC()
	move.PostedBy = uuidPointer(actorID)
	audit := newFinancialAudit("account_move", move.ID, "POST", actorID, now)
	if err := uc.repo.PostMoveAtomic(ctx, move, audit); err != nil {
		logger.ErrorContext(ctx, "atomic move posting failed", slog.String("move_id", moveID.String()), slog.Any("error", err))
		return nil, fmt.Errorf("post move atomically: %w", err)
	}

	logger.InfoContext(ctx, "move posted", slog.String("move_id", move.ID.String()), slog.String("move_name", move.Name))
	if uc.publisher != nil {
		err := uc.publisher.Publish(ctx, "accounting.move.posted", map[string]any{
			"move_id": move.ID, "move_name": move.Name, "date": move.Date, "journal_id": move.JournalID,
		})
		if err != nil {
			logger.ErrorContext(ctx, "move posted but event publication failed", slog.String("move_id", move.ID.String()), slog.Any("error", err))
		}
	}
	response := ToMoveResponse(move)
	return &response, nil
}

func newFinancialAudit(entityType string, entityID uuid.UUID, action string, actorID uuid.UUID, timestamp time.Time) *FinancialAuditLog {
	return &FinancialAuditLog{
		EntityType:     entityType,
		EntityID:       entityID,
		Action:         action,
		ActorID:        actorID,
		CurrentLogHash: hashFinancialAudit(entityType, entityID, action, actorID, timestamp),
		Timestamp:      timestamp,
	}
}

func hashFinancialAudit(entityType string, entityID uuid.UUID, action string, actorID uuid.UUID, timestamp time.Time) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s", entityType, entityID, action, actorID, timestamp.UTC().Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hash[:])
}
