package marketing

import "time"

type MailingTrace struct {
	ID           int64      `json:"id"`
	MailingID    int64      `json:"mailing_id"`
	ContactID    int64      `json:"contact_id"`
	Email        string     `json:"email"`
	SentAt       time.Time  `json:"sent_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	OpenedAt     *time.Time `json:"opened_at,omitempty"`
	ClickedAt    *time.Time `json:"clicked_at,omitempty"`
	BouncedAt    *time.Time `json:"bounced_at,omitempty"`
	BounceReason string     `json:"bounce_reason,omitempty"`
	TrackingCode string     `json:"tracking_code"`
}

type ClickEvent struct {
	TraceID   int64     `json:"trace_id"`
	URL       string    `json:"url"`
	ClickedAt time.Time `json:"clicked_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}
