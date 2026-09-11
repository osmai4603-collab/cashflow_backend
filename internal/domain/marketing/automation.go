package marketing

import (
	"encoding/json"
	"time"
)

type AutomationTriggerType string

const (
	TriggerOnCreate AutomationTriggerType = "on_create"
	TriggerOnUpdate AutomationTriggerType = "on_update"
	TriggerCustom   AutomationTriggerType = "custom"
)

type Automation struct {
	ID          int64                 `json:"id"`
	Name        string                `json:"name"`
	TriggerType AutomationTriggerType `json:"trigger_type"`
	TargetModel string                `json:"target_model"` // e.g. "crm.lead", "sale.order"
	FilterJSON  json.RawMessage       `json:"filter_json"`
	Active      bool                  `json:"active"`
	CompanyID   int64                 `json:"company_id"`
	CreatedAt   time.Time             `json:"created_at"`
}
