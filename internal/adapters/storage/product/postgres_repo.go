package productstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/product"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedProductFilterFields = map[string]string{
	"id":           "pt.id",
	"name":         "pt.name",
	"type":         "pt.type",
	"category_id":  "pt.category_id",
	"internal_ref": "pt.internal_ref",
	"barcode":      "pt.barcode",
	"sale_ok":      "pt.sale_ok",
	"purchase_ok":  "pt.purchase_ok",
	"active":       "pt.active",
	"sale_price":   "pt.sale_price",
	"cost_price":   "pt.cost_price",
}

const selectTemplateFields = `
	pt.id,
	pt.name,
	pt.type,
	pt.category_id,
	COALESCE(pt.internal_ref, ''),
	COALESCE(pt.barcode, ''),
	pt.sale_price,
	pt.cost_price,
	pt.uom_id,
	pt.sale_ok,
	pt.purchase_ok,
	pt.weight,
	pt.volume,
	COALESCE(pt.description, ''),
	pt.company_id,
	pt.active,
	pt.created_at,
	pt.updated_at,
	pt.created_by,
	pt.updated_by,
	c.name,
	c.complete_name,
	u.name,
	u.category,
	u.ratio,
	u.rounding
`

// PostgresRepo implements product.Repository against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Product Templates
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateTemplate(ctx context.Context, pt *product.ProductTemplate) error {
	query := `
		INSERT INTO product_templates (
			name, type, category_id, internal_ref, barcode,
			sale_price, cost_price, uom_id, sale_ok, purchase_ok,
			weight, volume, description, company_id,
			active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''),
			$6, $7, $8, $9, $10,
			$11, $12, NULLIF($13, ''), $14,
			$15, $16, $17, $18, $19
		) RETURNING id, created_at, updated_at
	`

	pt.Active = true
	err := r.pool.QueryRow(ctx, query,
		pt.Name, string(pt.Type), pt.CategoryID, pt.InternalRef, pt.Barcode,
		pt.SalePrice, pt.CostPrice, pt.UoMID, pt.SaleOK, pt.PurchaseOK,
		pt.Weight, pt.Volume, pt.Description, pt.CompanyID,
		pt.Active, pt.Audit.CreatedAt, pt.Audit.UpdatedAt, pt.Audit.CreatedBy, pt.Audit.UpdatedBy,
	).Scan(&pt.ID, &pt.Audit.CreatedAt, &pt.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create product template", err)
	}

	return nil
}

func (r *PostgresRepo) GetTemplateByID(ctx context.Context, id int64) (*product.ProductTemplate, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM product_templates pt
		LEFT JOIN product_categories c ON pt.category_id = c.id
		LEFT JOIN uom_uoms u ON pt.uom_id = u.id
		WHERE pt.id = $1 AND pt.active = true
	`, selectTemplateFields)

	var pt product.ProductTemplate
	var pType string
	var catName, catComplete sql.NullString
	var uomName, uomCat sql.NullString
	var uomRatio, uomRounding sql.NullFloat64

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pt.ID,
		&pt.Name,
		&pType,
		&pt.CategoryID,
		&pt.InternalRef,
		&pt.Barcode,
		&pt.SalePrice,
		&pt.CostPrice,
		&pt.UoMID,
		&pt.SaleOK,
		&pt.PurchaseOK,
		&pt.Weight,
		&pt.Volume,
		&pt.Description,
		&pt.CompanyID,
		&pt.Active,
		&pt.Audit.CreatedAt,
		&pt.Audit.UpdatedAt,
		&pt.Audit.CreatedBy,
		&pt.Audit.UpdatedBy,
		&catName,
		&catComplete,
		&uomName,
		&uomCat,
		&uomRatio,
		&uomRounding,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("product template with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get product template", err)
	}

	pt.Type = product.ProductType(pType)
	if pt.CategoryID != nil && catName.Valid {
		pt.Category = &product.ProductCategory{
			ID:           *pt.CategoryID,
			Name:         catName.String,
			CompleteName: catComplete.String,
			Active:       true,
		}
	}
	if pt.UoMID != nil && uomName.Valid {
		pt.UoM = &product.UnitOfMeasure{
			ID:       *pt.UoMID,
			Name:     uomName.String,
			Category: uomCat.String,
			Ratio:    uomRatio.Float64,
			Rounding: uomRounding.Float64,
			Active:   true,
		}
	}

	return &pt, nil
}

func (r *PostgresRepo) UpdateTemplate(ctx context.Context, pt *product.ProductTemplate) error {
	query := `
		UPDATE product_templates SET
			name = $1,
			type = $2,
			category_id = $3,
			internal_ref = NULLIF($4, ''),
			barcode = NULLIF($5, ''),
			sale_price = $6,
			cost_price = $7,
			uom_id = $8,
			sale_ok = $9,
			purchase_ok = $10,
			weight = $11,
			volume = $12,
			description = NULLIF($13, ''),
			company_id = $14,
			updated_at = $15,
			updated_by = $16
		WHERE id = $17 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		pt.Name, string(pt.Type), pt.CategoryID, pt.InternalRef, pt.Barcode,
		pt.SalePrice, pt.CostPrice, pt.UoMID, pt.SaleOK, pt.PurchaseOK,
		pt.Weight, pt.Volume, pt.Description, pt.CompanyID,
		pt.Audit.UpdatedAt, pt.Audit.UpdatedBy, pt.ID,
	).Scan(&pt.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("product template with id %d not found", pt.ID))
		}
		return platformerrors.Internal("failed to update product template", err)
	}

	return nil
}

