package ecommercestorage

import (
	"context"
	"cashflow_backend/internal/domain/ecommerce"
	"cashflow_backend/internal/platform/pagination"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) ListCategories(ctx context.Context, websiteID int64) ([]ecommerce.CategoryView, error) {
	return nil, nil
}

func (r *PostgresRepo) ListProducts(ctx context.Context, websiteID int64, filter ecommerce.CatalogFilter, page pagination.PageRequest) (pagination.PageResult[ecommerce.ProductWebView], error) {
	return pagination.PageResult[ecommerce.ProductWebView]{}, nil
}

func (r *PostgresRepo) GetProductBySlug(ctx context.Context, websiteID int64, slug string) (*ecommerce.ProductWebView, error) {
	return nil, nil
}

func (r *PostgresRepo) GetCartBySession(ctx context.Context, websiteID int64, sessionUUID string) (*ecommerce.Cart, error) {
	return nil, nil
}

func (r *PostgresRepo) GetCartByID(ctx context.Context, id int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (r *PostgresRepo) SaveCart(ctx context.Context, cart *ecommerce.Cart) error {
	return nil
}

func (r *PostgresRepo) DeleteCart(ctx context.Context, id int64) error {
	return nil
}

func (r *PostgresRepo) CreateSaleOrderFromCart(ctx context.Context, cartID int64) (int64, string, error) {
	return 0, "", nil
}
