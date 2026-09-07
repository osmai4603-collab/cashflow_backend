package sequencestorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/sequence"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const selectSequenceFields = `
	id,
	name,
	code,
	COALESCE(prefix, ''),
	COALESCE(suffix, ''),
	padding,
	increment_by,
	start_number,
	current_number,
	sequence_type,
	COALESCE(date_range, ''),
	company_id,
	active,
	created_at,
	updated_at
`

// PostgresRepo implements sequence.Repository using pgxpool against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo constructs a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func scanSequence(row pgx.Row) (*sequence.Sequence, error) {
	var s sequence.Sequence
	var seqType, dateRange string
	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.Code,
		&s.Prefix,
		&s.Suffix,
		&s.Padding,
		&s.IncrementBy,
		&s.StartNumber,
		&s.CurrentNumber,
		&seqType,
		&dateRange,
		&s.CompanyID,
		&s.Active,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.SequenceType = sequence.SequenceType(seqType)
	s.DateRange = sequence.DateRangeGranularity(dateRange)
	return &s, nil
}

func (r *PostgresRepo) Create(ctx context.Context, s *sequence.Sequence) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO ir_sequences (
			name, code, prefix, suffix, padding, increment_by,
			start_number, current_number, sequence_type, date_range, company_id,
			active, created_at, updated_at
		) VALUES (
			$1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, NULLIF($10, ''), $11,
			$12, $13, $14
		) RETURNING id, created_at, updated_at
	`

	s.Active = true
	err := r.pool.QueryRow(ctx, query,
		s.Name, s.Code, s.Prefix, s.Suffix, s.Padding, s.IncrementBy,
		s.StartNumber, s.CurrentNumber, string(s.SequenceType), string(s.DateRange), s.CompanyID,
		s.Active, now, now,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create sequence", err)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*sequence.Sequence, error) {
	query := fmt.Sprintf(`SELECT %s FROM ir_sequences WHERE id = $1 AND active = true`, selectSequenceFields)

	s, err := scanSequence(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("sequence with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get sequence", err)
	}

	return s, nil
}

func (r *PostgresRepo) GetByCode(ctx context.Context, code string) (*sequence.Sequence, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM ir_sequences WHERE code = $1 AND active = true LIMIT 1`,
		selectSequenceFields,
	)

	s, err := scanSequence(r.pool.QueryRow(ctx, query, code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("sequence not found")
		}
		return nil, platformerrors.Internal("failed to get sequence by code", err)
	}

	return s, nil
}

func (r *PostgresRepo) Update(ctx context.Context, s *sequence.Sequence) error {
	query := `
		UPDATE ir_sequences SET
			name = $1,
			code = $2,
			prefix = NULLIF($3, ''),
			suffix = NULLIF($4, ''),
			padding = $5,
			increment_by = $6,
			start_number = $7,
			sequence_type = $8,
			date_range = NULLIF($9, ''),
			company_id = $10,
			updated_at = $11
		WHERE id = $12 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		s.Name, s.Code, s.Prefix, s.Suffix, s.Padding, s.IncrementBy,
		s.StartNumber, string(s.SequenceType), string(s.DateRange), s.CompanyID,
		time.Now().UTC(), s.ID,
	).Scan(&s.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("sequence with id %d not found", s.ID))
		}
		return platformerrors.Internal("failed to update sequence", err)
	}

	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE ir_sequences SET active = false, updated_at = $2 WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return platformerrors.Internal("failed to soft delete sequence", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("sequence with id %d not found", id))
	}

	return nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]sequence.Sequence, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM ir_sequences WHERE active = true ORDER BY id ASC`,
		selectSequenceFields,
	)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list sequences", err)
	}
	defer rows.Close()

	var items []sequence.Sequence
	for rows.Next() {
		item, err := scanSequence(rows)
		if err != nil {
			return nil, platformerrors.Internal("failed to scan sequence row", err)
		}
		items = append(items, *item)
	}

	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("error iterating sequence rows", err)
	}

	return items, nil
}

func (r *PostgresRepo) NextValue(ctx context.Context, id int64) (int, error) {
	var next int

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, platformerrors.Internal("failed to begin sequence transaction", err)
	}
	defer tx.Rollback(ctx)

	// Atomic lock: prevent concurrent number generation duplicates
	var current, incrementBy, startNumber int
	var seqType, dateRange string
	err = tx.QueryRow(ctx,
		`SELECT current_number, increment_by, start_number, sequence_type, date_range, active
		 FROM ir_sequences WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&current, &incrementBy, &startNumber, &seqType, &dateRange)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, platformerrors.NotFound(fmt.Sprintf("sequence with id %d not found", id))
		}
		return 0, platformerrors.Internal("failed to lock sequence", err)
	}

	if current == 0 {
		next = startNumber
	} else {
		next = current + incrementBy
	}

	_, err = tx.Exec(ctx,
		`UPDATE ir_sequences SET current_number = $1, updated_at = NOW() WHERE id = $2`,
		next, id,
	)
	if err != nil {
		return 0, platformerrors.Internal("failed to update sequence value", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, platformerrors.Internal("failed to commit sequence transaction", err)
	}

	return next, nil
}

func (r *PostgresRepo) Reset(ctx context.Context, id int64) error {
	query := `
		UPDATE ir_sequences
		SET current_number = start_number - increment_by, updated_at = NOW()
		WHERE id = $1 AND active = true
	`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to reset sequence", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("sequence with id %d not found", id))
	}

	return nil
}
