package activityusecase

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"cashflow_backend/internal/domain/activity"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

type UseCase struct {
	repo      activity.ActivityRepository
	typeRepo  activity.ActivityTypeRepository
	msgRepo   activity.MessageRepository
	notifRepo activity.NotificationRepository
	emailRepo activity.EmailQueueRepository
	userRepo  activity.UserRepository
	bus       activity.Bus
}

func NewUseCase(
	repo activity.ActivityRepository,
	typeRepo activity.ActivityTypeRepository,
	msgRepo activity.MessageRepository,
	notifRepo activity.NotificationRepository,
	emailRepo activity.EmailQueueRepository,
	bus activity.Bus,
	userRepo activity.UserRepository,
) *UseCase {
	return &UseCase{
		repo:      repo,
		typeRepo:  typeRepo,
		msgRepo:   msgRepo,
		notifRepo: notifRepo,
		emailRepo: emailRepo,
		userRepo:  userRepo,
		bus:       bus,
	}
}

// --- Activity Type Operations ---

func (u *UseCase) CreateType(ctx context.Context, t *activity.ActivityType) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return u.typeRepo.CreateType(ctx, t)
}

func (u *UseCase) GetType(ctx context.Context, id int64) (*activity.ActivityType, error) {
	return u.typeRepo.GetTypeByID(ctx, id)
}

func (u *UseCase) UpdateType(ctx context.Context, t *activity.ActivityType) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return u.typeRepo.UpdateType(ctx, t)
}

func (u *UseCase) DeleteType(ctx context.Context, id int64) error {
	return u.typeRepo.DeleteType(ctx, id)
}

func (u *UseCase) ListTypes(ctx context.Context, companyID *int64) ([]activity.ActivityType, error) {
	return u.typeRepo.ListTypes(ctx, companyID)
}

// --- Activity Operations ---

var allowedModels = map[string]bool{
	"res.partner":      true,
	"product.template": true,
	"sale.order":       true,
	"purchase.order":   true,
	"account.move":     true,
	"hr.employee":      true,
	"project.project":  true,
	"project.task":     true,
}

func (u *UseCase) Schedule(ctx context.Context, a *activity.Activity) error {
	if err := a.Validate(); err != nil {
		return err
	}

	if a.ResModel != "" && !allowedModels[a.ResModel] {
		return platformerrors.Validation("unsupported resource model", map[string]string{"res_model": a.ResModel})
	}

	// Verify activity type exists
	at, err := u.typeRepo.GetTypeByID(ctx, a.ActivityTypeID)
	if err != nil {
		return err
	}

	// If summary is empty, use default from type
	if a.Summary == "" {
		a.Summary = at.Summary
		if a.Summary == "" {
			a.Summary = at.Name
		}
	}

	if err := u.repo.CreateActivity(ctx, a); err != nil {
		return err
	}

	// Create notification for assigned user if not the creator
	if a.CreatedBy != nil && *a.CreatedBy != a.AssignedUserID {
		u.notifyAssignment(ctx, a)
	}

	return nil
}

func (u *UseCase) GetActivity(ctx context.Context, id int64) (*activity.Activity, error) {
	return u.repo.GetActivityByID(ctx, id)
}

func (u *UseCase) UpdateActivity(ctx context.Context, a *activity.Activity) error {
	if err := a.Validate(); err != nil {
		return err
	}
	return u.repo.UpdateActivity(ctx, a)
}

func (u *UseCase) Complete(ctx context.Context, id int64, feedback string) (*activity.Activity, error) {
	now := time.Now().UTC()
	a, err := u.repo.CompleteActivity(ctx, id, now, feedback)
	if err != nil {
		return nil, err
	}

	// Create a message record for the completion
	msg := &activity.Message{
		Body:        feedback,
		MessageType: activity.MessageTypeNotification,
		ResModel:    a.ResModel,
		ResID:       a.ResID,
		ActivityID:  &a.ID,
		CompanyID:   a.CompanyID,
	}
	if a.Summary != "" {
		msg.Subject = "Activity Completed: " + a.Summary
	} else {
		msg.Subject = "Activity Completed"
	}

	_ = u.msgRepo.CreateMessage(ctx, msg)

	return a, nil
}

func (u *UseCase) ListActivities(ctx context.Context, filter activity.ActivityFilter) (pagination.PageResult[activity.Activity], error) {
	return u.repo.ListActivities(ctx, filter)
}

func (u *UseCase) notifyAssignment(ctx context.Context, a *activity.Activity) {
	msg := &activity.Message{
		Subject:     "New Activity Assigned",
		Body:        "You have been assigned a new activity: " + a.Summary,
		MessageType: activity.MessageTypeNotification,
		ResModel:    a.ResModel,
		ResID:       a.ResID,
		ActivityID:  &a.ID,
		CompanyID:   a.CompanyID,
	}
	if err := u.msgRepo.CreateMessage(ctx, msg); err == nil {
		notif := &activity.Notification{
			MessageID:        &msg.ID,
			ActivityID:       &a.ID,
			RecipientUserID:  a.AssignedUserID,
			NotificationType: activity.NotificationTypeInbox,
			Status:           activity.NotificationStatusUnread,
			CompanyID:        a.CompanyID,
		}
		if err := u.notifRepo.CreateNotification(ctx, notif); err == nil {
			recipientEmail := ""
			if u.userRepo != nil {
				if userRecord, err := u.userRepo.GetByID(ctx, a.AssignedUserID); err == nil && userRecord != nil && userRecord.EmailNotificationsEnabled {
					recipientEmail = strings.TrimSpace(userRecord.Email)
					if recipientEmail != "" {
						if _, err := mail.ParseAddress(recipientEmail); err != nil {
							recipientEmail = ""
						}
					}
				}
			}

			if recipientEmail != "" {
				emailItem := &activity.EmailQueueItem{
					NotificationID: &notif.ID,
					RecipientEmail: recipientEmail,
					Subject:        msg.Subject,
					Body:           msg.Body,
					Status:         activity.EmailStatusQueued,
					NextAttemptAt:  time.Now().UTC(),
					CompanyID:      a.CompanyID,
				}
				_ = u.emailRepo.PushEmail(ctx, emailItem)
			}

			if u.bus != nil {
				_ = u.bus.NotifyUser(ctx, a.AssignedUserID, "mail.notification", notif)
			}
		}
	}
}

// --- Notification Operations ---

func (u *UseCase) ListNotifications(ctx context.Context, userID int64, status *activity.NotificationStatus, page pagination.PageRequest) (pagination.PageResult[activity.Notification], error) {
	return u.notifRepo.ListNotificationsForUser(ctx, userID, status, page)
}

func (u *UseCase) MarkRead(ctx context.Context, id int64) error {
	n, err := u.notifRepo.GetNotificationByID(ctx, id)
	if err != nil {
		return err
	}
	if err := n.MarkRead(time.Now().UTC()); err != nil {
		return err
	}
	return u.notifRepo.UpdateNotification(ctx, n)
}

func (u *UseCase) MarkAllRead(ctx context.Context, userID int64, companyID int64) error {
	return u.notifRepo.MarkAllRead(ctx, userID, companyID)
}
