package websitestorage

import (
	"context"
	"cashflow_backend/internal/domain/website"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) GetSiteByID(ctx context.Context, id int64) (*website.Site, error) {
	// Implementation
	return nil, nil
}

func (r *PostgresRepo) GetSiteByDomain(ctx context.Context, domain string) (*website.Site, error) {
	// Implementation
	return nil, nil
}

func (r *PostgresRepo) SaveSite(ctx context.Context, site *website.Site) error {
	// Implementation
	return nil
}

func (r *PostgresRepo) GetPageBySlug(ctx context.Context, websiteID int64, slug string) (*website.Page, error) {
	// Implementation
	return nil, nil
}

func (r *PostgresRepo) GetHomepage(ctx context.Context, websiteID int64) (*website.Page, error) {
	// Implementation
	return nil, nil
}

func (r *PostgresRepo) SavePage(ctx context.Context, page *website.Page) error {
	// Implementation
	return nil
}

func (r *PostgresRepo) ListPages(ctx context.Context, websiteID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[website.Page], error) {
	// Implementation
	return pagination.PageResult[website.Page]{}, nil
}

func (r *PostgresRepo) GetMenus(ctx context.Context, websiteID int64) ([]website.Menu, error) {
	// Implementation
	return nil, nil
}

func (r *PostgresRepo) SaveMenu(ctx context.Context, menu *website.Menu) error {
	// Implementation
	return nil
}

func (r *PostgresRepo) DeleteMenu(ctx context.Context, id int64) error {
	// Implementation
	return nil
}

func (r *PostgresRepo) GetSnippetByKey(ctx context.Context, websiteID int64, key string) (*website.Snippet, error) {
	// Implementation
	return nil, nil
}

func (r *PostgresRepo) SaveSnippet(ctx context.Context, snippet *website.Snippet) error {
	// Implementation
	return nil
}