func (r *PostgresRepo) DeleteTemplate(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return platformerrors.Internal("failed to begin transaction", err)
	}
	defer tx.Rollback(ctx)

	// Soft delete template
	cmdTag, err := tx.Exec(ctx, `UPDATE product_templates SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete product template", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("product template with id %d not found", id))
	}

	// Also soft delete associated variants
	_, err = tx.Exec(ctx, `UPDATE product_variants SET active = false, updated_at = NOW() WHERE template_id = $1 AND active = true`, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete product variants", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return platformerrors.Internal("failed to commit delete product transaction", err)
	}

	return nil
}

func (r *PostgresRepo) ListTemplates(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[product.ProductTemplate], error) {
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

	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedProductFilterFields, 1)
	if err != nil {
		return pagination.PageResult[product.ProductTemplate]{}, err
	}

	// 1. Total count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM product_templates pt %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[product.ProductTemplate]{}, platformerrors.Internal("failed to count products", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]product.ProductTemplate{}, 0, page), nil
	}

	// 2. Data query
	sortCol := "pt.id"
	switch strings.ToLower(page.SortBy) {
	case "name":
		sortCol = "pt.name"
	case "sale_price":
		sortCol = "pt.sale_price"
	case "created_at":
		sortCol = "pt.created_at"
	}
	sortDir := page.OrderDirection()

	dataQuery := fmt.Sprintf(`
		SELECT %s
		FROM product_templates pt
		LEFT JOIN product_categories c ON pt.category_id = c.id
		LEFT JOIN uom_uoms u ON pt.uom_id = u.id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, selectTemplateFields, whereClause, sortCol, sortDir, nextIdx, nextIdx+1)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[product.ProductTemplate]{}, platformerrors.Internal("failed to list product templates", err)
	}
	defer rows.Close()

	items := make([]product.ProductTemplate, 0, page.LimitClamped())
	for rows.Next() {
		var pt product.ProductTemplate
		var pType string
		var catName, catComplete sql.NullString
		var uomName, uomCat sql.NullString
		var uomRatio, uomRounding sql.NullFloat64

		if err := rows.Scan(
			&pt.ID,
			&pt.Name,
			&pType,
			&pt.CategoryID,
			&pt.InternalRef,
			&pt.Barcode,
			&pt.SalePrice,
			&pt.CostPrice,
			&pt.UoMID,
			&pt.SaleOK,
			&pt.PurchaseOK,
			&pt.Weight,
			&pt.Volume,
			&pt.Description,
			&pt.CompanyID,
			&pt.Active,
			&pt.Audit.CreatedAt,
			&pt.Audit.UpdatedAt,
			&pt.Audit.CreatedBy,
			&pt.Audit.UpdatedBy,
			&catName,
			&catComplete,
			&uomName,
			&uomCat,
			&uomRatio,
			&uomRounding,
		); err != nil {
			return pagination.PageResult[product.ProductTemplate]{}, platformerrors.Internal("failed to scan product template row", err)
		}

		pt.Type = product.ProductType(pType)
		if pt.CategoryID != nil && catName.Valid {
			pt.Category = &product.ProductCategory{
				ID:           *pt.CategoryID,
				Name:         catName.String,
				CompleteName: catComplete.String,
				Active:       true,
			}
		}
		if pt.UoMID != nil && uomName.Valid {
			pt.UoM = &product.UnitOfMeasure{
				ID:       *pt.UoMID,
				Name:     uomName.String,
				Category: uomCat.String,
				Ratio:    uomRatio.Float64,
				Rounding: uomRounding.Float64,
				Active:   true,
			}
		}

		items = append(items, pt)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[product.ProductTemplate]{}, platformerrors.Internal("error iterating product template rows", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Product Variants
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateVariant(ctx context.Context, pv *product.ProductVariant) error {
	query := `
		INSERT INTO product_variants (
			template_id, sku, barcode, extra_price,
			active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, NULLIF($2, ''), NULLIF($3, ''), $4,
			$5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at
	`

	pv.Active = true
	err := r.pool.QueryRow(ctx, query,
		pv.TemplateID, pv.SKU, pv.Barcode, pv.ExtraPrice,
		pv.Active, pv.Audit.CreatedAt, pv.Audit.UpdatedAt, pv.Audit.CreatedBy, pv.Audit.UpdatedBy,
	).Scan(&pv.ID, &pv.Audit.CreatedAt, &pv.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create product variant", err)
	}

	return nil
}

func (r *PostgresRepo) GetVariantByID(ctx context.Context, id int64) (*product.ProductVariant, error) {
	query := `
		SELECT id, template_id, COALESCE(sku, ''), COALESCE(barcode, ''), extra_price,
		       active, created_at, updated_at, created_by, updated_by
		FROM product_variants
		WHERE id = $1 AND active = true
	`

	var pv product.ProductVariant
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pv.ID,
		&pv.TemplateID,
		&pv.SKU,
		&pv.Barcode,
		&pv.ExtraPrice,
		&pv.Active,
		&pv.Audit.CreatedAt,
		&pv.Audit.UpdatedAt,
		&pv.Audit.CreatedBy,
		&pv.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("product variant with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get product variant", err)
	}

	return &pv, nil
}

func (r *PostgresRepo) GetVariantsByTemplateID(ctx context.Context, templateID int64) ([]product.ProductVariant, error) {
	query := `
		SELECT id, template_id, COALESCE(sku, ''), COALESCE(barcode, ''), extra_price,
		       active, created_at, updated_at, created_by, updated_by
		FROM product_variants
		WHERE template_id = $1 AND active = true
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, query, templateID)
	if err != nil {
		return nil, platformerrors.Internal("failed to query product variants", err)
	}
	defer rows.Close()

	var variants []product.ProductVariant
	for rows.Next() {
		var pv product.ProductVariant
		if err := rows.Scan(
			&pv.ID,
			&pv.TemplateID,
			&pv.SKU,
			&pv.Barcode,
			&pv.ExtraPrice,
			&pv.Active,
			&pv.Audit.CreatedAt,
			&pv.Audit.UpdatedAt,
			&pv.Audit.CreatedBy,
			&pv.Audit.UpdatedBy,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan product variant", err)
		}
		variants = append(variants, pv)
	}

	return variants, nil
}

func (r *PostgresRepo) UpdateVariant(ctx context.Context, pv *product.ProductVariant) error {
	query := `
		UPDATE product_variants SET
			sku = NULLIF($1, ''),
			barcode = NULLIF($2, ''),
			extra_price = $3,
			updated_at = $4,
			updated_by = $5
		WHERE id = $6 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		pv.SKU, pv.Barcode, pv.ExtraPrice, pv.Audit.UpdatedAt, pv.Audit.UpdatedBy, pv.ID,
	).Scan(&pv.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("product variant with id %d not found", pv.ID))
		}
		return platformerrors.Internal("failed to update product variant", err)
	}

	return nil
}

