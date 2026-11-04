package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CreateAccountUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewCreateAccountUseCase(repo AccountingRepository, logger *slog.Logger) *CreateAccountUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &CreateAccountUseCase{repo: repo, logger: logger}
}

func (uc *CreateAccountUseCase) Execute(ctx context.Context, request CreateAccountRequest) (*AccountResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "create_account")
	if err := request.Validate(); err != nil {
		return nil, err
	}

	existing, err := uc.repo.GetAccountByCode(ctx, request.Code)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: %s", ErrDuplicateAccountCode, request.Code)
	}
	if err != nil && !errors.Is(err, ErrAccountNotFound) {
		return nil, fmt.Errorf("check account code: %w", err)
	}

	if request.ParentID != nil {
		parent, err := uc.repo.GetAccountByID(ctx, *request.ParentID)
		if err != nil {
			return nil, fmt.Errorf("get parent account: %w", err)
		}
		if parent == nil || !parent.IsActive {
			return nil, errors.New("parent account must exist and be active")
		}
	}

	now := time.Now().UTC()
	account := &Account{
		ID:                  uuid.New(),
		Code:                request.Code,
		Name:                request.Name,
		Type:                AccountType(request.Type),
		NormalBalance:       NormalBalance(request.NormalBalance),
		CurrencyID:          request.CurrencyID,
		ParentID:            cloneUUID(request.ParentID),
		IsReconcilable:      request.IsReconcilable,
		IsActive:            true,
		IsCashEquivalent:    request.IsCashEquivalent,
		IsNonCashAdjustment: request.IsNonCashAdjustment,
		CashFlowActivity:    request.CashFlowActivity,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := uc.repo.CreateAccount(ctx, account); err != nil {
		logger.ErrorContext(ctx, "account creation failed", slog.String("account_code", account.Code), slog.Any("error", err))
		return nil, fmt.Errorf("create account: %w", err)
	}

	logger.InfoContext(ctx, "account created", slog.String("account_id", account.ID.String()))
	response := ToAccountResponse(account)
	return &response, nil
}
