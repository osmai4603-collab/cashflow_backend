package accounting

import (
	"context"
	"fmt"
)

type ListSubledgerMappingsUseCase struct {
	repo AccountingRepository
}

func NewListSubledgerMappingsUseCase(repo AccountingRepository) *ListSubledgerMappingsUseCase {
	return &ListSubledgerMappingsUseCase{repo: repo}
}

func (uc *ListSubledgerMappingsUseCase) Execute(ctx context.Context) ([]SubledgerEventMappingResponse, error) {
	mappings, err := uc.repo.ListSubledgerEventMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subledger event mappings: %w", err)
	}
	responses := make([]SubledgerEventMappingResponse, len(mappings))
	for i, m := range mappings {
		responses[i] = ToSubledgerEventMappingResponse(m)
	}
	return responses, nil
}

