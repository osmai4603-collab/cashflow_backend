package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/example/starter-service/internal/core/domain"
	"github.com/example/starter-service/internal/core/ports"
	"github.com/example/starter-service/internal/usecases"
)

// FakeWalletRepository بديل وهمي خفيف للاختبارات الأحادية
type FakeWalletRepository struct {
	wallets map[string]*domain.Wallet
}

func (f *FakeWalletRepository) GetByID(ctx context.Context, id string) (*domain.Wallet, error) {
	w, ok := f.wallets[id]
	if !ok {
		return nil, domain.ErrWalletNotFound
	}
	cp := *w
	return &cp, nil
}

func (f *FakeWalletRepository) Save(ctx context.Context, wallet *domain.Wallet) error {
	cp := *wallet
	f.wallets[wallet.ID] = &cp
	return nil
}

type FakeUoW struct {
	repo *FakeWalletRepository
}

func (u *FakeUoW) ExecuteInTx(ctx context.Context, fn func(txRepo ports.WalletRepository) error) error {
	return fn(u.repo)
}

func TestTransferFundsUseCase_Success(t *testing.T) {
	ctx := context.Background()

	w1, _ := domain.NewWallet("w-1", "user-1", "USD")
	_ = w1.Credit(5000)

	w2, _ := domain.NewWallet("w-2", "user-2", "USD")

	repo := &FakeWalletRepository{
		wallets: map[string]*domain.Wallet{
			"w-1": w1,
			"w-2": w2,
		},
	}
	uow := &FakeUoW{repo: repo}

	uc := usecases.NewTransferFundsUseCase(repo, uow)

	result, err := uc.Execute(ctx, usecases.TransferRequest{
		FromWalletID: "w-1",
		ToWalletID:   "w-2",
		Amount:       2000,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.NewFromBalance != 3000 {
		t.Errorf("expected new from balance to be 3000, got %d", result.NewFromBalance)
	}

	destWallet, _ := repo.GetByID(ctx, "w-2")
	if destWallet.Balance != 2000 {
		t.Errorf("expected destination balance to be 2000, got %d", destWallet.Balance)
	}
}

func TestTransferFundsUseCase_InsufficientBalance(t *testing.T) {
	ctx := context.Background()

	w1, _ := domain.NewWallet("w-1", "user-1", "USD")
	_ = w1.Credit(1000)

	w2, _ := domain.NewWallet("w-2", "user-2", "USD")

	repo := &FakeWalletRepository{
		wallets: map[string]*domain.Wallet{
			"w-1": w1,
			"w-2": w2,
		},
	}
	uow := &FakeUoW{repo: repo}

	uc := usecases.NewTransferFundsUseCase(repo, uow)

	_, err := uc.Execute(ctx, usecases.TransferRequest{
		FromWalletID: "w-1",
		ToWalletID:   "w-2",
		Amount:       5000,
	})

	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}
