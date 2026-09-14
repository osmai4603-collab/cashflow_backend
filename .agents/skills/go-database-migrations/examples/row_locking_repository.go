package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// =============================================================================
// Row-Level Locking (SELECT ... FOR UPDATE) with Deadlock Prevention
// =============================================================================

var (
	ErrInsufficientBalance = errors.New("insufficient account balance")
	ErrAccountNotFound     = errors.New("account not found")
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// TransferFunds transfers amount from source to target account with row locking and deadlock prevention.
func (r *AccountRepository) TransferFunds(ctx context.Context, fromID, toID string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	tx := ExtractTx(ctx)
	if tx == nil {
		return errors.New("transfer must be executed within an active transaction")
	}

	// 1. Consistent Lock Ordering: sort IDs lexicographically to prevent deadlocks
	firstID, secondID := fromID, toID
	if firstID > secondID {
		firstID, secondID = toID, fromID
	}

	// 2. Lock both accounts in deterministic order using SELECT FOR UPDATE
	if err := lockAccount(ctx, tx, firstID); err != nil {
		return fmt.Errorf("lock account %s: %w", firstID, err)
	}
	if err := lockAccount(ctx, tx, secondID); err != nil {
		return fmt.Errorf("lock account %s: %w", secondID, err)
	}

	// 3. Verify balance on source account
	var fromBalance float64
	queryBalance := `SELECT balance FROM accounts WHERE id = $1`
	if err := tx.QueryRowContext(ctx, queryBalance, fromID).Scan(&fromBalance); err != nil {
		return fmt.Errorf("query source balance: %w", err)
	}

	if fromBalance < amount {
		return ErrInsufficientBalance
	}

	// 4. Perform atomic balance updates
	debitQuery := `UPDATE accounts SET balance = balance - $1 WHERE id = $2`
	if _, err := tx.ExecContext(ctx, debitQuery, amount, fromID); err != nil {
		return fmt.Errorf("debit source: %w", err)
	}

	creditQuery := `UPDATE accounts SET balance = balance + $1 WHERE id = $2`
	if _, err := tx.ExecContext(ctx, creditQuery, amount, toID); err != nil {
		return fmt.Errorf("credit target: %w", err)
	}

	return nil
}

func lockAccount(ctx context.Context, tx *sql.Tx, accountID string) error {
	var dummy int
	query := `SELECT 1 FROM accounts WHERE id = $1 FOR UPDATE`
	err := tx.QueryRowContext(ctx, query, accountID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return ErrAccountNotFound
	}
	return err
}
