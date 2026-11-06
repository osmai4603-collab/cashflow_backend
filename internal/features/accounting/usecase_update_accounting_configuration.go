package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type UpdateAccountingConfigurationUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewUpdateAccountingConfigurationUseCase(repo AccountingRepository, logger *slog.Logger) *UpdateAccountingConfigurationUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UpdateAccountingConfigurationUseCase{repo: repo, logger: logger}
}

func (uc *UpdateAccountingConfigurationUseCase) Execute(ctx context.Context, req UpdateAccountingConfigurationRequest) (*AccountingConfigurationResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	retained, err := uc.repo.GetAccountByID(ctx, req.RetainedEarningsAccountID)
	if err != nil {
		return nil, fmt.Errorf("retained earnings account: %w", err)
	}
	if retained == nil || retained.Type != AccountTypeEquity {
		return nil, errors.New("retained earnings account must exist and have equity type")
	}

	fxGain, err := uc.repo.GetAccountByID(ctx, req.FXGainAccountID)
	if err != nil {
		return nil, fmt.Errorf("fx gain account: %w", err)
	}
	if fxGain == nil || fxGain.Type != AccountTypeIncome {
		return nil, errors.New("fx gain account must exist and have income type")
	}

	fxLoss, err := uc.repo.GetAccountByID(ctx, req.FXLossAccountID)
	if err != nil {
		return nil, fmt.Errorf("fx loss account: %w", err)
	}
	if fxLoss == nil || fxLoss.Type != AccountTypeExpense {
		return nil, errors.New("fx loss account must exist and have expense type")
	}

	journal, err := uc.repo.GetJournalByID(ctx, req.FXReconciliationJournalID)
	if err != nil {
		return nil, fmt.Errorf("fx reconciliation journal: %w", err)
	}
	if journal == nil || !journal.IsActive {
		return nil, errors.New("fx reconciliation journal must exist and be active")
	}

	config := &AccountingConfiguration{
		RetainedEarningsAccountID: req.RetainedEarningsAccountID,
		FXGainAccountID:           req.FXGainAccountID,
		FXLossAccountID:           req.FXLossAccountID,
		FXReconciliationJournalID: req.FXReconciliationJournalID,
		FunctionalCurrency:        req.FunctionalCurrency,
	}

	if err := uc.repo.UpdateAccountingConfiguration(ctx, config); err != nil {
		return nil, fmt.Errorf("update accounting configuration: %w", err)
	}
	res := ToAccountingConfigurationResponse(config)
	return &res, nil
}

