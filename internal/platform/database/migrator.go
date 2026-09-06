package database

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationFile represents a parsed migration SQL file.
type MigrationFile struct {
	Version   int64
	Name      string
	Direction string // "up" or "down"
	Filename  string
}

// Migrator manages and applies database schema migrations.
type Migrator struct {
	pool   *pgxpool.Pool
	fsys   fs.FS
	dir    string
	logger *slog.Logger
}

// NewMigrator returns a new Migrator instance.
func NewMigrator(pool *pgxpool.Pool, fsys fs.FS, dir string, logger *slog.Logger) *Migrator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Migrator{
		pool:   pool,
		fsys:   fsys,
		dir:    dir,
		logger: logger,
	}
}

// EnsureSchemaTable creates the migrations tracking table if not present.
func (m *Migrator) EnsureSchemaTable(ctx context.Context) error {
	const query = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	_, err := m.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// Up runs all pending up migrations in sequential order.
func (m *Migrator) Up(ctx context.Context) (int, error) {
	if err := m.EnsureSchemaTable(ctx); err != nil {
		return 0, err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	allUp, err := m.findFiles("up")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, mf := range allUp {
		if applied[mf.Version] {
			continue
		}

		filePath := filepath.Join(m.dir, mf.Filename)
		content, err := fs.ReadFile(m.fsys, filePath)
		if err != nil {
			return count, fmt.Errorf("failed to read migration file %s: %w", mf.Filename, err)
		}

		m.logger.Info("applying migration", "version", mf.Version, "name", mf.Name)

		err = WithTx(ctx, m.pool, func(tx pgx.Tx) error {
			if _, execErr := tx.Exec(ctx, string(content)); execErr != nil {
				return fmt.Errorf("failed to execute SQL for migration %s: %w", mf.Filename, execErr)
			}

			const insert = `INSERT INTO schema_migrations (version, name) VALUES ($1, $2);`
			if _, insErr := tx.Exec(ctx, insert, mf.Version, mf.Name); insErr != nil {
				return fmt.Errorf("failed to record migration %s: %w", mf.Filename, insErr)
			}
			return nil
		})

		if err != nil {
			return count, err
		}

		count++
		m.logger.Info("migration applied successfully", "version", mf.Version, "name", mf.Name)
	}

	return count, nil
}

// Down rolls back the single most recently applied migration.
func (m *Migrator) Down(ctx context.Context) (int, error) {
	if err := m.EnsureSchemaTable(ctx); err != nil {
		return 0, err
	}

	const latestQuery = `SELECT version, name FROM schema_migrations ORDER BY version DESC LIMIT 1;`
	var latestVersion int64
	var latestName string
	err := m.pool.QueryRow(ctx, latestQuery).Scan(&latestVersion, &latestName)
	if err != nil {
		if err == pgx.ErrNoRows {
			m.logger.Info("no migrations to rollback")
			return 0, nil
		}
		return 0, fmt.Errorf("failed to query latest migration: %w", err)
	}

	allDown, err := m.findFiles("down")
	if err != nil {
		return 0, err
	}

	var targetFile *MigrationFile
	for _, f := range allDown {
		if f.Version == latestVersion {
			targetFile = &f
			break
		}
	}

	if targetFile == nil {
		return 0, fmt.Errorf("down migration file for version %d not found", latestVersion)
	}

	filePath := filepath.Join(m.dir, targetFile.Filename)
	content, err := fs.ReadFile(m.fsys, filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read down migration %s: %w", targetFile.Filename, err)
	}

	m.logger.Info("rolling back migration", "version", latestVersion, "name", latestName)

	err = WithTx(ctx, m.pool, func(tx pgx.Tx) error {
		if _, execErr := tx.Exec(ctx, string(content)); execErr != nil {
			return fmt.Errorf("failed to execute rollback SQL %s: %w", targetFile.Filename, execErr)
		}

		const deleteQuery = `DELETE FROM schema_migrations WHERE version = $1;`
		if _, delErr := tx.Exec(ctx, deleteQuery, latestVersion); delErr != nil {
			return fmt.Errorf("failed to delete migration record: %w", delErr)
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	m.logger.Info("migration rolled back successfully", "version", latestVersion)
	return 1, nil
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[int64]bool, error) {
	const query = `SELECT version FROM schema_migrations;`
	rows, err := m.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
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

func (m *Migrator) findFiles(direction string) ([]MigrationFile, error) {
	entries, err := fs.ReadDir(m.fsys, m.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory %s: %w", m.dir, err)
	}

	var files []MigrationFile
	targetSuffix := "." + direction + ".sql"

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		filename := e.Name()
		if !strings.HasSuffix(filename, targetSuffix) {
			continue
		}

		mf, err := ParseMigrationFilename(filename)
		if err != nil {
			continue
		}
		files = append(files, mf)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})

	return files, nil
}

// ParseMigrationFilename parses filename format: <version>_<name>.<direction>.sql
func ParseMigrationFilename(filename string) (MigrationFile, error) {
	parts := strings.Split(filename, ".")
	if len(parts) < 3 {
		return MigrationFile{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	direction := parts[len(parts)-2]
	base := strings.Join(parts[:len(parts)-2], ".")

	subParts := strings.SplitN(base, "_", 2)
	if len(subParts) < 2 {
		return MigrationFile{}, fmt.Errorf("missing version separator in: %s", filename)
	}

	version, err := strconv.ParseInt(subParts[0], 10, 64)
	if err != nil {
		return MigrationFile{}, fmt.Errorf("invalid version number in %s: %w", filename, err)
	}

	return MigrationFile{
		Version:   version,
		Name:      subParts[1],
		Direction: direction,
		Filename:  filename,
	}, nil
}