func (r *PostgresRepo) DeleteVariant(ctx context.Context, id int64) error {
	query := `UPDATE product_variants SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete product variant", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("product variant with id %d not found", id))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Product Categories
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateCategory(ctx context.Context, c *product.ProductCategory) error {
	query := `
		INSERT INTO product_categories (name, parent_id, complete_name, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	c.Active = true
	err := r.pool.QueryRow(ctx, query,
		c.Name, c.ParentID, c.CompleteName, c.Active, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create product category", err)
	}

	return nil
}

func (r *PostgresRepo) GetCategoryByID(ctx context.Context, id int64) (*product.ProductCategory, error) {
	query := `
		SELECT id, name, parent_id, complete_name, active, created_at, updated_at
		FROM product_categories
		WHERE id = $1 AND active = true
	`

	var c product.ProductCategory
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.ParentID, &c.CompleteName, &c.Active, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("product category with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get product category", err)
	}

	return &c, nil
}

func (r *PostgresRepo) UpdateCategory(ctx context.Context, c *product.ProductCategory) error {
	query := `
		UPDATE product_categories SET
			name = $1,
			parent_id = $2,
			complete_name = $3,
			updated_at = $4
		WHERE id = $5 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		c.Name, c.ParentID, c.CompleteName, c.UpdatedAt, c.ID,
	).Scan(&c.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("product category with id %d not found", c.ID))
		}
		return platformerrors.Internal("failed to update product category", err)
	}

	return nil
}

func (r *PostgresRepo) DeleteCategory(ctx context.Context, id int64) error {
	// Check if active products reference this category
	var count int64
	checkQuery := `SELECT COUNT(*) FROM product_templates WHERE category_id = $1 AND active = true`
	if err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&count); err != nil {
		return platformerrors.Internal("failed to check category references", err)
	}
	if count > 0 {
		return platformerrors.Conflict("cannot delete category referenced by active products")
	}

	query := `UPDATE product_categories SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete product category", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("product category with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListCategories(ctx context.Context) ([]product.ProductCategory, error) {
	query := `
		SELECT id, name, parent_id, complete_name, active, created_at, updated_at
		FROM product_categories
		WHERE active = true
		ORDER BY complete_name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list product categories", err)
	}
	defer rows.Close()

	var categories []product.ProductCategory
	for rows.Next() {
		var c product.ProductCategory
		if err := rows.Scan(
			&c.ID, &c.Name, &c.ParentID, &c.CompleteName, &c.Active, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan product category", err)
		}
		categories = append(categories, c)
	}

	return categories, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Units of Measure (UoM)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateUoM(ctx context.Context, u *product.UnitOfMeasure) error {
	query := `
		INSERT INTO uom_uoms (name, category, ratio, rounding, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	u.Active = true
	err := r.pool.QueryRow(ctx, query,
		u.Name, u.Category, u.Ratio, u.Rounding, u.Active, u.CreatedAt, u.UpdatedAt,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create unit of measure", err)
	}

	return nil
}

func (r *PostgresRepo) GetUoMByID(ctx context.Context, id int64) (*product.UnitOfMeasure, error) {
	query := `
		SELECT id, name, category, ratio, rounding, active, created_at, updated_at
		FROM uom_uoms
		WHERE id = $1 AND active = true
	`

	var u product.UnitOfMeasure
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Name, &u.Category, &u.Ratio, &u.Rounding, &u.Active, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("unit of measure with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get unit of measure", err)
	}

	return &u, nil
}

func (r *PostgresRepo) UpdateUoM(ctx context.Context, u *product.UnitOfMeasure) error {
	query := `
		UPDATE uom_uoms SET
			name = $1,
			category = $2,
			ratio = $3,
			rounding = $4,
			updated_at = $5
		WHERE id = $6 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		u.Name, u.Category, u.Ratio, u.Rounding, u.UpdatedAt, u.ID,
	).Scan(&u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("unit of measure with id %d not found", u.ID))
		}
		return platformerrors.Internal("failed to update unit of measure", err)
	}

	return nil
}

func (r *PostgresRepo) DeleteUoM(ctx context.Context, id int64) error {
	var count int64
	checkQuery := `SELECT COUNT(*) FROM product_templates WHERE uom_id = $1 AND active = true`
	if err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&count); err != nil {
		return platformerrors.Internal("failed to check uom references", err)
	}
	if count > 0 {
		return platformerrors.Conflict("cannot delete unit of measure referenced by active products")
	}

	query := `UPDATE uom_uoms SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete unit of measure", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("unit of measure with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListUoMs(ctx context.Context) ([]product.UnitOfMeasure, error) {
	query := `
		SELECT id, name, category, ratio, rounding, active, created_at, updated_at
		FROM uom_uoms
		WHERE active = true
		ORDER BY category ASC, ratio ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list units of measure", err)
	}
	defer rows.Close()

	var uoms []product.UnitOfMeasure
	for rows.Next() {
		var u product.UnitOfMeasure
		if err := rows.Scan(
			&u.ID, &u.Name, &u.Category, &u.Ratio, &u.Rounding, &u.Active, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan unit of measure", err)
		}
		uoms = append(uoms, u)
	}

	return uoms, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pricelists
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreatePricelist(ctx context.Context, pl *product.Pricelist) error {
	query := `
		INSERT INTO product_pricelists (name, currency, active, created_at, updated_at, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	pl.Active = true
	err := r.pool.QueryRow(ctx, query,
		pl.Name, pl.Currency, pl.Active, pl.Audit.CreatedAt, pl.Audit.UpdatedAt, pl.Audit.CreatedBy, pl.Audit.UpdatedBy,
	).Scan(&pl.ID, &pl.Audit.CreatedAt, &pl.Audit.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create pricelist", err)
	}

	return nil
}

func (r *PostgresRepo) GetPricelistByID(ctx context.Context, id int64) (*product.Pricelist, error) {
	query := `
		SELECT id, name, currency, active, created_at, updated_at, created_by, updated_by
		FROM product_pricelists
		WHERE id = $1 AND active = true
	`

	var pl product.Pricelist
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pl.ID, &pl.Name, &pl.Currency, &pl.Active,
		&pl.Audit.CreatedAt, &pl.Audit.UpdatedAt, &pl.Audit.CreatedBy, &pl.Audit.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("pricelist with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get pricelist", err)
	}

	items, err := r.GetPricelistItems(ctx, id)
	if err != nil {
		return nil, err
	}
	pl.Items = items

	return &pl, nil
}

func (r *PostgresRepo) UpdatePricelist(ctx context.Context, pl *product.Pricelist) error {
	query := `
		UPDATE product_pricelists SET
			name = $1,
			currency = $2,
			updated_at = $3,
			updated_by = $4
		WHERE id = $5 AND active = true
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		pl.Name, pl.Currency, pl.Audit.UpdatedAt, pl.Audit.UpdatedBy, pl.ID,
	).Scan(&pl.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("pricelist with id %d not found", pl.ID))
		}
		return platformerrors.Internal("failed to update pricelist", err)
	}

	return nil
}

func (r *PostgresRepo) DeletePricelist(ctx context.Context, id int64) error {
	query := `UPDATE product_pricelists SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft delete pricelist", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("pricelist with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListPricelists(ctx context.Context) ([]product.Pricelist, error) {
	query := `
		SELECT id, name, currency, active, created_at, updated_at, created_by, updated_by
		FROM product_pricelists
		WHERE active = true
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list pricelists", err)
	}
	defer rows.Close()

	var pricelists []product.Pricelist
	for rows.Next() {
		var pl product.Pricelist
		if err := rows.Scan(
			&pl.ID, &pl.Name, &pl.Currency, &pl.Active,
			&pl.Audit.CreatedAt, &pl.Audit.UpdatedAt, &pl.Audit.CreatedBy, &pl.Audit.UpdatedBy,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan pricelist", err)
		}
		pricelists = append(pricelists, pl)
	}

	return pricelists, nil
}

func (r *PostgresRepo) AddPricelistItem(ctx context.Context, item *product.PricelistItem) error {
	query := `
		INSERT INTO product_pricelist_items (
			pricelist_id, applied_on, category_id, template_id, variant_id,
			min_quantity, compute_price, fixed_price, percent_price,
			date_start, date_end, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13
		) RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		item.PricelistID, string(item.AppliedOn), item.CategoryID, item.TemplateID, item.VariantID,
		item.MinQuantity, string(item.ComputePrice), item.FixedPrice, item.PercentPrice,
		item.DateStart, item.DateEnd, item.CreatedAt, item.UpdatedAt,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to add pricelist item", err)
	}

	return nil
}

func (r *PostgresRepo) DeletePricelistItem(ctx context.Context, id int64) error {
	query := `DELETE FROM product_pricelist_items WHERE id = $1`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete pricelist item", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("pricelist item with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) GetPricelistItems(ctx context.Context, pricelistID int64) ([]product.PricelistItem, error) {
	query := `
		SELECT id, pricelist_id, applied_on, category_id, template_id, variant_id,
		       min_quantity, compute_price, fixed_price, percent_price,
		       date_start, date_end, created_at, updated_at
		FROM product_pricelist_items
		WHERE pricelist_id = $1
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, query, pricelistID)
	if err != nil {
		return nil, platformerrors.Internal("failed to get pricelist items", err)
	}
	defer rows.Close()

	var items []product.PricelistItem
	for rows.Next() {
		var item product.PricelistItem
		var applied, compute string
		if err := rows.Scan(
			&item.ID, &item.PricelistID, &applied, &item.CategoryID, &item.TemplateID, &item.VariantID,
			&item.MinQuantity, &compute, &item.FixedPrice, &item.PercentPrice,
			&item.DateStart, &item.DateEnd, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan pricelist item", err)
		}
		item.AppliedOn = product.PricelistAppliedOn(applied)
		item.ComputePrice = product.PricelistComputeType(compute)
		items = append(items, item)
	}

	return items, nil
}
