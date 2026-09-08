package activity

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type MessageType string

const (
	MessageTypeNotification MessageType = "notification"
	MessageTypeComment      MessageType = "comment"
	MessageTypeEmail        MessageType = "email"
)

type Message struct {
	ID             int64           `json:"id"`
	Subject        string          `json:"subject,omitempty"`
	Body           string          `json:"body"`
	MessageType    MessageType     `json:"message_type"`
	ResModel       string          `json:"res_model,omitempty"`
	ResID          *int64          `json:"res_id,omitempty"`
	SubtypeID      *int64          `json:"subtype_id,omitempty"`
	ParentID       *int64          `json:"parent_id,omitempty"`
	AuthorID       *int64          `json:"author_id,omitempty"`
	ActivityID     *int64          `json:"activity_id,omitempty"`
	TrackingValues []TrackingValue `json:"tracking_values,omitempty"`
	CompanyID      int64           `json:"company_id"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (m *Message) Validate() error {
	m.Body = strings.TrimSpace(m.Body)
	if m.Body == "" {
		return platformerrors.Validation("message body is required", nil)
	}
	if m.MessageType == "" {
		m.MessageType = MessageTypeNotification
	}
	if m.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	if (m.ResModel == "") != (m.ResID == nil) {
		return platformerrors.Validation("res_model and res_id must be provided together", nil)
	}
	return nil
}

type NotificationType string

const (
	NotificationTypeInbox NotificationType = "inbox"
	NotificationTypeEmail NotificationType = "email"
)

type NotificationStatus string

const (
	NotificationStatusUnread     NotificationStatus = "unread"
	NotificationStatusRead       NotificationStatus = "read"
	NotificationStatusQueued     NotificationStatus = "queued"
	NotificationStatusSent       NotificationStatus = "sent"
	NotificationStatusFailed     NotificationStatus = "failed"
	NotificationStatusDeadLetter NotificationStatus = "dead_letter"
)

type Notification struct {
	ID              int64              `json:"id"`
	MessageID       *int64             `json:"message_id,omitempty"`
	ActivityID      *int64             `json:"activity_id,omitempty"`
	RecipientUserID int64              `json:"recipient_user_id"`
	NotificationType NotificationType  `json:"notification_type"`
	Status          NotificationStatus `json:"status"`
	ReadAt          *time.Time         `json:"read_at,omitempty"`
	Email           string             `json:"email,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	CompanyID       int64              `json:"company_id"`
}

func (n *Notification) Validate() error {
	if n.MessageID == nil && n.ActivityID == nil {
		return platformerrors.Validation("either message_id or activity_id is required", nil)
	}
	if n.RecipientUserID <= 0 {
		return platformerrors.Validation("recipient_user_id is required", nil)
	}
	if n.NotificationType == "" {
		n.NotificationType = NotificationTypeInbox
	}
	if n.Status == "" {
		n.Status = NotificationStatusUnread
	}
	if n.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	return nil
}

func (n *Notification) MarkRead(now time.Time) error {
	if n.Status == NotificationStatusRead {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	n.Status = NotificationStatusRead
	n.ReadAt = &now
	return nil
}

type EmailStatus string

const (
	EmailStatusQueued     EmailStatus = "queued"
	EmailStatusProcessing EmailStatus = "processing"
	EmailStatusSent       EmailStatus = "sent"
	EmailStatusFailed     EmailStatus = "failed"
	EmailStatusDeadLetter EmailStatus = "dead_letter"
)

type EmailQueueItem struct {
	ID             int64       `json:"id"`
	NotificationID *int64      `json:"notification_id,omitempty"`
	RecipientEmail string      `json:"recipient_email"`
	Subject        string      `json:"subject"`
	Body           string      `json:"body"`
	Status         EmailStatus `json:"status"`
	Attempts       int         `json:"attempts"`
	NextAttemptAt  time.Time   `json:"next_attempt_at"`
	LastError      string      `json:"last_error,omitempty"`
	SentAt         *time.Time  `json:"sent_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	CompanyID      int64       `json:"company_id"`
}

func (e *EmailQueueItem) Validate() error {
	if e.RecipientEmail == "" {
		return platformerrors.Validation("recipient_email is required", nil)
	}
	if e.Subject == "" {
		return platformerrors.Validation("subject is required", nil)
	}
	if e.Body == "" {
		return platformerrors.Validation("body is required", nil)
	}
	if e.Status == "" {
		e.Status = EmailStatusQueued
	}
	if e.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	return nil
}
