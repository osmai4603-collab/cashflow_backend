package livechat

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Channel represents a live chat widget configuration (im_livechat.channel in Odoo).
type Channel struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	WelcomeMsg  string    `json:"welcome_msg"`
	ButtonText  string    `json:"button_text"`
	HeaderColor string    `json:"header_color"`
	CompanyID   int64     `json:"company_id"`
	Active      bool      `json:"active"`
	OperatorIDs []int64   `json:"operator_ids"`
	CreatedAt   time.Time `json:"created_at"`
}

// Validate ensures the channel configuration is valid.
func (c *Channel) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return platformerrors.Validation("channel name is required", nil)
	}
	if c.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	if c.HeaderColor == "" {
		c.HeaderColor = "#1E3A8A"
	}
	return nil
}

// IsOperator checks if a user is assigned to this channel.
func (c *Channel) IsOperator(userID int64) bool {
	for _, id := range c.OperatorIDs {
		if id == userID {
			return true
		}
	}
	return false
}
