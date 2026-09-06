package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrEmptyDescription    = errors.New("description cannot be empty")
	ErrInvalidType         = errors.New("invalid transaction type: must be 'income' or 'expense'")
	ErrEmptyID             = errors.New("transaction id cannot be empty")
	ErrTransactionNotFound = errors.New("transaction not found")
)

type TransactionType string

const (
	TypeIncome  TransactionType = "income"
	TypeExpense TransactionType = "expense"
)

// Transaction is a pure domain entity representing a financial cash flow record.
type Transaction struct {
	ID          string          `json:"id"`
	Amount      float64         `json:"amount"`
	Type        TransactionType `json:"type"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
}

// NewTransaction validates domain invariants and creates a new Transaction.
func NewTransaction(id string, amount float64, txType TransactionType, description string, createdAt time.Time) (*Transaction, error) {
	if id == "" {
		return nil, ErrEmptyID
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if txType != TypeIncome && txType != TypeExpense {
		return nil, ErrInvalidType
	}
	if description == "" {
		return nil, ErrEmptyDescription
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &Transaction{
		ID:          id,
		Amount:      amount,
		Type:        txType,
		Description: description,
		CreatedAt:   createdAt,
	}, nil
}
