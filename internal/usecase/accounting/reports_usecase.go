package accountingusecase

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/accounting"
)

func (uc *UseCase) GetTrialBalance(ctx context.Context, fromDate, toDate time.Time, onlyPosted bool) (*accounting.TrialBalanceReport, error) {
	return uc.repo.GetTrialBalance(ctx, fromDate, toDate, onlyPosted)
}

func (uc *UseCase) GetProfitAndLoss(ctx context.Context, fromDate, toDate time.Time) (*accounting.ProfitAndLossReport, error) {
	return uc.repo.GetProfitAndLoss(ctx, fromDate, toDate)
}

func (uc *UseCase) GetBalanceSheet(ctx context.Context, asOfDate time.Time) (*accounting.BalanceSheetReport, error) {
	if asOfDate.IsZero() {
		asOfDate = time.Now().UTC()
	}
	return uc.repo.GetBalanceSheet(ctx, asOfDate)
}

func (uc *UseCase) GetGeneralLedger(ctx context.Context, accountID *int64, partnerID *int64, fromDate, toDate *time.Time) ([]accounting.GeneralLedgerItem, error) {
	return uc.repo.GetGeneralLedger(ctx, accountID, partnerID, fromDate, toDate)
}
