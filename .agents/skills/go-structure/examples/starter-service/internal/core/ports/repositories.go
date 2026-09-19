package ports

import (
	"context"

	"github.com/example/starter-service/internal/core/domain"
)

// WalletRepository واجهة منفذ التخزين (Secondary Port)
// يحدد النطاق ما يحتاجه من التخزين دون معرفة نوع قاعدة البيانات
type WalletRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Wallet, error)
	Save(ctx context.Context, wallet *domain.Wallet) error
}

// UnitOfWork واجهة تنسيق المعاملات المالية الحسابية
type UnitOfWork interface {
	ExecuteInTx(ctx context.Context, fn func(txRepo WalletRepository) error) error
}
