package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// =============================================================================
// Idempotent Transactional Schema Migration Runner
// =============================================================================

type Migration struct {
	Version int64
	Name    string
	UpSQL   string
	DownSQL string
}

type Runner struct {
	db *sql.DB
}

func NewRunner(db *sql.DB) *Runner {
	return &Runner{db: db}
}

// Bootstrap creates the tracking table if it does not exist.
func (r *Runner) Bootstrap(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
		);
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// Up applies all pending migrations in strictly ascending version order.
func (r *Runner) Up(ctx context.Context, list []Migration) error {
	if err := r.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap schema_migrations: %w", err)
	}

	applied, err := r.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("retrieve applied versions: %w", err)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Version < list[j].Version
	})

	for _, m := range list {
		if applied[m.Version] {
			continue
		}

		if err := r.applyOne(ctx, m); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err)
		}
	}

	return nil
}

func (r *Runner) applyOne(ctx context.Context, m Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, m.UpSQL); err != nil {
		return fmt.Errorf("execute ddl: %w", err)
	}

	recordQuery := `INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)`
	if _, err := tx.ExecContext(ctx, recordQuery, m.Version, m.Name, time.Now()); err != nil {
		return fmt.Errorf("record migration version: %w", err)
	}

	return tx.Commit()
}

func (r *Runner) getAppliedVersions(ctx context.Context) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}
