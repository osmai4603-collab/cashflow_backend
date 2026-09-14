package examples

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

// TestDatabase provides isolated transactions for integration tests.
type TestDatabase struct {
	db *sql.DB
}

// NewTestDatabase creates a test database harness wrapper.
func NewTestDatabase(db *sql.DB) *TestDatabase {
	return &TestDatabase{db: db}
}

// BeginTx starts an isolated transaction and registers an automatic rollback on test cleanup.
func (tdb *TestDatabase) BeginTx(t *testing.T) *sql.Tx {
	t.Helper()

	tx, err := tdb.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to begin test transaction: %v", err)
	}

	// Register rollback in test cleanup. Even if the test fails, panics,
	// or succeeds, the transaction will never be committed.
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("test transaction rollback warning: %v", err)
		}
	})

	return tx
}

// DummyRecord represents a record used in repository tests.
type DummyRecord struct {
	ID        string
	CompanyID string
	Name      string
}

// SampleRepository demonstrates a repository operating over an active transaction.
type SampleRepository struct {
	tx *sql.Tx
}

func NewSampleRepository(tx *sql.Tx) *SampleRepository {
	return &SampleRepository{tx: tx}
}

// Insert demonstrates inserting inside the test transaction.
func (r *SampleRepository) Insert(ctx context.Context, rec DummyRecord) error {
	query := `INSERT INTO sample_records (id, company_id, name) VALUES ($1, $2, $3)`
	_, err := r.tx.ExecContext(ctx, query, rec.ID, rec.CompanyID, rec.Name)
	return err
}

// ExampleTransactionalTest demonstrates the exact pattern to use in test files.
func ExampleTransactionalTest(t *testing.T, harness *TestDatabase) {
	t.Parallel()

	// 1. Begin isolated transaction
	tx := harness.BeginTx(t)

	// 2. Initialize repository with transaction
	repo := NewSampleRepository(tx)

	// 3. Execute repository action
	rec := DummyRecord{ID: "rec_1", CompanyID: "comp_100", Name: "Test Account"}
	if err := repo.Insert(context.Background(), rec); err != nil {
		t.Fatalf("failed to insert test record: %v", err)
	}

	// When test finishes, harness automatically rolls back the transaction.
}
