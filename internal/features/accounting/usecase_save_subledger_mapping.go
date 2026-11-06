package accounting

import (
	"context"
	"errors"
	"fmt"
)

type SaveSubledgerMappingUseCase struct {
	repo AccountingRepository
}

func NewSaveSubledgerMappingUseCase(repo AccountingRepository) *SaveSubledgerMappingUseCase {
	return &SaveSubledgerMappingUseCase{repo: repo}
}

func (uc *SaveSubledgerMappingUseCase) Execute(ctx context.Context, req SaveSubledgerEventMappingRequest) (*SubledgerEventMappingResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	journal, err := uc.repo.GetJournalByID(ctx, req.JournalID)
	if err != nil || journal == nil {
		return nil, ErrJournalNotFound
	}
	debitAcc, err := uc.repo.GetAccountByID(ctx, req.DebitAccountID)
	if err != nil || debitAcc == nil {
		return nil, errors.New("debit account not found")
	}
	creditAcc, err := uc.repo.GetAccountByID(ctx, req.CreditAccountID)
	if err != nil || creditAcc == nil {
		return nil, errors.New("credit account not found")
	}

	mapping := &SubledgerEventMapping{
		EventType:       req.EventType,
		JournalID:       req.JournalID,
		DebitAccountID:  req.DebitAccountID,
		CreditAccountID: req.CreditAccountID,
		Description:     req.Description,
		IsActive:        req.IsActive,
	}
	if err := uc.repo.SaveSubledgerEventMapping(ctx, mapping); err != nil {
		return nil, fmt.Errorf("save subledger mapping: %w", err)
	}
	res := ToSubledgerEventMappingResponse(mapping)
	return &res, nil
}

