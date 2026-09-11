package livechat

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// CannedResponse represents a predefined shortcut for quick replies (im_livechat.canned_response).
type CannedResponse struct {
	ID           int64  `json:"id"`
	Source       string `json:"source"` // The shortcut text, e.g., ":hello"
	Substitution string `json:"substitution"` // The full text
	CompanyID    int64  `json:"company_id"`
}

func (c *CannedResponse) Validate() error {
	c.Source = strings.TrimSpace(c.Source)
	if c.Source == "" {
		return platformerrors.Validation("source shortcut is required", nil)
	}
	if !strings.HasPrefix(c.Source, ":") {
		c.Source = ":" + c.Source
	}
	c.Substitution = strings.TrimSpace(c.Substitution)
	if c.Substitution == "" {
		return platformerrors.Validation("substitution text is required", nil)
	}
	return nil
}
