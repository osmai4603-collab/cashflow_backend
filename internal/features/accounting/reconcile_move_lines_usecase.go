package accounting

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ReconcileMoveLinesUseCase struct {
	repo AccountingRepository
}

func NewReconcileMoveLinesUseCase(repo AccountingRepository) *ReconcileMoveLinesUseCase {
	return &ReconcileMoveLinesUseCase{repo: repo}
}

func (uc *ReconcileMoveLinesUseCase) Execute(
	ctx context.Context,
	actorID uuid.UUID,
	request ReconcileMoveLinesRequest,
) (*ReconciliationResponse, error) {
	if actorID == uuid.Nil {
		return nil, ErrAccountingActorRequired
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	date, err := parseDate(request.Date, "date")
	if err != nil {
		return nil, err
	}
	allocations := make([]ReconciliationAllocation, len(request.Allocations))
	for i, allocation := range request.Allocations {
		allocations[i] = ReconciliationAllocation{MoveLineID: allocation.MoveLineID, Amount: allocation.Amount}
	}
	id, err := uc.repo.ReconcileMoveLines(ctx, date, actorID, uuid.Nil, "", allocations)
	if err != nil {
		return nil, fmt.Errorf("reconcile move lines: %w", err)
	}
	return &ReconciliationResponse{ID: id, Date: date.Format(dateLayout)}, nil
}
