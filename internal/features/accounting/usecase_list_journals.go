package accounting

import (
	"context"
	"fmt"
)

type ListJournalsUseCase struct {
	repo AccountingRepository
}

func NewListJournalsUseCase(repo AccountingRepository) *ListJournalsUseCase {
	return &ListJournalsUseCase{repo: repo}
}

func (uc *ListJournalsUseCase) Execute(ctx context.Context) ([]JournalResponse, error) {
	journals, err := uc.repo.ListJournals(ctx)
	if err != nil {
		return nil, fmt.Errorf("list journals: %w", err)
	}
	responses := make([]JournalResponse, len(journals))
	for i, journal := range journals {
		responses[i] = ToJournalResponse(journal)
	}
	return responses, nil
}
