package domain

import "context"

// TransactionRepository defines the storage port for transactions.
// Implementations live in the adapters/storage layer.
type TransactionRepository interface {
	Save(ctx context.Context, tx *Transaction) error
	FindByID(ctx context.Context, id string) (*Transaction, error)
	FindAll(ctx context.Context) ([]*Transaction, error)
}
