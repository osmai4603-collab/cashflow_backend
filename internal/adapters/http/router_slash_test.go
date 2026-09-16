package httpadapter_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "cashflow_backend/internal/adapters/http"
	partnerhttp "cashflow_backend/internal/adapters/http/partner"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	partnerusecase "cashflow_backend/internal/usecase/partner"
)

func setupSlashRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	baseHandler := httpadapter.NewBaseHandler("odoo_go_backend", "0.1.0", logger)
	health := &mockHealthRoutes{}
	repo := partnerstorage.NewMemoryRepo()
	uc := partnerusecase.New(repo, logger)
	partnerHandler := partnerhttp.NewHandler(uc, logger)
	return httpadapter.NewRouter(baseHandler, health, partnerHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, logger)
}

func TestRouter_TrailingSlashServedByCanonicalRoute(t *testing.T) {
	router := setupSlashRouter()

	tests := []struct {
		name         string
		path         string
		wantCode     int
		wantBodyPart string
	}{
		{
			name:         "registered trailing slash root stays 200",
			path:         "/api/v1/",
			wantCode:     http.StatusOK,
			wantBodyPart: "ERP API v1 is online",
		},
		{
			name:         "no trailing slash keeps working",
			path:         "/api/v1/partners",
			wantCode:     http.StatusOK,
			wantBodyPart: `"success":true`,
		},
		{
			name:         "base collection with trailing slash",
			path:         "/api/v1/partners/",
			wantCode:     http.StatusOK,
			wantBodyPart: `"success":true`,
		},
		{
			name:         "nested collection with trailing slash",
			path:         "/api/v1/partners/suppliers/",
			wantCode:     http.StatusOK,
			wantBodyPart: `"success":true`,
		},
		{
			name:         "query string preserved",
			path:         "/api/v1/partners/suppliers/?page=2&limit=10",
			wantCode:     http.StatusOK,
			wantBodyPart: `"success":true`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := withAuth(httptest.NewRequest(http.MethodGet, tt.path, nil), "user")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("path %s: expected status %d, got %d", tt.path, tt.wantCode, rr.Code)
			}
			if !strings.Contains(rr.Body.String(), tt.wantBodyPart) {
				t.Errorf("path %s: expected body to contain %q, got:\n%s", tt.path, tt.wantBodyPart, rr.Body.String())
			}
		})
	}
}

func TestRouter_TrailingSlashRequiresAuthLikeCanonical(t *testing.T) {
	// Auth middleware lives on the /api/v1 subtree; a trailing-slash variant
	// must not bypass it.
	router := setupSlashRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/partners/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token on trailing-slash path, got %d", rr.Code)
	}
}

func TestRouter_TrailingSlashUnknownPath404(t *testing.T) {
	router := setupSlashRouter()

	req := withAuth(httptest.NewRequest(http.MethodGet, "/api/v1/definitely-not-registered/", nil), "user")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown trailing-slash path, got %d", rr.Code)
	}
}