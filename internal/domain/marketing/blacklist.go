package marketing

import (
	"context"
	"time"
)

type BlacklistType string

const (
	BlacklistTypeEmail BlacklistType = "email"
	BlacklistTypePhone BlacklistType = "phone"
)

type BlacklistEntry struct {
	ID        int64         `json:"id"`
	Value     string        `json:"value"` // email or phone number
	Type      BlacklistType `json:"type"`
	Reason    string        `json:"reason,omitempty"`
	CompanyID int64         `json:"company_id"`
	CreatedAt time.Time     `json:"created_at"`
}

type BlacklistRepository interface {
	Add(ctx context.Context, entry *BlacklistEntry) error
	Remove(ctx context.Context, companyID int64, value string) error
	IsBlacklisted(ctx context.Context, companyID int64, value string) (bool, error)
}
