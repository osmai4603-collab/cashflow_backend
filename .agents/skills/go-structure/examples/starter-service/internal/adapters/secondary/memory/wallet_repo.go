package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/starter-service/internal/core/domain"
	"github.com/example/starter-service/internal/core/ports"
)

// InMemWalletRepository محول ثانوي (Secondary Adapter) يخزن البيانات في الذاكرة
// يطابق واجهات ports.WalletRepository و ports.UnitOfWork
type InMemWalletRepository struct {
	mu      sync.RWMutex
	wallets map[string]*domain.Wallet
}

// NewInMemWalletRepository ينشئ مستودعاً في الذاكرة مع تعبئة بيانات أولية
func NewInMemWalletRepository() *InMemWalletRepository {
	repo := &InMemWalletRepository{
		wallets: make(map[string]*domain.Wallet),
	}

	// بذر بيانات أولية للتجربة (Seed initial wallets)
	w1, _ := domain.NewWallet("wallet-1", "user-alice", "USD")
	_ = w1.Credit(10000) // $100.00
	repo.wallets[w1.ID] = w1

	w2, _ := domain.NewWallet("wallet-2", "user-bob", "USD")
	_ = w2.Credit(2500) // $25.00
	repo.wallets[w2.ID] = w2

	return repo
}

// GetByID استرجاع المحفظة بالمعرف
func (r *InMemWalletRepository) GetByID(ctx context.Context, id string) (*domain.Wallet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	w, exists := r.wallets[id]
	if !exists {
		return nil, domain.ErrWalletNotFound
	}

	// إرجاع نسخة عميقة لمنع التعديل غير المحمي بالذاكرة
	copied := *w
	return &copied, nil
}

// Save حفظ أو تحديث المحفظة
func (r *InMemWalletRepository) Save(ctx context.Context, wallet *domain.Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copied := *wallet
	r.wallets[wallet.ID] = &copied
	return nil
}

// ExecuteInTx تنفيذ المعاملة الحسابية بقفل عام محاكي للمعاملات الذرية
func (r *InMemWalletRepository) ExecuteInTx(ctx context.Context, fn func(txRepo ports.WalletRepository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// لقطة ذاكرة للتراجع في حال الفشل (Snapshot for rollback)
	snapshot := make(map[string]*domain.Wallet, len(r.wallets))
	for k, v := range r.wallets {
		cp := *v
		snapshot[k] = &cp
	}

	// ممرر داخلي لا يعيد استخدام أقفال جديدة لمنع الـ Deadlock
	txAdapter := &txRepoAdapter{repo: r}

	if err := fn(txAdapter); err != nil {
		r.wallets = snapshot // Rollback
		return fmt.Errorf("transaction rolled back: %w", err)
	}

	return nil
}

type txRepoAdapter struct {
	repo *InMemWalletRepository
}

func (t *txRepoAdapter) GetByID(ctx context.Context, id string) (*domain.Wallet, error) {
	w, exists := t.repo.wallets[id]
	if !exists {
		return nil, domain.ErrWalletNotFound
	}
	copied := *w
	return &copied, nil
}

func (t *txRepoAdapter) Save(ctx context.Context, wallet *domain.Wallet) error {
	copied := *wallet
	t.repo.wallets[wallet.ID] = &copied
	return nil
}
