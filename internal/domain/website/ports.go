package website

import (
	"context"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

type Repository interface {
	// Site operations
	GetSiteByID(ctx context.Context, id int64) (*Site, error)
	GetSiteByDomain(ctx context.Context, domain string) (*Site, error)
	SaveSite(ctx context.Context, site *Site) error

	// Page operations
	GetPageBySlug(ctx context.Context, websiteID int64, slug string) (*Page, error)
	GetHomepage(ctx context.Context, websiteID int64) (*Page, error)
	SavePage(ctx context.Context, page *Page) error
	ListPages(ctx context.Context, websiteID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Page], error)

	// Menu operations
	GetMenus(ctx context.Context, websiteID int64) ([]Menu, error)
	SaveMenu(ctx context.Context, menu *Menu) error
	DeleteMenu(ctx context.Context, id int64) error

	// Snippet operations
	GetSnippetByKey(ctx context.Context, websiteID int64, key string) (*Snippet, error)
	SaveSnippet(ctx context.Context, snippet *Snippet) error
}

type Service interface {
	RenderPage(ctx context.Context, domain string, slug string) (*Page, error)
	GetNavigation(ctx context.Context, domain string) ([]Menu, error)
	GetGlobalContext(ctx context.Context, domain string) (map[string]interface{}, error)
}
