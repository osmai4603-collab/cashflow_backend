package accounting

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ListMovesUseCase struct {
	repo AccountingRepository
}

func NewListMovesUseCase(repo AccountingRepository) *ListMovesUseCase {
	return &ListMovesUseCase{repo: repo}
}

func (uc *ListMovesUseCase) Execute(ctx context.Context, filter MoveFilter) ([]MoveResponse, int, error) {
	if filter.State != "" && filter.State != MoveStateDraft && filter.State != MoveStatePosted && filter.State != MoveStateReversed {
		return nil, 0, errors.New("state must be draft, posted, or reversed")
	}
	if filter.FromDate != nil && filter.ToDate != nil && filter.FromDate.After(*filter.ToDate) {
		return nil, 0, errors.New("from must not be after to")
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		return nil, 0, errors.New("limit must be between 1 and 200")
	}
	if filter.Offset < 0 {
		return nil, 0, errors.New("offset must not be negative")
	}
	if filter.FromDate != nil {
		*filter.FromDate = time.Date(filter.FromDate.Year(), filter.FromDate.Month(), filter.FromDate.Day(), 0, 0, 0, 0, time.UTC)
	}
	if filter.ToDate != nil {
		*filter.ToDate = time.Date(filter.ToDate.Year(), filter.ToDate.Month(), filter.ToDate.Day(), 0, 0, 0, 0, time.UTC)
	}
	moves, total, err := uc.repo.ListMoves(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list accounting moves: %w", err)
	}
	responses := make([]MoveResponse, len(moves))
	for i, move := range moves {
		responses[i] = ToMoveResponse(move)
	}
	return responses, total, nil
}
