package pagination

import (
	"net/http"
	"strconv"
	"strings"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// PageRequest encapsulates standard pagination parameters from client requests.
type PageRequest struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"` // "asc" or "desc"
}

// Offset returns the zero-indexed offset for SQL OFFSET clause.
func (p PageRequest) Offset() int {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * p.LimitClamped()
}

// LimitClamped returns the limit guaranteed between 1 and MaxLimit.
func (p PageRequest) LimitClamped() int {
	if p.Limit <= 0 {
		return DefaultLimit
	}
	if p.Limit > MaxLimit {
		return MaxLimit
	}
	return p.Limit
}

// OrderDirection returns sanitized "ASC" or "DESC".
func (p PageRequest) OrderDirection() string {
	if strings.ToLower(p.SortOrder) == "desc" {
		return "DESC"
	}
	return "ASC"
}

// Parse extracts pagination parameters from HTTP query string.
func Parse(r *http.Request) PageRequest {
	req := PageRequest{
		Page:      DefaultPage,
		Limit:     DefaultLimit,
		SortOrder: "asc",
	}

	q := r.URL.Query()

	if pageStr := q.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}

	if sortBy := q.Get("sort_by"); sortBy != "" {
		req.SortBy = strings.TrimSpace(sortBy)
	}

	if sortOrder := q.Get("sort_order"); sortOrder != "" {
		req.SortOrder = strings.ToLower(strings.TrimSpace(sortOrder))
	}

	return req
}

// PageResult is a generic wrapper holding paginated items and metadata.
type PageResult[T any] struct {
	Items      []T   `json:"items"`
	TotalItems int64 `json:"total_items"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// NewPageResult creates a new PageResult with calculated metadata.
func NewPageResult[T any](items []T, totalItems int64, req PageRequest) PageResult[T] {
	limit := req.LimitClamped()
	page := req.Page
	if page < 1 {
		page = 1
	}

	totalPages := 0
	if totalItems > 0 && limit > 0 {
		totalPages = int((totalItems + int64(limit) - 1) / int64(limit))
	}

	if items == nil {
		items = make([]T, 0)
	}

	return PageResult[T]{
		Items:      items,
		TotalItems: totalItems,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

// CursorRequest encapsulates cursor-based pagination parameters.
type CursorRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

// LimitClamped returns the limit guaranteed between 1 and MaxLimit.
func (c CursorRequest) LimitClamped() int {
	if c.Limit <= 0 {
		return DefaultLimit
	}
	if c.Limit > MaxLimit {
		return MaxLimit
	}
	return c.Limit
}

// ParseCursor extracts cursor pagination parameters from HTTP request.
func ParseCursor(r *http.Request) CursorRequest {
	req := CursorRequest{
		Limit: DefaultLimit,
	}

	q := r.URL.Query()
	if cursor := q.Get("cursor"); cursor != "" {
		req.Cursor = strings.TrimSpace(cursor)
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}

	return req
}

// CursorResult is a generic wrapper holding cursor-paginated items and pagination state.
type CursorResult[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
	Limit      int    `json:"limit"`
}

// NewCursorResult builds a CursorResult.
func NewCursorResult[T any](items []T, limit int, nextCursor string, hasMore bool) CursorResult[T] {
	if items == nil {
		items = make([]T, 0)
	}
	return CursorResult[T]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Limit:      limit,
	}
}

