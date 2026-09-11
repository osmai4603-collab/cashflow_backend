package websiteusecase

import (
	"context"
	"cashflow_backend/internal/domain/website"
)

type UseCase struct {
	repo website.Repository
}

func New(repo website.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) RenderPage(ctx context.Context, domain string, slug string) (*website.Page, error) {
	return nil, nil
}

func (u *UseCase) GetSiteByID(ctx context.Context, id int64) (*website.Site, error) {
	return nil, nil
}

func (u *UseCase) GetNavigation(ctx context.Context, domain string) ([]website.Menu, error) {
	return nil, nil
}

func (u *UseCase) GetGlobalContext(ctx context.Context, domain string) (map[string]interface{}, error) {
	return nil, nil
}
