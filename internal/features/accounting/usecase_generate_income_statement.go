package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GenerateIncomeStatementUseCase struct {
	repo AccountingRepository
}

func NewGenerateIncomeStatementUseCase(repo AccountingRepository) *GenerateIncomeStatementUseCase {
	return &GenerateIncomeStatementUseCase{repo: repo}
}

func (uc *GenerateIncomeStatementUseCase) Execute(
	ctx context.Context,
	fromDate, toDate time.Time,
) (*IncomeStatementResponse, error) {
	if fromDate.After(toDate) {
		return nil, fmt.Errorf("from date must not be after to date")
	}
	rows, err := uc.repo.GetTrialBalanceData(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get income statement balances: %w", err)
	}
	accounts, err := uc.repo.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get income statement accounts: %w", err)
	}
	byID := make(map[uuid.UUID]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			byID[account.ID] = account
		}
	}
	response := &IncomeStatementResponse{
		FromDate: fromDate.Format(dateLayout), ToDate: toDate.Format(dateLayout),
		Revenue: make([]StatementAccountLine, 0), Expenses: make([]StatementAccountLine, 0),
	}
	for _, row := range rows {
		account, ok := byID[row.AccountID]
		if !ok {
			return nil, fmt.Errorf("trial balance returned unknown account %s", row.AccountID)
		}
		switch account.Type {
		case AccountTypeIncome:
			amount := row.PeriodCredit.Sub(row.PeriodDebit)
			response.Revenue = append(response.Revenue, StatementAccountLine{
				AccountID: account.ID, Code: account.Code, Name: account.Name, Amount: amount,
			})
			response.TotalRevenue = response.TotalRevenue.Add(amount)
		case AccountTypeExpense:
			amount := row.PeriodDebit.Sub(row.PeriodCredit)
			response.Expenses = append(response.Expenses, StatementAccountLine{
				AccountID: account.ID, Code: account.Code, Name: account.Name, Amount: amount,
			})
			response.TotalExpenses = response.TotalExpenses.Add(amount)
		}
	}
	response.NetIncome = response.TotalRevenue.Sub(response.TotalExpenses)
	return response, nil
}
