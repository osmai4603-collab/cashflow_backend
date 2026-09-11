package websiteusecase

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/website"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo website.Repository
}

func New(repo website.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) RenderPage(ctx context.Context, domain string, slug string) (*website.Page, error) {
	if domain == "" {
		return nil, platformerrors.Validation("website domain is required", map[string]string{"domain": "cannot be empty"})
	}
	site, err := u.repo.GetSiteByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("site for domain %s not found", domain))
	}
	if slug == "" {
		slug = "home"
	}
	page, err := u.repo.GetPageBySlug(ctx, site.ID, slug)
	if err != nil {
		return nil, err
	}
	if page == nil {
		page, err = u.repo.GetHomepage(ctx, site.ID)
		if err != nil {
			return nil, err
		}
	}
	if page == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("page %s not found for site %d", slug, site.ID))
	}
	return page, nil
}

func (u *UseCase) GetSiteByID(ctx context.Context, id int64) (*website.Site, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("site id is required", map[string]string{"site_id": "must be positive"})
	}
	site, err := u.repo.GetSiteByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("site with id %d not found", id))
	}
	return site, nil
}

func (u *UseCase) GetNavigation(ctx context.Context, domain string) ([]website.Menu, error) {
	if domain == "" {
		return nil, platformerrors.Validation("website domain is required", map[string]string{"domain": "cannot be empty"})
	}
	site, err := u.repo.GetSiteByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("site for domain %s not found", domain))
	}
	menus, err := u.repo.GetMenus(ctx, site.ID)
	if err != nil {
		return nil, err
	}
	return menus, nil
}

func (u *UseCase) GetGlobalContext(ctx context.Context, domain string) (map[string]interface{}, error) {
	if domain == "" {
		return nil, platformerrors.Validation("website domain is required", map[string]string{"domain": "cannot be empty"})
	}
	site, err := u.repo.GetSiteByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("site for domain %s not found", domain))
	}
	menus, err := u.repo.GetMenus(ctx, site.ID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"site":   site,
		"menus":  menus,
		"domain": domain,
	}, nil
}
