package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type UpdateAccountUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewUpdateAccountUseCase(repo AccountingRepository, logger *slog.Logger) *UpdateAccountUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UpdateAccountUseCase{repo: repo, logger: logger}
}

func (uc *UpdateAccountUseCase) Execute(
	ctx context.Context,
	accountID uuid.UUID,
	request UpdateAccountRequest,
) (*AccountResponse, error) {
	if accountID == uuid.Nil {
		return nil, errors.New("account id is required")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	account, err := uc.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}
	if request.ParentID != nil {
		if *request.ParentID == accountID {
			return nil, errors.New("an account cannot be its own parent")
		}
		parent, err := uc.repo.GetAccountByID(ctx, *request.ParentID)
		if err != nil {
			return nil, fmt.Errorf("get parent account: %w", err)
		}
		if parent == nil || !parent.IsActive {
			return nil, errors.New("parent account must exist and be active")
		}
	}
	account.Code = request.Code
	account.Name = request.Name
	account.Type = AccountType(request.Type)
	account.NormalBalance = NormalBalance(request.NormalBalance)
	account.CurrencyID = request.CurrencyID
	account.ParentID = cloneUUID(request.ParentID)
	account.IsReconcilable = request.IsReconcilable
	account.IsActive = request.IsActive
	account.IsCashEquivalent = request.IsCashEquivalent
	account.IsNonCashAdjustment = request.IsNonCashAdjustment
	account.CashFlowActivity = request.CashFlowActivity
	account.UpdatedAt = time.Now().UTC()
	if err := uc.repo.UpdateAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}
	response := ToAccountResponse(account)
	return &response, nil
}
