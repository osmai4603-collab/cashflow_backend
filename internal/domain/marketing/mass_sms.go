package marketing

import "time"

type MassSMSState string

const (
	MassSMSStateDraft     MassSMSState = "draft"
	MassSMSStateInQueue   MassSMSState = "in_queue"
	MassSMSStateSending   MassSMSState = "sending"
	MassSMSStateDone      MassSMSState = "done"
	MassSMSStateCancelled MassSMSState = "cancelled"
)

type MassSMS struct {
	ID            int64        `json:"id"`
	CampaignID    *int64       `json:"campaign_id,omitempty"`
	Body          string       `json:"body"`
	ScheduledDate *time.Time   `json:"scheduled_date,omitempty"`
	SentDate      *time.Time   `json:"sent_date,omitempty"`
	State         MassSMSState `json:"state"`
	CompanyID     int64        `json:"company_id"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}
