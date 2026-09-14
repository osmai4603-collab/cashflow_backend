package migrations

import (
	"context"
	"database/sql"
	"fmt"
)

// =============================================================================
// Unit of Work (Transaction Context Propagation)
// =============================================================================

type txContextKey struct{}

// UnitOfWork coordinates transactions without leaking sql.Tx to business logic.
type UnitOfWork struct {
	db *sql.DB
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

// Execute wraps a business operation inside an atomic database transaction.
func (u *UnitOfWork) Execute(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx, err := u.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed (%v) after error: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ExtractTx allows repository adapters to safely acquire the active transaction if present.
func ExtractTx(ctx context.Context) *sql.Tx {
	if val := ctx.Value(txContextKey{}); val != nil {
		if tx, ok := val.(*sql.Tx); ok {
			return tx
		}
	}
	return nil
}
