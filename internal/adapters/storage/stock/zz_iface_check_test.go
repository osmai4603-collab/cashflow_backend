package stockstorage_test

import (
	"testing"

	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/stock"
)

func TestIfaceCheck(t *testing.T) {
	var _ stock.Repository = (*stockstorage.MemoryRepo)(nil)
	var _ stock.Repository = (*stockstorage.PostgresRepo)(nil)
}
