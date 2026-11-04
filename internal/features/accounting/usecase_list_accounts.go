package accounting

import (
	"context"
	"fmt"
	"log/slog"
)

type ListAccountsUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewListAccountsUseCase(repo AccountingRepository, logger *slog.Logger) *ListAccountsUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ListAccountsUseCase{repo: repo, logger: logger}
}

func (uc *ListAccountsUseCase) Execute(ctx context.Context) ([]AccountResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "list_accounts")
	accounts, err := uc.repo.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	responses := make([]AccountResponse, 0, len(accounts))
	for i, account := range accounts {
		if account == nil {
			return nil, fmt.Errorf("account repository returned a nil account at index %d", i)
		}
		responses = append(responses, ToAccountResponse(account))
	}
	logger.DebugContext(ctx, "accounts listed", slog.Int("account_count", len(responses)))
	return responses, nil
}
