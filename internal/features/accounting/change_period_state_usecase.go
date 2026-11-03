package accounting

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type ChangePeriodStateUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewChangePeriodStateUseCase(repo AccountingRepository, logger *slog.Logger) *ChangePeriodStateUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChangePeriodStateUseCase{repo: repo, logger: logger}
}

func (uc *ChangePeriodStateUseCase) Execute(
	ctx context.Context,
	periodID, actorID uuid.UUID,
	request ChangePeriodStateRequest,
) (*FiscalPeriodResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "change_period_state")
	if err := request.Validate(); err != nil {
		return nil, err
	}
	period, err := uc.repo.GetPeriodByID(ctx, periodID)
	if err != nil {
		return nil, fmt.Errorf("get fiscal period: %w", err)
	}
	if period == nil {
		return nil, ErrPeriodNotFound
	}

	newState := PeriodState(request.State)
	if period.State != newState && !validPeriodTransition(period.State, newState) {
		return nil, fmt.Errorf("%w: %s to %s", ErrInvalidPeriodTransition, period.State, newState)
	}
	if period.State == newState {
		response := ToFiscalPeriodResponse(period)
		return &response, nil
	}
	if err := uc.repo.UpdatePeriodState(ctx, periodID, newState, actorID); err != nil {
		logger.ErrorContext(ctx, "fiscal period state change failed", slog.String("period_id", periodID.String()), slog.Any("error", err))
		return nil, fmt.Errorf("update fiscal period state: %w", err)
	}

	period.State = newState
	if newState == PeriodStateOpen {
		period.ClosedAt = nil
		period.ClosedBy = nil
	} else {
		now := time.Now().UTC()
		period.ClosedAt = &now
		period.ClosedBy = uuidPointer(actorID)
	}
	logger.InfoContext(ctx, "fiscal period state changed", slog.String("period_id", periodID.String()), slog.String("state", string(newState)))
	response := ToFiscalPeriodResponse(period)
	return &response, nil
}

func validPeriodTransition(from, to PeriodState) bool {
	switch from {
	case PeriodStateOpen:
		return to == PeriodStateSoftClosed || to == PeriodStateHardClosed
	case PeriodStateSoftClosed:
		return to == PeriodStateOpen || to == PeriodStateHardClosed
	case PeriodStateHardClosed:
		return false
	default:
		return false
	}
}

func uuidPointer(value uuid.UUID) *uuid.UUID {
	copy := value
	return &copy
}
