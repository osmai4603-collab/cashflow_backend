package websiteusecase

import (
	"context"
	"testing"
	"time"

	websitestorage "cashflow_backend/internal/adapters/storage/website"
	"cashflow_backend/internal/domain/website"
)

func TestWebsiteUseCase_RenderPageAndNavigation(t *testing.T) {
	repo := websitestorage.NewMemoryRepo()
	ctx := context.Background()
	now := time.Now().UTC()

	site := &website.Site{
		ID:              10,
		Name:            "Main Store",
		Domain:          "store.example.com",
		CompanyID:       1,
		DefaultLanguage: "en",
		SupportedLangs:  []string{"en", "ar"},
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.SaveSite(ctx, site); err != nil {
		t.Fatalf("save site: %v", err)
	}

	page := &website.Page{
		ID:          11,
		WebsiteID:   10,
		Title:       "Home",
		Slug:        "home",
		ContentJSON: []byte(`{"blocks":[]}`),
		IsPublished: true,
		IsHomepage:  true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.SavePage(ctx, page); err != nil {
		t.Fatalf("save page: %v", err)
	}

	menu := &website.Menu{ID: 12, WebsiteID: 10, Name: "Home", URL: "/", Sequence: 1}
	if err := repo.SaveMenu(ctx, menu); err != nil {
		t.Fatalf("save menu: %v", err)
	}

	uc := New(repo)
	got, err := uc.RenderPage(ctx, "store.example.com", "home")
	if err != nil {
		t.Fatalf("render page err: %v", err)
	}
	if got == nil || got.WebsiteID != 10 || got.Slug != "home" {
		t.Fatalf("unexpected page: %#v", got)
	}

	nav, err := uc.GetNavigation(ctx, "store.example.com")
	if err != nil {
		t.Fatalf("get navigation err: %v", err)
	}
	if len(nav) != 1 || nav[0].Name != "Home" {
		t.Fatalf("unexpected navigation: %#v", nav)
	}

	ctxMap, err := uc.GetGlobalContext(ctx, "store.example.com")
	if err != nil {
		t.Fatalf("get global context err: %v", err)
	}
	if ctxMap["site"] == nil || ctxMap["menus"] == nil {
		t.Fatalf("global context incomplete: %#v", ctxMap)
	}
}
