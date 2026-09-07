package currencystorage

import (
	"context"
	"errors"
	"fmt"

	"cashflow_backend/internal/domain/currency"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const selectCurrencyRateFields = `
	id,
	currency_id,
	rate,
	date,
	company_id,
	created_at,
	updated_at,
	created_by,
	updated_by
`

// PostgresRateRepo implements currency.RateRepository using pgxpool against PostgreSQL.
type PostgresRateRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRateRepo constructs a new PostgresRateRepo.
func NewPostgresRateRepo(pool *pgxpool.Pool) *PostgresRateRepo {
	return &PostgresRateRepo{pool: pool}
}

func scanCurrencyRate(row pgx.Row) (*currency.CurrencyRate, error) {
	var rate currency.CurrencyRate
	err := row.Scan(
		&rate.ID,
		&rate.CurrencyID,
		&rate.Rate,
		&rate.Date,
		&rate.CompanyID,
		&rate.Audit.CreatedAt,
		&rate.Audit.UpdatedAt,
		&rate.Audit.CreatedBy,
		&rate.Audit.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

func (r *PostgresRateRepo) Create(ctx context.Context, rate *currency.CurrencyRate) error {
	query := `
		INSERT INTO res_currency_rates (
			currency_id, rate, date, company_id,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id, created_at, updated_at
	`

	rate.Date = fmt.Sprintf("%s", rate.Date)
	err := r.pool.QueryRow(ctx, query,
		rate.CurrencyID, rate.Rate, rate.Date, rate.CompanyID,
		rate.Audit.CreatedAt, rate.Audit.UpdatedAt, rate.Audit.CreatedBy, rate.Audit.UpdatedBy,
	).Scan(&rate.ID, &rate.Audit.CreatedAt, &rate.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create currency rate", err)
	}

	return nil
}

func (r *PostgresRateRepo) GetByID(ctx context.Context, id int64) (*currency.CurrencyRate, error) {
	query := fmt.Sprintf(`SELECT %s FROM res_currency_rates WHERE id = $1`, selectCurrencyRateFields)

	rate, err := scanCurrencyRate(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("currency rate with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get currency rate", err)
	}

	return rate, nil
}

func (r *PostgresRateRepo) GetLatestRate(ctx context.Context, currencyID int64, companyID *int64) (*currency.CurrencyRate, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM res_currency_rates
		 WHERE currency_id = $1 AND ($2::bigint IS NULL OR company_id = $2)
		 ORDER BY date DESC, id DESC LIMIT 1`,
		selectCurrencyRateFields,
	)

	rate, err := scanCurrencyRate(r.pool.QueryRow(ctx, query, currencyID, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("no currency rate found")
		}
		return nil, platformerrors.Internal("failed to get latest currency rate", err)
	}

	return rate, nil
}

func (r *PostgresRateRepo) GetRateOnDate(ctx context.Context, currencyID int64, date string, companyID *int64) (*currency.CurrencyRate, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM res_currency_rates
		 WHERE currency_id = $1 AND date <= $2::date AND ($3::bigint IS NULL OR company_id = $3)
		 ORDER BY date DESC, id DESC LIMIT 1`,
		selectCurrencyRateFields,
	)

	rate, err := scanCurrencyRate(r.pool.QueryRow(ctx, query, currencyID, date, companyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("no currency rate found on or before date")
		}
		return nil, platformerrors.Internal("failed to get currency rate on date", err)
	}

	return rate, nil
}

func (r *PostgresRateRepo) ListByCurrency(ctx context.Context, currencyID int64, page pagination.PageRequest) (pagination.PageResult[currency.CurrencyRate], error) {
	// 1. Total count
	countQuery := `SELECT COUNT(*) FROM res_currency_rates WHERE currency_id = $1`
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, currencyID).Scan(&totalItems); err != nil {
		return pagination.PageResult[currency.CurrencyRate]{}, platformerrors.Internal("failed to count currency rates", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]currency.CurrencyRate{}, 0, page), nil
	}

	// 2. Data query
	dataQuery := fmt.Sprintf(
		`SELECT %s FROM res_currency_rates WHERE currency_id = $1 ORDER BY date DESC, id DESC LIMIT $2 OFFSET $3`,
		selectCurrencyRateFields,
	)

	rows, err := r.pool.Query(ctx, dataQuery, currencyID, page.LimitClamped(), page.Offset())
	if err != nil {
		return pagination.PageResult[currency.CurrencyRate]{}, platformerrors.Internal("failed to list currency rates", err)
	}
	defer rows.Close()

	items := make([]currency.CurrencyRate, 0, page.LimitClamped())
	for rows.Next() {
		var rate currency.CurrencyRate
		if err := rows.Scan(
			&rate.ID,
			&rate.CurrencyID,
			&rate.Rate,
			&rate.Date,
			&rate.CompanyID,
			&rate.Audit.CreatedAt,
			&rate.Audit.UpdatedAt,
			&rate.Audit.CreatedBy,
			&rate.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[currency.CurrencyRate]{}, platformerrors.Internal("failed to scan currency rate row", err)
		}
		items = append(items, rate)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[currency.CurrencyRate]{}, platformerrors.Internal("error iterating currency rate rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}
