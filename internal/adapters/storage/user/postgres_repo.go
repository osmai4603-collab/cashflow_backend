package userstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/user"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedFilterFields = map[string]string{
	"id":         "id",
	"login":      "login",
	"email":      "email",
	"name":       "name",
	"partner_id": "partner_id",
	"company_id": "company_id",
	"active":     "active",
}

const selectUserFields = `
	id,
	login,
	COALESCE(email, ''),
	email_notifications_enabled,
	name,
	password_hash,
	partner_id,
	company_id,
	active,
	is_superuser,
	last_login_at,
	created_at,
	updated_at,
	created_by,
	updated_by
`

// PostgresRepo implements user.Repository using pgxpool against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo constructs a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO res_users (
			login, email, name, password_hash, partner_id, company_id,
			active, is_superuser, email_notifications_enabled, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, NULLIF($2, ''), $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13
		) RETURNING id, created_at, updated_at
	`

	u.Active = true
	err := r.pool.QueryRow(ctx, query,
		u.Login, u.Email, u.Name, u.PasswordHash, u.PartnerID, u.CompanyID,
		u.Active, u.IsSuperuser, u.EmailNotificationsEnabled, u.Audit.CreatedAt, u.Audit.UpdatedAt, u.Audit.CreatedBy, u.Audit.UpdatedBy,
	).Scan(&u.ID, &u.Audit.CreatedAt, &u.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create user", err)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM res_users WHERE id = $1 AND active = true`, selectUserFields)

	var u user.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Login,
		&u.Email,
		&u.EmailNotificationsEnabled,
		&u.Name,
		&u.PasswordHash,
		&u.PartnerID,
		&u.CompanyID,
		&u.Active,
		&u.IsSuperuser,
		&u.LastLoginAt,
		&u.Audit.CreatedAt,
		&u.Audit.UpdatedAt,
		&u.Audit.CreatedBy,
		&u.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("user with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get user", err)
	}

	return &u, nil
}

func (r *PostgresRepo) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM res_users WHERE LOWER(login) = LOWER($1) AND active = true LIMIT 1`,
		selectUserFields,
	)

	var u user.User
	err := r.pool.QueryRow(ctx, query, strings.TrimSpace(login)).Scan(
		&u.ID,
		&u.Login,
		&u.Email,
		&u.EmailNotificationsEnabled,
		&u.Name,
		&u.PasswordHash,
		&u.PartnerID,
		&u.CompanyID,
		&u.Active,
		&u.IsSuperuser,
		&u.LastLoginAt,
		&u.Audit.CreatedAt,
		&u.Audit.UpdatedAt,
		&u.Audit.CreatedBy,
		&u.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("user not found")
		}
		return nil, platformerrors.Internal("failed to get user by login", err)
	}

	return &u, nil
}

func (r *PostgresRepo) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE res_users SET
			login = $1,
			email = NULLIF($2, ''),
			name = $3,
			password_hash = $4,
			partner_id = $5,
			company_id = $6,
			is_superuser = $7,
			email_notifications_enabled = $8,
			updated_at = $9,
			updated_by = $10
		WHERE id = $11 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		u.Login, u.Email, u.Name, u.PasswordHash, u.PartnerID, u.CompanyID,
		u.IsSuperuser, u.EmailNotificationsEnabled, u.Audit.UpdatedAt, u.Audit.UpdatedBy, u.ID,
	).Scan(&u.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("user with id %d not found", u.ID))
		}
		return platformerrors.Internal("failed to update user", err)
	}

	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE res_users SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete user", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("user with id %d not found", id))
	}

	return nil
}

func (r *PostgresRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[user.User], error) {
	var activeExplicit bool
	if f != nil {
		for _, c := range f.Criteria {
			if strings.EqualFold(c.Field, "active") {
				activeExplicit = true
				break
			}
		}
	}
	if !activeExplicit {
		if f == nil {
			f = filter.NewFilter()
		}
		f.Add("active", filter.OpEqual, true)
	}

	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedFilterFields, 1)
	if err != nil {
		return pagination.PageResult[user.User]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM res_users %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[user.User]{}, platformerrors.Internal("failed to count users", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]user.User{}, 0, page), nil
	}

	sortCol := "id"
	switch strings.ToLower(page.SortBy) {
	case "name":
		sortCol = "name"
	case "login":
		sortCol = "login"
	case "created_at":
		sortCol = "created_at"
	}
	sortDir := page.OrderDirection()

	dataQuery := fmt.Sprintf(
		"SELECT %s FROM res_users %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		selectUserFields, whereClause, sortCol, sortDir, nextIdx, nextIdx+1,
	)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[user.User]{}, platformerrors.Internal("failed to list users", err)
	}
	defer rows.Close()

	items := make([]user.User, 0, page.LimitClamped())
	for rows.Next() {
		var u user.User
		if err := rows.Scan(
			&u.ID,
			&u.Login,
			&u.Email,
			&u.EmailNotificationsEnabled,
			&u.Name,
			&u.PasswordHash,
			&u.PartnerID,
			&u.CompanyID,
			&u.Active,
			&u.IsSuperuser,
			&u.LastLoginAt,
			&u.Audit.CreatedAt,
			&u.Audit.UpdatedAt,
			&u.Audit.CreatedBy,
			&u.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[user.User]{}, platformerrors.Internal("failed to scan user row", err)
		}
		items = append(items, u)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[user.User]{}, platformerrors.Internal("error iterating user rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *PostgresRepo) UpdateLastLogin(ctx context.Context, userID int64) error {
	query := `UPDATE res_users SET last_login_at = NOW() WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return platformerrors.Internal("failed to update last login", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("user with id %d not found", userID))
	}

	return nil
}
