package survey

import (
	"time"
)

type Survey struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	IsScoring    bool      `json:"is_scoring"`
	PassingScore *float64  `json:"passing_score,omitempty"`
	Active       bool      `json:"active"`
	CompanyID    int64     `json:"company_id"`
	IsPublic     bool      `json:"is_public"` // Public vs private survey support
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Survey) Validate() error {
	if s.Title == "" {
		return errInvalid("survey title cannot be empty")
	}
	if s.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}
