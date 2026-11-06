package accounting

import (
	"context"
	"fmt"
)

type GetAccountingConfigurationUseCase struct {
	repo AccountingRepository
}

func NewGetAccountingConfigurationUseCase(repo AccountingRepository) *GetAccountingConfigurationUseCase {
	return &GetAccountingConfigurationUseCase{repo: repo}
}

func (uc *GetAccountingConfigurationUseCase) Execute(ctx context.Context) (*AccountingConfigurationResponse, error) {
	config, err := uc.repo.GetAccountingConfiguration(ctx)
	if err != nil {
		return nil, fmt.Errorf("get accounting configuration: %w", err)
	}
	res := ToAccountingConfigurationResponse(config)
	return &res, nil
}

