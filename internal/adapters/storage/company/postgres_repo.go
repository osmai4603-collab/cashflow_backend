package companystorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/company"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedFilterFields = map[string]string{
	"id":          "id",
	"name":        "name",
	"currency_id": "currency_id",
	"active":      "active",
	"country":     "country",
	"created_at":  "created_at",
}

const selectCompanyFields = `
	id,
	name,
	partner_id,
	currency_id,
	COALESCE(phone, ''),
	COALESCE(email, ''),
	COALESCE(website, ''),
	COALESCE(vat, ''),
	COALESCE(street, ''),
	COALESCE(street2, ''),
	COALESCE(city, ''),
	COALESCE(state, ''),
	COALESCE(country, ''),
	COALESCE(zip_code, ''),
	active,
	COALESCE(attendance_kiosk_mode, 'barcode_pin'),
	COALESCE(attendance_kiosk_delay, 10),
	COALESCE(overtime_company_threshold, 0),
	COALESCE(auto_check_out_tolerance, 0),
	lc_journal_id,
	created_at,
	updated_at,
	created_by,
	updated_by
`

// PostgresRepo implements company.Repository using pgxpool against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo constructs a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, c *company.Company) error {
	query := `
		INSERT INTO res_companies (
			name, partner_id, currency_id, phone, email, website, vat,
			street, street2, city, state, country, zip_code,
			active, attendance_kiosk_mode, attendance_kiosk_delay,
			overtime_company_threshold, auto_check_out_tolerance, lc_journal_id,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''),
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''),
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		) RETURNING id, created_at, updated_at
	`

	c.Active = true
	err := r.pool.QueryRow(ctx, query,
		c.Name, c.PartnerID, c.CurrencyID, c.Phone, c.Email, c.Website, c.VAT,
		c.Street, c.Street2, c.City, c.State, c.Country, c.ZipCode,
		c.Active, c.AttendanceKioskMode, c.AttendanceKioskDelay,
		c.OvertimeCompanyThreshold, c.AutoCheckOutTolerance, c.LandedCostJournalID,
		c.Audit.CreatedAt, c.Audit.UpdatedAt, c.Audit.CreatedBy, c.Audit.UpdatedBy,
	).Scan(&c.ID, &c.Audit.CreatedAt, &c.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create company", err)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*company.Company, error) {
	query := fmt.Sprintf(`SELECT %s FROM res_companies WHERE id = $1 AND active = true`, selectCompanyFields)

	var c company.Company
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.PartnerID,
		&c.CurrencyID,
		&c.Phone,
		&c.Email,
		&c.Website,
		&c.VAT,
		&c.Street,
		&c.Street2,
		&c.City,
		&c.State,
		&c.Country,
		&c.ZipCode,
		&c.Active,
		&c.AttendanceKioskMode,
		&c.AttendanceKioskDelay,
		&c.OvertimeCompanyThreshold,
		&c.AutoCheckOutTolerance,
		&c.LandedCostJournalID,
		&c.Audit.CreatedAt,
		&c.Audit.UpdatedAt,
		&c.Audit.CreatedBy,
		&c.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("company with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get company", err)
	}

	return &c, nil
}

func (r *PostgresRepo) Update(ctx context.Context, c *company.Company) error {
	query := `
		UPDATE res_companies SET
			name = $1,
			partner_id = $2,
			currency_id = $3,
			phone = NULLIF($4, ''),
			email = NULLIF($5, ''),
			website = NULLIF($6, ''),
			vat = NULLIF($7, ''),
			street = NULLIF($8, ''),
			street2 = NULLIF($9, ''),
			city = NULLIF($10, ''),
			state = NULLIF($11, ''),
			country = NULLIF($12, ''),
			zip_code = NULLIF($13, ''),
			attendance_kiosk_mode = $14,
			attendance_kiosk_delay = $15,
			overtime_company_threshold = $16,
			auto_check_out_tolerance = $17,
			lc_journal_id = $18,
			updated_at = $19,
			updated_by = $20
		WHERE id = $21 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		c.Name, c.PartnerID, c.CurrencyID, c.Phone, c.Email, c.Website, c.VAT,
		c.Street, c.Street2, c.City, c.State, c.Country, c.ZipCode,
		c.AttendanceKioskMode, c.AttendanceKioskDelay,
		c.OvertimeCompanyThreshold, c.AutoCheckOutTolerance, c.LandedCostJournalID,
		c.Audit.UpdatedAt, c.Audit.UpdatedBy, c.ID,
	).Scan(&c.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("company with id %d not found", c.ID))
		}
		return platformerrors.Internal("failed to update company", err)
	}

	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE res_companies SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete company", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("company with id %d not found", id))
	}

	return nil
}

func (r *PostgresRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[company.Company], error) {
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
		return pagination.PageResult[company.Company]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM res_companies %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[company.Company]{}, platformerrors.Internal("failed to count companies", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]company.Company{}, 0, page), nil
	}

	sortCol := "id"
	switch strings.ToLower(page.SortBy) {
	case "name":
		sortCol = "name"
	case "created_at":
		sortCol = "created_at"
	}
	sortDir := page.OrderDirection()

	dataQuery := fmt.Sprintf(
		"SELECT %s FROM res_companies %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		selectCompanyFields, whereClause, sortCol, sortDir, nextIdx, nextIdx+1,
	)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[company.Company]{}, platformerrors.Internal("failed to list companies", err)
	}
	defer rows.Close()

	items := make([]company.Company, 0, page.LimitClamped())
	for rows.Next() {
		var c company.Company
		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.PartnerID,
			&c.CurrencyID,
			&c.Phone,
			&c.Email,
			&c.Website,
			&c.VAT,
			&c.Street,
			&c.Street2,
			&c.City,
			&c.State,
			&c.Country,
			&c.ZipCode,
			&c.Active,
			&c.AttendanceKioskMode,
			&c.AttendanceKioskDelay,
			&c.OvertimeCompanyThreshold,
			&c.AutoCheckOutTolerance,
			&c.LandedCostJournalID,
			&c.Audit.CreatedAt,
			&c.Audit.UpdatedAt,
			&c.Audit.CreatedBy,
			&c.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[company.Company]{}, platformerrors.Internal("failed to scan company row", err)
		}
		items = append(items, c)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[company.Company]{}, platformerrors.Internal("error iterating company rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *PostgresRepo) GetDefaultCompany(ctx context.Context) (*company.Company, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM res_companies WHERE id = 1 AND active = true LIMIT 1`,
		selectCompanyFields,
	)

	var c company.Company
	err := r.pool.QueryRow(ctx, query).Scan(
		&c.ID,
		&c.Name,
		&c.PartnerID,
		&c.CurrencyID,
		&c.Phone,
		&c.Email,
		&c.Website,
		&c.VAT,
		&c.Street,
		&c.Street2,
		&c.City,
		&c.State,
		&c.Country,
		&c.ZipCode,
		&c.Active,
		&c.AttendanceKioskMode,
		&c.AttendanceKioskDelay,
		&c.OvertimeCompanyThreshold,
		&c.AutoCheckOutTolerance,
		&c.Audit.CreatedAt,
		&c.Audit.UpdatedAt,
		&c.Audit.CreatedBy,
		&c.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("default company is not configured")
		}
		return nil, platformerrors.Internal("failed to get default company", err)
	}

	return &c, nil
}
