package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/starter-service/internal/core/domain"
	"github.com/example/starter-service/internal/core/ports"
)

// TransferRequest يمثل بيانات دخل حالة الاستخدام
type TransferRequest struct {
	FromWalletID string
	ToWalletID   string
	Amount       int64
}

// TransferResult يمثل نتيجة إتمام حالة الاستخدام
type TransferResult struct {
	FromWalletID   string `json:"from_wallet_id"`
	ToWalletID     string `json:"to_wallet_id"`
	Amount         int64  `json:"amount"`
	NewFromBalance int64  `json:"new_from_balance"`
}

// TransferFundsUseCase حالة الاستخدام لتحويل الأموال بين محفظتين
type TransferFundsUseCase struct {
	repo ports.WalletRepository
	uow  ports.UnitOfWork
}

// NewTransferFundsUseCase المنشئ الذي يقبل الواجهات كمعاملات (Accept interfaces)
func NewTransferFundsUseCase(repo ports.WalletRepository, uow ports.UnitOfWork) *TransferFundsUseCase {
	return &TransferFundsUseCase{
		repo: repo,
		uow:  uow,
	}
}

// Execute تنفيذ العملية التجارية داخل معاملة ذرية متناسقة
func (uc *TransferFundsUseCase) Execute(ctx context.Context, req TransferRequest) (*TransferResult, error) {
	if req.FromWalletID == "" || req.ToWalletID == "" {
		return nil, errors.New("source and destination wallet IDs must be provided")
	}
	if req.FromWalletID == req.ToWalletID {
		return nil, domain.ErrSameAccountTransfer
	}
	if req.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	var result TransferResult

	// تنفيذ العمليات عبر منفذ UnitOfWork لضمان سلامة المعاملة الذرية
	err := uc.uow.ExecuteInTx(ctx, func(txRepo ports.WalletRepository) error {
		fromWallet, err := txRepo.GetByID(ctx, req.FromWalletID)
		if err != nil {
			return fmt.Errorf("failed to fetch source wallet: %w", err)
		}

		toWallet, err := txRepo.GetByID(ctx, req.ToWalletID)
		if err != nil {
			return fmt.Errorf("failed to fetch destination wallet: %w", err)
		}

		if err := fromWallet.Debit(req.Amount); err != nil {
			return fmt.Errorf("debit error: %w", err)
		}

		if err := toWallet.Credit(req.Amount); err != nil {
			return fmt.Errorf("credit error: %w", err)
		}

		if err := txRepo.Save(ctx, fromWallet); err != nil {
			return fmt.Errorf("failed to update source wallet: %w", err)
		}

		if err := txRepo.Save(ctx, toWallet); err != nil {
			return fmt.Errorf("failed to update destination wallet: %w", err)
		}

		result = TransferResult{
			FromWalletID:   fromWallet.ID,
			ToWalletID:     toWallet.ID,
			Amount:         req.Amount,
			NewFromBalance: fromWallet.Balance,
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("transfer failed: %w", err)
	}

	return &result, nil
}
