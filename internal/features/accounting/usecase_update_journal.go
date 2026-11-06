package accounting

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type UpdateJournalUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewUpdateJournalUseCase(repo AccountingRepository, logger *slog.Logger) *UpdateJournalUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UpdateJournalUseCase{repo: repo, logger: logger}
}

func (uc *UpdateJournalUseCase) Execute(ctx context.Context, id uuid.UUID, req UpdateJournalRequest) (*JournalResponse, error) {
	if id == uuid.Nil {
		return nil, ErrJournalNotFound
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	journal, err := uc.repo.GetJournalByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if journal == nil {
		return nil, ErrJournalNotFound
	}

	journal.Name = req.Name
	journal.SequencePrefix = req.SequencePrefix
	if req.IsActive != nil {
		journal.IsActive = *req.IsActive
	}

	if err := uc.repo.UpdateJournal(ctx, journal); err != nil {
		return nil, fmt.Errorf("update journal: %w", err)
	}
	res := ToJournalResponse(journal)
	return &res, nil
}

