package examples

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// LocalizedProduct represents a product with its localized name and description.
type LocalizedProduct struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	SKU         string    `json:"sku"`
	Price       float64   `json:"price"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// ProductRepository executes multi-tenant, localized queries.
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new localized repository.
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetByID retrieves a product scoped to tenant company_id with a dual-fallback translation query.
func (r *ProductRepository) GetByID(ctx context.Context, companyID, productID uuid.UUID, targetLang, fallbackLang string) (*LocalizedProduct, error) {
	const query = `
		SELECT 
			p.id,
			p.company_id,
			p.sku,
			p.price,
			COALESCE(curr_t.name, fb_t.name, 'N/A') AS name,
			COALESCE(curr_t.description, fb_t.description, '') AS description
		FROM products p
		LEFT JOIN product_translations curr_t 
			ON curr_t.product_id = p.id AND curr_t.lang_code = $3
		LEFT JOIN product_translations fb_t 
			ON fb_t.product_id = p.id AND fb_t.lang_code = $4
		WHERE p.id = $1 AND p.company_id = $2;
	`

	row := r.db.QueryRowContext(ctx, query, productID, companyID, targetLang, fallbackLang)
	var p LocalizedProduct
	if err := row.Scan(&p.ID, &p.CompanyID, &p.SKU, &p.Price, &p.Name, &p.Description); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &p, nil
}
