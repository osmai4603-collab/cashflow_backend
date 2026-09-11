package activity

import (
	"fmt"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MessageSubtype defines the category of an activity or message (mail.message.subtype in Odoo).
type MessageSubtype struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ResModel    string `json:"res_model,omitempty"` // If empty, it's global
	Description string `json:"description,omitempty"`
	Internal    bool   `json:"internal"` // Hidden from portal
	Default     bool   `json:"default"`
	Sequence    int    `json:"sequence"`
}

// Follower represents a user or partner following a specific record (mail.followers in Odoo).
type Follower struct {
	ID         int64   `json:"id"`
	ResModel   string  `json:"res_model"`
	ResID      int64   `json:"res_id"`
	PartnerID  *int64  `json:"partner_id,omitempty"`
	UserID     *int64  `json:"user_id,omitempty"`
	SubtypeIDs []int64 `json:"subtype_ids,omitempty"`
	CompanyID  int64   `json:"company_id"`
}

func (f *Follower) Validate() error {
	if f.ResModel == "" || f.ResID <= 0 {
		return platformerrors.Validation("res_model and res_id are required for followers", nil)
	}
	if f.UserID == nil && f.PartnerID == nil {
		return platformerrors.Validation("either user_id or partner_id must be provided", nil)
	}
	if f.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	return nil
}

// TrackingValue records a single field change (mail.tracking.value in Odoo).
type TrackingValue struct {
	ID           int64  `json:"id"`
	MessageID    int64  `json:"message_id"`
	Field        string `json:"field"`
	FieldName    string `json:"field_desc"` // Human readable name
	OldValueText string `json:"old_value_text,omitempty"`
	NewValueText string `json:"new_value_text,omitempty"`
}

// ThreadContext provides basic info about the record being tracked.
type ThreadContext struct {
	ResModel  string
	ResID     int64
	CompanyID int64
}

func (c ThreadContext) String() string {
	return fmt.Sprintf("%s(%d)", c.ResModel, c.ResID)
}

// Threadable represents any domain entity that supports Chatter / Messages and Field Tracking.
// Mirrors Odoo 19 mail.thread mixin.
type Threadable interface {
	ThreadModel() string
	ThreadID() int64
	ThreadCompanyID() int64
}

