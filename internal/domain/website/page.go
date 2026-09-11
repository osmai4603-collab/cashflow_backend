package website

import (
	"encoding/json"
	"fmt"
	"time"
)

// Page represents a CMS page.
type Page struct {
	ID              int64           `json:"id"`
	WebsiteID       int64           `json:"website_id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	ContentJSON     json.RawMessage `json:"content_json"`
	MetaTitle       *string         `json:"meta_title,omitempty"`
	MetaDescription *string         `json:"meta_description,omitempty"`
	MetaKeywords    *string         `json:"meta_keywords,omitempty"`
	IsPublished     bool            `json:"is_published"`
	IsHomepage      bool            `json:"is_homepage"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (p *Page) Validate() error {
	if p.WebsiteID <= 0 {
		return fmt.Errorf("website_id is required")
	}
	if p.Title == "" {
		return fmt.Errorf("page title is required")
	}
	if p.Slug == "" {
		return fmt.Errorf("page slug is required")
	}
	return nil
}
