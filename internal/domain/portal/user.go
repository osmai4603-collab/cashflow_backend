package portal

import (
	"time"
)

type User struct {
	ID             int64      `json:"id"`
	PartnerID      int64      `json:"partner_id"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"`
	IsActive       bool       `json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CompanyID      int64      `json:"company_id"`
	InviteToken    *string    `json:"invite_token,omitempty"`
	InviteAccepted bool       `json:"invite_accepted"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
