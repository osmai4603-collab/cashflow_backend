package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CreateJournalUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewCreateJournalUseCase(repo AccountingRepository, logger *slog.Logger) *CreateJournalUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &CreateJournalUseCase{repo: repo, logger: logger}
}

func (uc *CreateJournalUseCase) Execute(ctx context.Context, req CreateJournalRequest) (*JournalResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	existing, err := uc.repo.GetJournalByCode(ctx, req.Code)
	if err == nil && existing != nil {
		return nil, ErrDuplicateJournalCode
	}
	if err != nil && !errors.Is(err, ErrJournalNotFound) {
		return nil, fmt.Errorf("check journal code: %w", err)
	}

	journal := &Journal{
		ID:             uuid.New(),
		Code:           req.Code,
		Name:           req.Name,
		Type:           req.Type,
		SequencePrefix: req.SequencePrefix,
		NextNumber:     1,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
	}
	if err := uc.repo.CreateJournal(ctx, journal); err != nil {
		return nil, fmt.Errorf("create journal: %w", err)
	}
	res := ToJournalResponse(journal)
	return &res, nil
}

