package website

import (
	"encoding/json"
	"fmt"
	"time"
)

// Site represents a website instance in a multi-website environment.
type Site struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	Domain            string          `json:"domain"`
	CompanyID         int64           `json:"company_id"`
	DefaultLanguage   string          `json:"default_language"`
	SupportedLangs    []string        `json:"supported_langs"`
	PricelistID       int64           `json:"pricelist_id"`
	WarehouseID       int64           `json:"warehouse_id"`
	HeaderLogoURL     *string         `json:"header_logo_url,omitempty"`
	FaviconURL        *string         `json:"favicon_url,omitempty"`
	GoogleAnalyticsID *string         `json:"google_analytics_id,omitempty"`
	ThemeConfig       json.RawMessage `json:"theme_config"`
	Active            bool            `json:"active"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (s *Site) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("site name is required")
	}
	if s.Domain == "" {
		return fmt.Errorf("site domain is required")
	}
	if s.CompanyID <= 0 {
		return fmt.Errorf("valid company_id is required")
	}
	return nil
}
