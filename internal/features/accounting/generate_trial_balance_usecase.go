package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

type GenerateTrialBalanceUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewGenerateTrialBalanceUseCase(repo AccountingRepository, logger *slog.Logger) *GenerateTrialBalanceUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GenerateTrialBalanceUseCase{repo: repo, logger: logger}
}

func (uc *GenerateTrialBalanceUseCase) Execute(ctx context.Context, from, to time.Time) (*TrialBalanceResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "generate_trial_balance")
	if from.IsZero() || to.IsZero() {
		return nil, errors.New("trial balance from and to dates are required")
	}
	if from.After(to) {
		return nil, errors.New("trial balance from date must not be after to date")
	}
	rows, err := uc.repo.GetTrialBalanceData(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("get trial balance data: %w", err)
	}

	var totalDebit, totalCredit = decimal.Zero, decimal.Zero
	for i, row := range rows {
		if row == nil {
			return nil, fmt.Errorf("trial balance repository returned a nil row at index %d", i)
		}
		totalDebit = totalDebit.Add(row.EndingDebit)
		totalCredit = totalCredit.Add(row.EndingCredit)
	}
	response := ToTrialBalanceResponse(from, to, totalDebit, totalCredit, rows)
	logger.DebugContext(ctx, "trial balance generated",
		slog.Int("account_count", len(rows)),
		slog.Bool("is_balanced", response.IsBalanced),
	)
	return &response, nil
}
