package accounting

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type UnreconcileMoveLinesUseCase struct {
	repo AccountingRepository
}

func NewUnreconcileMoveLinesUseCase(repo AccountingRepository) *UnreconcileMoveLinesUseCase {
	return &UnreconcileMoveLinesUseCase{repo: repo}
}

func (uc *UnreconcileMoveLinesUseCase) Execute(
	ctx context.Context,
	actorID uuid.UUID,
	request UnreconcileMoveLinesRequest,
) error {
	if actorID == uuid.Nil {
		return ErrAccountingActorRequired
	}
	if err := request.Validate(); err != nil {
		return err
	}
	date, err := parseDate(request.Date, "date")
	if err != nil {
		return err
	}
	if err := uc.repo.UnreconcileMoveLines(ctx, request.ReconciliationID, date, actorID); err != nil {
		return fmt.Errorf("unreconcile move lines: %w", err)
	}
	return nil
}
