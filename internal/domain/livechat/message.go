package livechat

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type SenderType string

const (
	SenderTypeVisitor  SenderType = "visitor"
	SenderTypeOperator SenderType = "operator"
	SenderTypeSystem   SenderType = "system"
)

// Message represents a single chat message (mail.message in Odoo livechat).
type Message struct {
	ID         int64      `json:"id"`
	SessionID  int64      `json:"session_id"`
	SenderType SenderType `json:"sender_type"`
	SenderID   *int64     `json:"sender_id,omitempty"` // UserID if operator, nil if visitor
	Body       string     `json:"body"`
	FileURL    string     `json:"file_url,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Validate ensures the message is valid.
func (m *Message) Validate() error {
	if m.SessionID <= 0 {
		return platformerrors.Validation("session_id is required", nil)
	}
	if m.Body == "" && m.FileURL == "" {
		return platformerrors.Validation("message body or file is required", nil)
	}
	if m.SenderType == "" {
		return platformerrors.Validation("sender_type is required", nil)
	}
	return nil
}
