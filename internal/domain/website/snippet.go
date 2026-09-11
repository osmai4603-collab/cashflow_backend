package website

import "encoding/json"

// Snippet represents a reusable content block or UI component.
type Snippet struct {
	ID        int64           `json:"id"`
	WebsiteID int64           `json:"website_id"`
	Key       string          `json:"key"` // e.g., "footer_contact", "home_banner"
	Content   json.RawMessage `json:"content"`
	Active    bool            `json:"active"`
}
