package activity

import (
	"context"
	"time"

	"cashflow_backend/internal/platform/pagination"
)

type ActivityFilter struct {
	AssignedUserID *int64
	State          *State
	ResModel       string
	ResID          *int64
	Active         *bool
	Page           pagination.PageRequest
}

type ActivityRepository interface {
	CreateActivity(ctx context.Context, value *Activity) error
	GetActivityByID(ctx context.Context, id int64) (*Activity, error)
	UpdateActivity(ctx context.Context, value *Activity) error
	CompleteActivity(ctx context.Context, id int64, now time.Time, feedback string) (*Activity, error)
	ListActivities(ctx context.Context, filter ActivityFilter) (pagination.PageResult[Activity], error)
}

type ActivityTypeRepository interface {
	CreateType(ctx context.Context, value *ActivityType) error
	GetTypeByID(ctx context.Context, id int64) (*ActivityType, error)
	UpdateType(ctx context.Context, value *ActivityType) error
	DeleteType(ctx context.Context, id int64) error
	ListTypes(ctx context.Context, companyID *int64) ([]ActivityType, error)
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, value *Message) error
	GetMessageByID(ctx context.Context, id int64) (*Message, error)
	ListMessagesByResource(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[Message], error)
}

type NotificationRepository interface {
	CreateNotification(ctx context.Context, value *Notification) error
	GetNotificationByID(ctx context.Context, id int64) (*Notification, error)
	UpdateNotification(ctx context.Context, n *Notification) error
	ListNotificationsForUser(ctx context.Context, userID int64, status *NotificationStatus, page pagination.PageRequest) (pagination.PageResult[Notification], error)
	MarkAllRead(ctx context.Context, userID int64, companyID int64) error
}

type EmailQueueRepository interface {
	PushEmail(ctx context.Context, value *EmailQueueItem) error
	PopEmails(ctx context.Context, limit int) ([]EmailQueueItem, error)
	UpdateEmail(ctx context.Context, value *EmailQueueItem) error
}

type Bus interface {
	NotifyUser(ctx context.Context, userID int64, channel string, payload any) error
}
