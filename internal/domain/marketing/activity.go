package marketing

import "time"

type ActivityActionType string

const (
	ActionSendEmail ActivityActionType = "send_email"
	ActionSendSMS   ActivityActionType = "send_sms"
	ActionWait      ActivityActionType = "wait"
)

type ActivityConditionType string

const (
	ConditionNone    ActivityConditionType = "none"
	ConditionOpened  ActivityConditionType = "if_opened"
	ConditionClicked ActivityConditionType = "if_clicked"
)

type AutomationActivity struct {
	ID            int64                 `json:"id"`
	AutomationID  int64                 `json:"automation_id"`
	ParentID      *int64                `json:"parent_id,omitempty"`
	ActionType    ActivityActionType    `json:"action_type"`
	DelayHours    int                   `json:"delay_hours"`
	ConditionType ActivityConditionType `json:"condition_type"`
	TemplateID    *int64                `json:"template_id,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
}
