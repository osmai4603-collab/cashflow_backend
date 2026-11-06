package accounting

import (
	"context"
	"fmt"
)

type ListSubledgerEventsUseCase struct {
	repo AccountingRepository
}

func NewListSubledgerEventsUseCase(repo AccountingRepository) *ListSubledgerEventsUseCase {
	return &ListSubledgerEventsUseCase{repo: repo}
}

func (uc *ListSubledgerEventsUseCase) Execute(ctx context.Context, limit, offset int) (*ListProcessedEventsResponse, error) {
	events, total, err := uc.repo.ListProcessedEvents(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list processed events: %w", err)
	}
	responses := make([]ProcessedEventSummaryResponse, len(events))
	for i, e := range events {
		responses[i] = ToProcessedEventSummaryResponse(e)
	}
	return &ListProcessedEventsResponse{
		Events: responses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

