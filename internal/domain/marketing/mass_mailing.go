package marketing

import "time"

type MassMailingState string

const (
	MassMailingStateDraft     MassMailingState = "draft"
	MassMailingStateInQueue   MassMailingState = "in_queue"
	MassMailingStateSending   MassMailingState = "sending"
	MassMailingStateDone      MassMailingState = "done"
	MassMailingStateCancelled MassMailingState = "cancelled"
)

type MassMailing struct {
	ID            int64            `json:"id"`
	CampaignID    *int64           `json:"campaign_id,omitempty"`
	Subject       string           `json:"subject"`
	SenderName    string           `json:"sender_name"`
	SenderEmail   string           `json:"sender_email"`
	ReplyTo       string           `json:"reply_to,omitempty"`
	BodyHTML      string           `json:"body_html"`
	ScheduledDate *time.Time       `json:"scheduled_date,omitempty"`
	SentDate      *time.Time       `json:"sent_date,omitempty"`
	State         MassMailingState `json:"state"`
	CompanyID     int64            `json:"company_id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}
