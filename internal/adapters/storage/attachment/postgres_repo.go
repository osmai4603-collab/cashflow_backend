package attachmentstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/attachment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const selectAttachmentFields = `
	id,
	name,
	filename,
	mimetype,
	file_size,
	COALESCE(checksum, ''),
	COALESCE(storage_path, ''),
	COALESCE(res_model, ''),
	res_id,
	COALESCE(description, ''),
	company_id,
	active,
	created_at,
	updated_at,
	created_by,
	updated_by
`

// PostgresRepo implements attachment.Repository using pgxpool against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo constructs a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func scanAttachment(row pgx.Row) (*attachment.Attachment, error) {
	var a attachment.Attachment
	err := row.Scan(
		&a.ID,
		&a.Name,
		&a.Filename,
		&a.MimeType,
		&a.FileSize,
		&a.Checksum,
		&a.StoragePath,
		&a.ResModel,
		&a.ResID,
		&a.Description,
		&a.CompanyID,
		&a.Active,
		&a.Audit.CreatedAt,
		&a.Audit.UpdatedAt,
		&a.Audit.CreatedBy,
		&a.Audit.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresRepo) Create(ctx context.Context, a *attachment.Attachment) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO ir_attachments (
			name, filename, mimetype, file_size, checksum, storage_path,
			res_model, res_id, description, company_id,
			active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''),
			NULLIF($7, ''), $8, NULLIF($9, ''), $10,
			$11, $12, $13, $14, $15
		) RETURNING id, created_at, updated_at
	`

	a.Active = true
	err := r.pool.QueryRow(ctx, query,
		a.Name, a.Filename, a.MimeType, a.FileSize, a.Checksum, a.StoragePath,
		a.ResModel, a.ResID, a.Description, a.CompanyID,
		a.Active, now, now, a.Audit.CreatedBy, a.Audit.UpdatedBy,
	).Scan(&a.ID, &a.Audit.CreatedAt, &a.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create attachment", err)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*attachment.Attachment, error) {
	query := fmt.Sprintf(`SELECT %s FROM ir_attachments WHERE id = $1 AND active = true`, selectAttachmentFields)

	a, err := scanAttachment(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("attachment with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get attachment", err)
	}

	return a, nil
}

func (r *PostgresRepo) Update(ctx context.Context, a *attachment.Attachment) error {
	now := time.Now().UTC()
	query := `
		UPDATE ir_attachments SET
			name = $1,
			filename = $2,
			mimetype = $3,
			file_size = $4,
			checksum = NULLIF($5, ''),
			storage_path = NULLIF($6, ''),
			res_model = NULLIF($7, ''),
			res_id = $8,
			description = NULLIF($9, ''),
			company_id = $10,
			updated_at = $11,
			updated_by = $12
		WHERE id = $13 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		a.Name, a.Filename, a.MimeType, a.FileSize, a.Checksum, a.StoragePath,
		a.ResModel, a.ResID, a.Description, a.CompanyID,
		now, a.Audit.UpdatedBy, a.ID,
	).Scan(&a.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("attachment with id %d not found", a.ID))
		}
		return platformerrors.Internal("failed to update attachment", err)
	}

	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE ir_attachments SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete attachment", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("attachment with id %d not found", id))
	}

	return nil
}

func (r *PostgresRepo) ListByModel(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[attachment.Attachment], error) {
	// 1. Total count
	var totalItems int64
	if resID > 0 {
		countQuery := `SELECT COUNT(*) FROM ir_attachments WHERE res_model = $1 AND res_id = $2 AND active = true`
		if err := r.pool.QueryRow(ctx, countQuery, strings.TrimSpace(resModel), resID).Scan(&totalItems); err != nil {
			return pagination.PageResult[attachment.Attachment]{}, platformerrors.Internal("failed to count attachments", err)
		}
	} else {
		countQuery := `SELECT COUNT(*) FROM ir_attachments WHERE res_model = $1 AND active = true`
		if err := r.pool.QueryRow(ctx, countQuery, strings.TrimSpace(resModel)).Scan(&totalItems); err != nil {
			return pagination.PageResult[attachment.Attachment]{}, platformerrors.Internal("failed to count attachments", err)
		}
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]attachment.Attachment{}, 0, page), nil
	}

	// 2. Data query
	var dataQuery string
	var args []any
	if resID > 0 {
		dataQuery = fmt.Sprintf(
			`SELECT %s FROM ir_attachments WHERE res_model = $1 AND res_id = $2 AND active = true ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`,
			selectAttachmentFields,
		)
		args = []any{strings.TrimSpace(resModel), resID, page.LimitClamped(), page.Offset()}
	} else {
		dataQuery = fmt.Sprintf(
			`SELECT %s FROM ir_attachments WHERE res_model = $1 AND active = true ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`,
			selectAttachmentFields,
		)
		args = []any{strings.TrimSpace(resModel), page.LimitClamped(), page.Offset()}
	}

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[attachment.Attachment]{}, platformerrors.Internal("failed to list attachments", err)
	}
	defer rows.Close()

	items := make([]attachment.Attachment, 0, page.LimitClamped())
	for rows.Next() {
		item, err := scanAttachment(rows)
		if err != nil {
			return pagination.PageResult[attachment.Attachment]{}, platformerrors.Internal("failed to scan attachment row", err)
		}
		items = append(items, *item)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[attachment.Attachment]{}, platformerrors.Internal("error iterating attachment rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}
