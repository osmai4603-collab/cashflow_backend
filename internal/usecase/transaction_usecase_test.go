package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cashflow_backend/internal/domain"
	"cashflow_backend/internal/usecase"
)

// mockRepo is an in-memory mock implementing domain.TransactionRepository for usecase tests.
type mockRepo struct {
	items map[string]*domain.Transaction
}

func newMockRepo() *mockRepo {
	return &mockRepo{items: make(map[string]*domain.Transaction)}
}

func (m *mockRepo) Save(ctx context.Context, tx *domain.Transaction) error {
	m.items[tx.ID] = tx
	return nil
}

func (m *mockRepo) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	tx, ok := m.items[id]
	if !ok {
		return nil, domain.ErrTransactionNotFound
	}
	return tx, nil
}

func (m *mockRepo) FindAll(ctx context.Context) ([]*domain.Transaction, error) {
	res := make([]*domain.Transaction, 0, len(m.items))
	for _, tx := range m.items {
		res = append(res, tx)
	}
	return res, nil
}

func TestTransactionUseCase_CreateTransaction(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := usecase.NewTransactionUseCase(repo)

	input := usecase.CreateTransactionInput{
		Amount:      500.0,
		Type:        "income",
		Description: "Consulting",
	}

	tx, err := uc.CreateTransaction(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if tx.Amount != 500.0 || tx.Type != domain.TypeIncome {
		t.Errorf("unexpected transaction values: %+v", tx)
	}

	stored, err := repo.FindByID(ctx, tx.ID)
	if err != nil {
		t.Fatalf("expected transaction in repo, got err: %v", err)
	}
	if stored.ID != tx.ID {
		t.Errorf("stored ID %s != created ID %s", stored.ID, tx.ID)
	}
}

func TestTransactionUseCase_CreateTransaction_Invalid(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	uc := usecase.NewTransactionUseCase(repo)

	input := usecase.CreateTransactionInput{
		Amount:      -10,
		Type:        "income",
		Description: "Invalid",
	}

	_, err := uc.CreateTransaction(ctx, input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransactionUseCase_GetSummary(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	now := time.Now().UTC()

	tx1, _ := domain.NewTransaction("1", 1000, domain.TypeIncome, "Salary", now)
	tx2, _ := domain.NewTransaction("2", 300, domain.TypeExpense, "Groceries", now)
	tx3, _ := domain.NewTransaction("3", 200, domain.TypeIncome, "Dividend", now)

	_ = repo.Save(ctx, tx1)
	_ = repo.Save(ctx, tx2)
	_ = repo.Save(ctx, tx3)

	uc := usecase.NewTransactionUseCase(repo)
	summary, err := uc.GetSummary(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if summary.TotalIncome != 1200 {
		t.Errorf("expected TotalIncome 1200, got %f", summary.TotalIncome)
	}
	if summary.TotalExpense != 300 {
		t.Errorf("expected TotalExpense 300, got %f", summary.TotalExpense)
	}
	if summary.NetCashflow != 900 {
		t.Errorf("expected NetCashflow 900, got %f", summary.NetCashflow)
	}
	if summary.Count != 3 {
		t.Errorf("expected Count 3, got %d", summary.Count)
	}
}
