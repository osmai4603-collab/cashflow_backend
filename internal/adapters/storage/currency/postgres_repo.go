package currencystorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/currency"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedFilterFields = map[string]string{
	"id":        "id",
	"name":      "name",
	"full_name": "full_name",
	"active":    "active",
}

const selectCurrencyFields = `
	id,
	name,
	full_name,
	symbol,
	decimal_places,
	active,
	created_at,
	updated_at
`

// PostgresRepo implements currency.Repository using pgxpool against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo constructs a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, c *currency.Currency) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO res_currencies (
			name, full_name, symbol, decimal_places, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id, created_at, updated_at
	`

	c.Active = true
	err := r.pool.QueryRow(ctx, query,
		c.Name, c.FullName, c.Symbol, c.DecimalPlaces, c.Active, now, now,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create currency", err)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*currency.Currency, error) {
	query := fmt.Sprintf(`SELECT %s FROM res_currencies WHERE id = $1 AND active = true`, selectCurrencyFields)

	var c currency.Currency
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.FullName,
		&c.Symbol,
		&c.DecimalPlaces,
		&c.Active,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("currency with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get currency", err)
	}

	return &c, nil
}

func (r *PostgresRepo) GetByName(ctx context.Context, name string) (*currency.Currency, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM res_currencies WHERE UPPER(name) = UPPER($1) AND active = true LIMIT 1`,
		selectCurrencyFields,
	)

	var c currency.Currency
	err := r.pool.QueryRow(ctx, query, strings.TrimSpace(name)).Scan(
		&c.ID,
		&c.Name,
		&c.FullName,
		&c.Symbol,
		&c.DecimalPlaces,
		&c.Active,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("currency not found")
		}
		return nil, platformerrors.Internal("failed to get currency by name", err)
	}

	return &c, nil
}

func (r *PostgresRepo) Update(ctx context.Context, c *currency.Currency) error {
	query := `
		UPDATE res_currencies SET
			name = $1,
			full_name = $2,
			symbol = $3,
			decimal_places = $4,
			updated_at = $5
		WHERE id = $6 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		c.Name, c.FullName, c.Symbol, c.DecimalPlaces, time.Now().UTC(), c.ID,
	).Scan(&c.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("currency with id %d not found", c.ID))
		}
		return platformerrors.Internal("failed to update currency", err)
	}

	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE res_currencies SET active = false, updated_at = $2 WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return platformerrors.Internal("failed to soft delete currency", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("currency with id %d not found", id))
	}

	return nil
}

func (r *PostgresRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[currency.Currency], error) {
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
		return pagination.PageResult[currency.Currency]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM res_currencies %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[currency.Currency]{}, platformerrors.Internal("failed to count currencies", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]currency.Currency{}, 0, page), nil
	}

	sortCol := "id"
	switch strings.ToLower(page.SortBy) {
	case "name":
		sortCol = "name"
	case "full_name":
		sortCol = "full_name"
	}
	sortDir := page.OrderDirection()

	dataQuery := fmt.Sprintf(
		"SELECT %s FROM res_currencies %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		selectCurrencyFields, whereClause, sortCol, sortDir, nextIdx, nextIdx+1,
	)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[currency.Currency]{}, platformerrors.Internal("failed to list currencies", err)
	}
	defer rows.Close()

	items := make([]currency.Currency, 0, page.LimitClamped())
	for rows.Next() {
		var c currency.Currency
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.FullName,
			&c.Symbol,
			&c.DecimalPlaces,
			&c.Active,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return pagination.PageResult[currency.Currency]{}, platformerrors.Internal("failed to scan currency row", err)
		}
		items = append(items, c)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[currency.Currency]{}, platformerrors.Internal("error iterating currency rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}
