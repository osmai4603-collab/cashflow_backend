package activity

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type State string

const (
	StateOverdue State = "overdue"
	StateToday   State = "today"
	StatePlanned State = "planned"
)

type ActivityType struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Summary     string `json:"summary,omitempty"`
	ResModel    string `json:"res_model,omitempty"`
	Category    string `json:"category"`
	DelayCount  int    `json:"delay_count"`
	DelayUnit   string `json:"delay_unit"`
	Icon        string `json:"icon,omitempty"`
	Sequence    int    `json:"sequence"`
	DefaultNote string `json:"default_note,omitempty"`
	Active      bool       `json:"active"`
	SystemType  bool       `json:"system_type"`
	CompanyID   *int64     `json:"company_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	UpdatedBy   *int64     `json:"updated_by,omitempty"`
}

func (t *ActivityType) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("activity type name is required", nil)
	}
	if len(t.Name) > 128 {
		return platformerrors.Validation("activity type name cannot exceed 128 characters", nil)
	}
	if t.DelayCount < 0 {
		return platformerrors.Validation("delay_count cannot be negative", nil)
	}
	if t.DelayUnit == "" {
		t.DelayUnit = "days"
	}
	if t.DelayUnit != "days" && t.DelayUnit != "weeks" && t.DelayUnit != "months" {
		return platformerrors.Validation("invalid activity type delay unit", nil)
	}
	if t.Category == "" {
		t.Category = "default"
	}
	if t.Sequence == 0 {
		t.Sequence = 10
	}
	return nil
}

type Activity struct {
	ID              int64      `json:"id"`
	ActivityTypeID  int64      `json:"activity_type_id"`
	Summary         string     `json:"summary"`
	Note            string     `json:"note,omitempty"`
	DateDeadline    time.Time  `json:"date_deadline"`
	AssignedUserID  int64      `json:"assigned_user_id"`
	ResModel        string     `json:"res_model,omitempty"`
	ResID           *int64      `json:"res_id,omitempty"`
	Active          bool       `json:"active"`
	DateDone        *time.Time `json:"date_done,omitempty"`
	Feedback        string     `json:"feedback,omitempty"`
	CompanyID       int64      `json:"company_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *int64     `json:"created_by,omitempty"`
	UpdatedBy       *int64     `json:"updated_by,omitempty"`
}

func (a *Activity) Validate() error {
	a.Summary = strings.TrimSpace(a.Summary)
	if a.ActivityTypeID <= 0 {
		return platformerrors.Validation("activity_type_id is required", nil)
	}
	if a.Summary == "" {
		return platformerrors.Validation("activity summary is required", nil)
	}
	if len(a.Summary) > 255 {
		return platformerrors.Validation("activity summary cannot exceed 255 characters", nil)
	}
	if a.DateDeadline.IsZero() {
		return platformerrors.Validation("date_deadline is required", nil)
	}
	if a.AssignedUserID <= 0 {
		return platformerrors.Validation("assigned_user_id is required", nil)
	}
	if a.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	if (a.ResModel == "") != (a.ResID == nil) {
		return platformerrors.Validation("res_model and res_id must be provided together", nil)
	}
	if a.ResModel != "" && a.ResID != nil && *a.ResID <= 0 {
		return platformerrors.Validation("res_id must be positive", nil)
	}
	return nil
}

// StateAt derives the display state in the user's timezone without mutating the entity.
func (a Activity) StateAt(now time.Time, location *time.Location) State {
	if location == nil {
		location = time.UTC
	}
	deadline := a.DateDeadline.In(location)
	current := now.In(location)
	today := current.YearDay()
	deadlineYear, deadlineDay := deadline.Year(), deadline.YearDay()
	currentYear := current.Year()
	if deadlineYear < currentYear || (deadlineYear == currentYear && deadlineDay < today) {
		return StateOverdue
	}
	if deadlineYear == currentYear && deadlineDay == today {
		return StateToday
	}
	return StatePlanned
}

func (a *Activity) Complete(now time.Time, feedback string) error {
	if !a.Active {
		return platformerrors.Conflict("activity is already completed or archived")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	a.Active = false
	a.DateDone = &now
	a.Feedback = strings.TrimSpace(feedback)
	return nil
}

func (a *Activity) Archive() error {
	if !a.Active {
		return platformerrors.Conflict("activity is already archived")
	}
	a.Active = false
	return nil
}

func (a *Activity) Reschedule(deadline time.Time) error {
	if !a.Active {
		return platformerrors.Conflict("cannot reschedule an inactive activity")
	}
	if deadline.IsZero() {
		return platformerrors.Validation("date_deadline is required", nil)
	}
	a.DateDeadline = deadline
	return nil
}
