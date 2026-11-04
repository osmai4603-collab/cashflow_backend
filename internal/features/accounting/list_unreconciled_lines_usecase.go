package accounting

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ListUnreconciledLinesUseCase struct {
	repo AccountingRepository
}

func NewListUnreconciledLinesUseCase(repo AccountingRepository) *ListUnreconciledLinesUseCase {
	return &ListUnreconciledLinesUseCase{repo: repo}
}

func (uc *ListUnreconciledLinesUseCase) Execute(
	ctx context.Context,
	filter UnreconciledLineFilter,
) ([]UnreconciledLineResponse, int, error) {
	if filter.AccountID != nil && *filter.AccountID == uuid.Nil {
		return nil, 0, fmt.Errorf("account_id must be a non-zero UUID")
	}
	filter.CurrencyID = strings.ToUpper(strings.TrimSpace(filter.CurrencyID))
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit < 1 || filter.Limit > 200 || filter.Offset < 0 {
		return nil, 0, fmt.Errorf("limit must be 1-200 and offset must be non-negative")
	}
	lines, total, err := uc.repo.ListUnreconciledLines(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list unreconciled lines: %w", err)
	}
	responses := make([]UnreconciledLineResponse, len(lines))
	for i, line := range lines {
		if line == nil {
			return nil, 0, fmt.Errorf("repository returned a nil unreconciled line at index %d", i)
		}
		responses[i] = ToUnreconciledLineResponse(line)
	}
	return responses, total, nil
}
