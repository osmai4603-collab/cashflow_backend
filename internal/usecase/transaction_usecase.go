package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"cashflow_backend/internal/domain"
)

type CreateTransactionInput struct {
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
}

type Summary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetCashflow  float64 `json:"net_cashflow"`
	Count        int     `json:"count"`
}

type TransactionUseCase struct {
	repo domain.TransactionRepository
}

func NewTransactionUseCase(repo domain.TransactionRepository) *TransactionUseCase {
	return &TransactionUseCase{repo: repo}
}

func (uc *TransactionUseCase) CreateTransaction(ctx context.Context, input CreateTransactionInput) (*domain.Transaction, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate transaction id: %w", err)
	}

	txType := domain.TransactionType(input.Type)
	tx, err := domain.NewTransaction(id, input.Amount, txType, input.Description, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	return tx, nil
}

func (uc *TransactionUseCase) GetTransaction(ctx context.Context, id string) (*domain.Transaction, error) {
	if id == "" {
		return nil, domain.ErrEmptyID
	}
	return uc.repo.FindByID(ctx, id)
}

func (uc *TransactionUseCase) ListTransactions(ctx context.Context) ([]*domain.Transaction, error) {
	return uc.repo.FindAll(ctx)
}

func (uc *TransactionUseCase) GetSummary(ctx context.Context) (*Summary, error) {
	txs, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate summary: %w", err)
	}

	var income, expense float64
	for _, tx := range txs {
		switch tx.Type {
		case domain.TypeIncome:
			income += tx.Amount
		case domain.TypeExpense:
			expense += tx.Amount
		}
	}

	return &Summary{
		TotalIncome:  income,
		TotalExpense: expense,
		NetCashflow:  income - expense,
		Count:        len(txs),
	}, nil
}

func generateID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "tx_" + hex.EncodeToString(bytes), nil
}
