package database_test

import (
	"context"
	"testing"

	"cashflow_backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

func TestWithTx_NilPool(t *testing.T) {
	err := database.WithTx(context.Background(), nil, func(tx pgx.Tx) error {
		return nil
	})
	if err == nil {
		t.Errorf("expected error when pool is nil")
	}
}
