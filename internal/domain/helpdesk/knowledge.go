package helpdesk

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// KnowledgeCategory represents a grouping for articles (helpdesk.knowledge.category).
type KnowledgeCategory struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Sequence  int    `json:"sequence"`
	CompanyID int64  `json:"company_id"`
}

// KnowledgeArticle represents a help article (helpdesk.knowledge.article).
type KnowledgeArticle struct {
	ID           int64     `json:"id"`
	CategoryID   int64     `json:"category_id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	ContentHTML  string    `json:"content_html"`
	IsInternal   bool      `json:"is_internal"`
	ViewCount    int       `json:"view_count"`
	HelpfulCount int       `json:"helpful_count"`
	CompanyID    int64     `json:"company_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (a *KnowledgeArticle) Validate() error {
	a.Title = strings.TrimSpace(a.Title)
	if a.Title == "" {
		return platformerrors.Validation("article title is required", nil)
	}
	if a.CategoryID <= 0 {
		return platformerrors.Validation("category_id is required", nil)
	}
	if a.Slug == "" {
		a.Slug = strings.ToLower(strings.ReplaceAll(a.Title, " ", "-"))
	}
	return nil
}
