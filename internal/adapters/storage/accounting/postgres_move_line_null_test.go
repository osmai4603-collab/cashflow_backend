package accountingstorage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool connects to the local PostgreSQL instance, skipping when unavailable.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("CASHFLOW_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/cashflow?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping: cannot parse database url: %v", err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	if err := pool.Ping(ctx2); err != nil {
		pool.Close()
		t.Skipf("skipping: database is unreachable: %v", err)
	}
	return pool
}

// TestPostgresRepo_GetMoveWithLines_ScansNullDisplayType guards the fix for the
// "failed to scan move line" 500: account_move_lines.display_type is nullable
// but was scanned directly into a Go string.
func TestPostgresRepo_GetMoveWithLines_ScansNullDisplayType(t *testing.T) {
	pool := testPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	// Find a move that has a line with display_type IS NULL to exercise the
	// exact production failure path.
	var moveID int64
	err := pool.QueryRow(ctx, `
		SELECT l.move_id
		FROM account_move_lines l
		WHERE l.display_type IS NULL
		ORDER BY l.move_id
		LIMIT 1
	`).Scan(&moveID)
	if err != nil {
		t.Skip("skipping: no move line with NULL display_type in database")
	}

	repo := NewPostgresRepo(pool)
	move, err := repo.GetMoveWithLines(ctx, moveID)
	if err != nil {
		t.Fatalf("GetMoveWithLines must not fail on NULL display_type: %v", err)
	}
	if len(move.Lines) == 0 {
		t.Fatalf("expected at least one line for move %d", moveID)
	}
	for _, l := range move.Lines {
		if l.DisplayType != "" {
			t.Fatalf("expected NULL display_type to be read as empty string, got %q (line %d)", l.DisplayType, l.ID)
		}
	}
}

// TestPostgresRepo_GetMoveLineByID_ScansNullDisplayType guards the shared
// moveLineColumns constant used by the single-line fetch.
func TestPostgresRepo_GetMoveLineByID_ScansNullDisplayType(t *testing.T) {
	pool := testPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	var lineID int64
	err := pool.QueryRow(ctx, `
		SELECT l.id
		FROM account_move_lines l
		WHERE l.display_type IS NULL
		ORDER BY l.id
		LIMIT 1
	`).Scan(&lineID)
	if err != nil {
		t.Skip("skipping: no move line with NULL display_type in database")
	}

	repo := NewPostgresRepo(pool)
	line, err := repo.GetMoveLineByID(ctx, lineID)
	if err != nil {
		t.Fatalf("GetMoveLineByID must not fail on NULL display_type: %v", err)
	}
	if line.DisplayType != "" {
		t.Fatalf("expected NULL display_type to be read as empty string, got %q", line.DisplayType)
	}
}