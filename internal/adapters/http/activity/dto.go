package activityhttp

import (
	"time"

	"cashflow_backend/internal/domain/activity"
)

type ActivityTypeDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	ResModel    string `json:"res_model"`
	Category    string `json:"category"`
	DelayCount  int    `json:"delay_count"`
	DelayUnit   string `json:"delay_unit"`
	Icon        string `json:"icon"`
	Sequence    int    `json:"sequence"`
	DefaultNote string `json:"default_note"`
}

func ToActivityTypeDTO(t activity.ActivityType) ActivityTypeDTO {
	return ActivityTypeDTO{
		ID:          t.ID,
		Name:        t.Name,
		Summary:     t.Summary,
		ResModel:    t.ResModel,
		Category:    t.Category,
		DelayCount:  t.DelayCount,
		DelayUnit:   t.DelayUnit,
		Icon:        t.Icon,
		Sequence:    t.Sequence,
		DefaultNote: t.DefaultNote,
	}
}

type CreateActivityRequest struct {
	ActivityTypeID int64     `json:"activity_type_id" validate:"required"`
	Summary        string    `json:"summary"`
	Note           string    `json:"note"`
	DateDeadline   time.Time `json:"date_deadline" validate:"required"`
	AssignedUserID int64     `json:"assigned_user_id" validate:"required"`
	ResModel       string    `json:"res_model"`
	ResID          *int64    `json:"res_id"`
}

type ActivityDTO struct {
	ID             int64          `json:"id"`
	ActivityTypeID int64          `json:"activity_type_id"`
	Summary        string         `json:"summary"`
	Note           string         `json:"note"`
	DateDeadline   time.Time      `json:"date_deadline"`
	AssignedUserID int64          `json:"assigned_user_id"`
	ResModel       string         `json:"res_model"`
	ResID          *int64         `json:"res_id"`
	State          activity.State `json:"state"`
	Active         bool           `json:"active"`
	DateDone       *time.Time     `json:"date_done,omitempty"`
	Feedback       string         `json:"feedback,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

func ToActivityDTO(a activity.Activity, now time.Time, loc *time.Location) ActivityDTO {
	return ActivityDTO{
		ID:             a.ID,
		ActivityTypeID: a.ActivityTypeID,
		Summary:        a.Summary,
		Note:           a.Note,
		DateDeadline:   a.DateDeadline,
		AssignedUserID: a.AssignedUserID,
		ResModel:       a.ResModel,
		ResID:          a.ResID,
		State:          a.StateAt(now, loc),
		Active:         a.Active,
		DateDone:       a.DateDone,
		Feedback:       a.Feedback,
		CreatedAt:      a.CreatedAt,
	}
}

type CompleteActivityRequest struct {
	Feedback string `json:"feedback"`
}

type NotificationDTO struct {
	ID               int64                       `json:"id"`
	MessageID        *int64                      `json:"message_id,omitempty"`
	ActivityID       *int64                      `json:"activity_id,omitempty"`
	NotificationType activity.NotificationType   `json:"notification_type"`
	Status           activity.NotificationStatus `json:"status"`
	ReadAt           *time.Time                  `json:"read_at,omitempty"`
	CreatedAt        time.Time                   `json:"created_at"`
}

func ToNotificationDTO(n activity.Notification) NotificationDTO {
	return NotificationDTO{
		ID:               n.ID,
		MessageID:        n.MessageID,
		ActivityID:       n.ActivityID,
		NotificationType: n.NotificationType,
		Status:           n.Status,
		ReadAt:           n.ReadAt,
		CreatedAt:        n.CreatedAt,
	}
}
