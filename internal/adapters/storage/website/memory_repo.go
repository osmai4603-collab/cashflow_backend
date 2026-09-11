package websitestorage

import (
	"context"
	"cashflow_backend/internal/domain/website"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"sync"
)

type MemoryRepo struct {
	mu    sync.RWMutex
	sites map[int64]*website.Site
	pages map[int64]*website.Page
	menus map[int64]*website.Menu
	snippets map[int64]*website.Snippet
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		sites:    make(map[int64]*website.Site),
		pages:    make(map[int64]*website.Page),
		menus:    make(map[int64]*website.Menu),
		snippets: make(map[int64]*website.Snippet),
	}
}

func (r *MemoryRepo) GetSiteByID(ctx context.Context, id int64) (*website.Site, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sites[id], nil
}

func (r *MemoryRepo) GetSiteByDomain(ctx context.Context, domain string) (*website.Site, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sites {
		if s.Domain == domain {
			return s, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) SaveSite(ctx context.Context, site *website.Site) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sites[site.ID] = site
	return nil
}

func (r *MemoryRepo) GetPageBySlug(ctx context.Context, websiteID int64, slug string) (*website.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.pages {
		if p.WebsiteID == websiteID && p.Slug == slug {
			return p, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) GetHomepage(ctx context.Context, websiteID int64) (*website.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.pages {
		if p.WebsiteID == websiteID && p.IsHomepage {
			return p, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) SavePage(ctx context.Context, page *website.Page) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pages[page.ID] = page
	return nil
}

func (r *MemoryRepo) ListPages(ctx context.Context, websiteID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[website.Page], error) {
	return pagination.PageResult[website.Page]{}, nil
}

func (r *MemoryRepo) GetMenus(ctx context.Context, websiteID int64) ([]website.Menu, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []website.Menu
	for _, m := range r.menus {
		if m.WebsiteID == websiteID {
			res = append(res, *m)
		}
	}
	return res, nil
}

func (r *MemoryRepo) SaveMenu(ctx context.Context, menu *website.Menu) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.menus[menu.ID] = menu
	return nil
}

func (r *MemoryRepo) DeleteMenu(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.menus, id)
	return nil
}

func (r *MemoryRepo) GetSnippetByKey(ctx context.Context, websiteID int64, key string) (*website.Snippet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.snippets {
		if s.WebsiteID == websiteID && s.Key == key {
			return s, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) SaveSnippet(ctx context.Context, snippet *website.Snippet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snippets[snippet.ID] = snippet
	return nil
}
