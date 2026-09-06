package partnerstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/partner"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedFilterFields = map[string]string{
	"id":          "id",
	"name":        "name",
	"email":       "email",
	"phone":       "phone",
	"mobile":      "mobile",
	"type":        "type",
	"is_customer": "is_customer",
	"is_supplier": "is_supplier",
	"vat_number":  "vat_number",
	"city":        "city",
	"state":       "state",
	"country":     "country",
	"active":      "active",
	"company_id":  "company_id",
	"parent_id":   "parent_id",
}

const selectPartnerFields = `
	id,
	name,
	COALESCE(email, ''),
	COALESCE(phone, ''),
	COALESCE(mobile, ''),
	type,
	is_customer,
	is_supplier,
	COALESCE(vat_number, ''),
	COALESCE(website, ''),
	company_id,
	parent_id,
	COALESCE(street, ''),
	COALESCE(street2, ''),
	COALESCE(city, ''),
	COALESCE(state, ''),
	COALESCE(country, ''),
	COALESCE(zip_code, ''),
	active,
	created_at,
	updated_at,
	created_by,
	updated_by
`

// PostgresRepo implements partner.Repository using pgxpool against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo constructs a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, p *partner.Partner) error {
	query := `
		INSERT INTO res_partners (
			name, email, phone, mobile, type,
			is_customer, is_supplier, vat_number, website,
			company_id, parent_id, street, street2, city, state, country, zip_code,
			active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5,
			$6, $7, NULLIF($8, ''), NULLIF($9, ''),
			$10, $11, NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''), NULLIF($17, ''),
			$18, $19, $20, $21, $22
		) RETURNING id, created_at, updated_at
	`

	p.Active = true
	err := r.pool.QueryRow(ctx, query,
		p.Name, p.Email, p.Phone, p.Mobile, string(p.Type),
		p.IsCustomer, p.IsSupplier, p.VATNumber, p.Website,
		p.CompanyID, p.ParentID, p.Street, p.Street2, p.City, p.State, p.Country, p.ZipCode,
		p.Active, p.Audit.CreatedAt, p.Audit.UpdatedAt, p.Audit.CreatedBy, p.Audit.UpdatedBy,
	).Scan(&p.ID, &p.Audit.CreatedAt, &p.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create partner", err)
	}

	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*partner.Partner, error) {
	query := fmt.Sprintf(`SELECT %s FROM res_partners WHERE id = $1 AND active = true`, selectPartnerFields)

	var p partner.Partner
	var pType string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Email,
		&p.Phone,
		&p.Mobile,
		&pType,
		&p.IsCustomer,
		&p.IsSupplier,
		&p.VATNumber,
		&p.Website,
		&p.CompanyID,
		&p.ParentID,
		&p.Street,
		&p.Street2,
		&p.City,
		&p.State,
		&p.Country,
		&p.ZipCode,
		&p.Active,
		&p.Audit.CreatedAt,
		&p.Audit.UpdatedAt,
		&p.Audit.CreatedBy,
		&p.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("partner with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get partner", err)
	}

	p.Type = partner.PartnerType(pType)
	return &p, nil
}

func (r *PostgresRepo) Update(ctx context.Context, p *partner.Partner) error {
	query := `
		UPDATE res_partners SET
			name = $1,
			email = NULLIF($2, ''),
			phone = NULLIF($3, ''),
			mobile = NULLIF($4, ''),
			type = $5,
			is_customer = $6,
			is_supplier = $7,
			vat_number = NULLIF($8, ''),
			website = NULLIF($9, ''),
			company_id = $10,
			parent_id = $11,
			street = NULLIF($12, ''),
			street2 = NULLIF($13, ''),
			city = NULLIF($14, ''),
			state = NULLIF($15, ''),
			country = NULLIF($16, ''),
			zip_code = NULLIF($17, ''),
			updated_at = $18,
			updated_by = $19
		WHERE id = $20 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		p.Name, p.Email, p.Phone, p.Mobile, string(p.Type),
		p.IsCustomer, p.IsSupplier, p.VATNumber, p.Website,
		p.CompanyID, p.ParentID, p.Street, p.Street2, p.City, p.State, p.Country, p.ZipCode,
		p.Audit.UpdatedAt, p.Audit.UpdatedBy, p.ID,
	).Scan(&p.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("partner with id %d not found", p.ID))
		}
		return platformerrors.Internal("failed to update partner", err)
	}

	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE res_partners SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete partner", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("partner with id %d not found", id))
	}

	return nil
}

func (r *PostgresRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	// If no filter or no explicit active filter, enforce active = true
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
		return pagination.PageResult[partner.Partner]{}, err
	}

	// 1. Total count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM res_partners %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[partner.Partner]{}, platformerrors.Internal("failed to count partners", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]partner.Partner{}, 0, page), nil
	}

	// 2. Data query
	sortCol := "id"
	switch strings.ToLower(page.SortBy) {
	case "name":
		sortCol = "name"
	case "created_at":
		sortCol = "created_at"
	case "type":
		sortCol = "type"
	}
	sortDir := page.OrderDirection()

	dataQuery := fmt.Sprintf(
		"SELECT %s FROM res_partners %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		selectPartnerFields, whereClause, sortCol, sortDir, nextIdx, nextIdx+1,
	)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[partner.Partner]{}, platformerrors.Internal("failed to list partners", err)
	}
	defer rows.Close()

	items := make([]partner.Partner, 0, page.LimitClamped())
	for rows.Next() {
		var p partner.Partner
		var pType string

		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Email,
			&p.Phone,
			&p.Mobile,
			&pType,
			&p.IsCustomer,
			&p.IsSupplier,
			&p.VATNumber,
			&p.Website,
			&p.CompanyID,
			&p.ParentID,
			&p.Street,
			&p.Street2,
			&p.City,
			&p.State,
			&p.Country,
			&p.ZipCode,
			&p.Active,
			&p.Audit.CreatedAt,
			&p.Audit.UpdatedAt,
			&p.Audit.CreatedBy,
			&p.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[partner.Partner]{}, platformerrors.Internal("failed to scan partner row", err)
		}

		p.Type = partner.PartnerType(pType)
		items = append(items, p)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[partner.Partner]{}, platformerrors.Internal("error iterating partner rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *PostgresRepo) ListCustomers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	f := filter.NewFilter(
		filter.Criterion{Field: "is_customer", Operator: filter.OpEqual, Value: true},
		filter.Criterion{Field: "active", Operator: filter.OpEqual, Value: true},
	)
	return r.List(ctx, f, page)
}

func (r *PostgresRepo) ListSuppliers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	f := filter.NewFilter(
		filter.Criterion{Field: "is_supplier", Operator: filter.OpEqual, Value: true},
		filter.Criterion{Field: "active", Operator: filter.OpEqual, Value: true},
	)
	return r.List(ctx, f, page)
}
