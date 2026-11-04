package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GenerateBalanceSheetUseCase struct {
	repo AccountingRepository
}

func NewGenerateBalanceSheetUseCase(repo AccountingRepository) *GenerateBalanceSheetUseCase {
	return &GenerateBalanceSheetUseCase{repo: repo}
}

func (uc *GenerateBalanceSheetUseCase) Execute(ctx context.Context, asOf time.Time) (*BalanceSheetResponse, error) {
	rows, err := uc.repo.GetAccountBalancesAsOf(ctx, asOf)
	if err != nil {
		return nil, fmt.Errorf("get balance sheet balances: %w", err)
	}
	accounts, err := uc.repo.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get balance sheet accounts: %w", err)
	}
	byID := make(map[uuid.UUID]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			byID[account.ID] = account
		}
	}
	response := &BalanceSheetResponse{
		AsOfDate:    asOf.Format(dateLayout),
		Assets:      make([]StatementAccountLine, 0),
		Liabilities: make([]StatementAccountLine, 0),
		Equity:      make([]StatementAccountLine, 0),
	}
	for _, row := range rows {
		account, ok := byID[row.AccountID]
		if !ok {
			return nil, fmt.Errorf("balance sheet returned unknown account %s", row.AccountID)
		}
		switch account.Type {
		case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity:
			amount := row.EndingDebit.Sub(row.EndingCredit)
			if account.NormalBalance == NormalBalanceCredit {
				amount = amount.Neg()
			}
			line := StatementAccountLine{AccountID: account.ID, Code: account.Code, Name: account.Name, Amount: amount}
			switch account.Type {
			case AccountTypeAsset:
				response.Assets = append(response.Assets, line)
				response.TotalAssets = response.TotalAssets.Add(amount)
			case AccountTypeLiability:
				response.Liabilities = append(response.Liabilities, line)
				response.TotalLiabilities = response.TotalLiabilities.Add(amount)
			case AccountTypeEquity:
				response.Equity = append(response.Equity, line)
				response.TotalEquity = response.TotalEquity.Add(amount)
			}
		case AccountTypeIncome:
			response.CurrentPeriodIncome = response.CurrentPeriodIncome.Add(row.EndingCredit.Sub(row.EndingDebit))
		case AccountTypeExpense:
			response.CurrentPeriodIncome = response.CurrentPeriodIncome.Sub(row.EndingDebit.Sub(row.EndingCredit))
		}
	}
	response.TotalEquity = response.TotalEquity.Add(response.CurrentPeriodIncome)
	response.BalanceDifference = response.TotalAssets.Sub(response.TotalLiabilities).Sub(response.TotalEquity)
	response.IsBalanced = response.BalanceDifference.IsZero()
	return response, nil
}
