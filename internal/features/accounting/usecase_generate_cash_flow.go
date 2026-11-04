package accounting

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GenerateCashFlowUseCase struct {
	repo        AccountingRepository
	incomeStats *GenerateIncomeStatementUseCase
}

func NewGenerateCashFlowUseCase(repo AccountingRepository) *GenerateCashFlowUseCase {
	return &GenerateCashFlowUseCase{
		repo: repo, incomeStats: NewGenerateIncomeStatementUseCase(repo),
	}
}

func (uc *GenerateCashFlowUseCase) Execute(
	ctx context.Context,
	fromDate, toDate time.Time,
) (*CashFlowStatementResponse, error) {
	if fromDate.After(toDate) {
		return nil, errors.New("from date must not be after to date")
	}
	balances, err := uc.repo.GetTrialBalanceData(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get cash flow balances: %w", err)
	}
	accounts, err := uc.repo.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cash flow accounts: %w", err)
	}
	income, err := uc.incomeStats.Execute(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get cash flow net income: %w", err)
	}
	byID := make(map[uuid.UUID]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			byID[account.ID] = account
		}
	}
	response := &CashFlowStatementResponse{
		FromDate: fromDate.Format(dateLayout), ToDate: toDate.Format(dateLayout),
		NetIncome: income.NetIncome,
	}
	for _, row := range balances {
		account, ok := byID[row.AccountID]
		if !ok {
			return nil, fmt.Errorf("cash flow balance returned unknown account %s", row.AccountID)
		}
		periodNet := row.PeriodDebit.Sub(row.PeriodCredit)
		if account.IsCashEquivalent {
			response.NetChangeInCash = response.NetChangeInCash.Add(periodNet)
			continue
		}
		if account.IsNonCashAdjustment {
			response.NonCashAdjustments = response.NonCashAdjustments.Add(periodNet)
		}
		if account.CashFlowActivity != "operating" {
			continue
		}
		switch account.Type {
		case AccountTypeAsset:
			response.WorkingCapitalChanges = response.WorkingCapitalChanges.Sub(periodNet)
		case AccountTypeLiability:
			response.WorkingCapitalChanges = response.WorkingCapitalChanges.Add(periodNet.Neg())
		}
	}
	response.OperatingCashFlow = response.NetIncome.
		Add(response.NonCashAdjustments).
		Add(response.WorkingCapitalChanges)

	movements, err := uc.repo.GetCashFlowMovements(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get classified cash movements: %w", err)
	}
	for _, movement := range movements {
		if movement == nil {
			return nil, errors.New("cash flow repository returned a nil movement")
		}
		if movement.Amount.IsZero() {
			continue
		}
		if movement.HasUnclassified || len(movement.Activities) == 0 {
			response.UnclassifiedCashMovement = response.UnclassifiedCashMovement.Add(movement.Amount)
			continue
		}
		if len(movement.Activities) != 1 {
			return nil, fmt.Errorf("move %s has cash counterparts in multiple IAS 7 activities", movement.MoveID)
		}
		switch movement.Activities[0] {
		case "operating":
		case "investing":
			response.InvestingCashFlow = response.InvestingCashFlow.Add(movement.Amount)
		case "financing":
			response.FinancingCashFlow = response.FinancingCashFlow.Add(movement.Amount)
		default:
			return nil, fmt.Errorf("move %s has invalid cash flow activity %q", movement.MoveID, movement.Activities[0])
		}
	}
	calculated := response.OperatingCashFlow.
		Add(response.InvestingCashFlow).
		Add(response.FinancingCashFlow).
		Add(response.UnclassifiedCashMovement)
	response.ReconciliationDifference = response.NetChangeInCash.Sub(calculated)
	response.IsReconciled = response.ReconciliationDifference.IsZero()
	return response, nil
}
