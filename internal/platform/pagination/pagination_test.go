package pagination_test

import (
	"net/http/httptest"
	"testing"

	"cashflow_backend/internal/platform/pagination"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantPage  int
		wantLimit int
		wantSort  string
		wantOrder string
	}{
		{
			name:      "default values when empty",
			url:       "/api/v1/partners",
			wantPage:  1,
			wantLimit: 20,
			wantSort:  "",
			wantOrder: "asc",
		},
		{
			name:      "custom values",
			url:       "/api/v1/partners?page=3&limit=50&sort_by=name&sort_order=desc",
			wantPage:  3,
			wantLimit: 50,
			wantSort:  "name",
			wantOrder: "desc",
		},
		{
			name:      "invalid values fallback to defaults",
			url:       "/api/v1/partners?page=-5&limit=abc",
			wantPage:  1,
			wantLimit: 20,
			wantSort:  "",
			wantOrder: "asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.url, nil)
			req := pagination.Parse(r)

			if req.Page != tt.wantPage {
				t.Errorf("got Page=%d, want %d", req.Page, tt.wantPage)
			}
			if req.Limit != tt.wantLimit {
				t.Errorf("got Limit=%d, want %d", req.Limit, tt.wantLimit)
			}
			if req.SortBy != tt.wantSort {
				t.Errorf("got SortBy=%q, want %q", req.SortBy, tt.wantSort)
			}
			if req.SortOrder != tt.wantOrder {
				t.Errorf("got SortOrder=%q, want %q", req.SortOrder, tt.wantOrder)
			}
		})
	}
}

func TestPageRequest_OffsetAndClamping(t *testing.T) {
	req := pagination.PageRequest{Page: 2, Limit: 25}
	if req.Offset() != 25 {
		t.Errorf("expected offset 25, got %d", req.Offset())
	}

	maxReq := pagination.PageRequest{Page: 1, Limit: 500}
	if maxReq.LimitClamped() != pagination.MaxLimit {
		t.Errorf("expected limit clamped to %d, got %d", pagination.MaxLimit, maxReq.LimitClamped())
	}

	if req.OrderDirection() != "ASC" {
		t.Errorf("expected ASC, got %s", req.OrderDirection())
	}
	descReq := pagination.PageRequest{SortOrder: "desc"}
	if descReq.OrderDirection() != "DESC" {
		t.Errorf("expected DESC, got %s", descReq.OrderDirection())
	}
}

func TestNewPageResult(t *testing.T) {
	items := []string{"p1", "p2", "p3"}
	req := pagination.PageRequest{Page: 1, Limit: 10}
	res := pagination.NewPageResult(items, 25, req)

	if res.TotalPages != 3 {
		t.Errorf("expected 3 total pages for 25 items with limit 10, got %d", res.TotalPages)
	}
	if res.TotalItems != 25 {
		t.Errorf("expected 25 total items, got %d", res.TotalItems)
	}
	if len(res.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(res.Items))
	}
}

func TestParseCursor(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/partners?cursor=abc123xyz&limit=15", nil)
	req := pagination.ParseCursor(r)

	if req.Cursor != "abc123xyz" {
		t.Errorf("expected cursor abc123xyz, got %s", req.Cursor)
	}
	if req.Limit != 15 {
		t.Errorf("expected limit 15, got %d", req.Limit)
	}

	// Default fallback
	rEmpty := httptest.NewRequest("GET", "/api/v1/partners", nil)
	reqEmpty := pagination.ParseCursor(rEmpty)
	if reqEmpty.Cursor != "" {
		t.Errorf("expected empty cursor, got %s", reqEmpty.Cursor)
	}
	if reqEmpty.LimitClamped() != pagination.DefaultLimit {
		t.Errorf("expected default limit %d, got %d", pagination.DefaultLimit, reqEmpty.LimitClamped())
	}
}

func TestNewCursorResult(t *testing.T) {
	items := []int{1, 2, 3}
	res := pagination.NewCursorResult(items, 10, "next_token_456", true)

	if len(res.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(res.Items))
	}
	if res.NextCursor != "next_token_456" {
		t.Errorf("expected next cursor next_token_456, got %s", res.NextCursor)
	}
	if !res.HasMore {
		t.Errorf("expected HasMore to be true")
	}
	if res.Limit != 10 {
		t.Errorf("expected limit 10, got %d", res.Limit)
	}
}

