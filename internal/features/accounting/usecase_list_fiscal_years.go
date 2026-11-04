package accounting

import (
	"context"
	"fmt"
)

type ListFiscalYearsUseCase struct {
	repo AccountingRepository
}

func NewListFiscalYearsUseCase(repo AccountingRepository) *ListFiscalYearsUseCase {
	return &ListFiscalYearsUseCase{repo: repo}
}

func (uc *ListFiscalYearsUseCase) Execute(ctx context.Context) ([]FiscalYearResponse, error) {
	years, err := uc.repo.ListFiscalYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("list fiscal years: %w", err)
	}
	responses := make([]FiscalYearResponse, len(years))
	for i, summary := range years {
		if summary == nil || summary.Year == nil {
			return nil, fmt.Errorf("fiscal year repository returned a nil summary at index %d", i)
		}
		responses[i] = ToFiscalYearResponse(summary.Year, summary.Periods)
	}
	return responses, nil
}
