package domain_test

import (
	"errors"
	"testing"
	"time"

	"cashflow_backend/internal/domain"
)

func TestNewTransaction_Valid(t *testing.T) {
	now := time.Now().UTC()
	tx, err := domain.NewTransaction("tx-1", 150.50, domain.TypeIncome, "Freelance payment", now)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if tx.ID != "tx-1" {
		t.Errorf("expected ID 'tx-1', got '%s'", tx.ID)
	}
	if tx.Amount != 150.50 {
		t.Errorf("expected Amount 150.50, got %f", tx.Amount)
	}
	if tx.Type != domain.TypeIncome {
		t.Errorf("expected Type '%s', got '%s'", domain.TypeIncome, tx.Type)
	}
	if tx.Description != "Freelance payment" {
		t.Errorf("expected Description 'Freelance payment', got '%s'", tx.Description)
	}
}

func TestNewTransaction_InvalidInvariants(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name        string
		id          string
		amount      float64
		txType      domain.TransactionType
		description string
		expectedErr error
	}{
		{
			name:        "empty id",
			id:          "",
			amount:      100,
			txType:      domain.TypeIncome,
			description: "valid",
			expectedErr: domain.ErrEmptyID,
		},
		{
			name:        "zero amount",
			id:          "tx-2",
			amount:      0,
			txType:      domain.TypeIncome,
			description: "valid",
			expectedErr: domain.ErrInvalidAmount,
		},
		{
			name:        "negative amount",
			id:          "tx-3",
			amount:      -25.0,
			txType:      domain.TypeExpense,
			description: "valid",
			expectedErr: domain.ErrInvalidAmount,
		},
		{
			name:        "invalid type",
			id:          "tx-4",
			amount:      50,
			txType:      domain.TransactionType("invalid_type"),
			description: "valid",
			expectedErr: domain.ErrInvalidType,
		},
		{
			name:        "empty description",
			id:          "tx-5",
			amount:      50,
			txType:      domain.TypeExpense,
			description: "",
			expectedErr: domain.ErrEmptyDescription,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewTransaction(tc.id, tc.amount, tc.txType, tc.description, now)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tc.expectedErr)
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}
