package accounting

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type CloseFiscalYearUseCase struct {
	repo AccountingRepository
}

func NewCloseFiscalYearUseCase(repo AccountingRepository) *CloseFiscalYearUseCase {
	return &CloseFiscalYearUseCase{repo: repo}
}

func (uc *CloseFiscalYearUseCase) Execute(
	ctx context.Context,
	fiscalYearID, actorID uuid.UUID,
	request CloseFiscalYearRequest,
) (*CloseFiscalYearResponse, error) {
	if fiscalYearID == uuid.Nil {
		return nil, errors.New("fiscal year id is required")
	}
	if actorID == uuid.Nil {
		return nil, ErrAccountingActorRequired
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	var retainedAccountID uuid.UUID
	if request.RetainedEarningsAccountID != nil {
		retainedAccountID = *request.RetainedEarningsAccountID
	}
	result, err := uc.repo.CloseFiscalYear(
		ctx, fiscalYearID, request.JournalID, retainedAccountID, actorID,
	)
	if err != nil {
		return nil, fmt.Errorf("close fiscal year: %w", err)
	}
	return &CloseFiscalYearResponse{
		FiscalYearID: fiscalYearID, SuccessorYearID: result.SuccessorYearID,
		ClosingMoveID: result.ClosingMoveID, OpeningMoveID: result.OpeningMoveID, State: "closed",
	}, nil
}
