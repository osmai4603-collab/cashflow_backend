package activityusecase

import (
	"context"

	"cashflow_backend/internal/domain/activity"
)

// MessagePost records a message on a thread and notifies followers.
func (u *UseCase) MessagePost(ctx context.Context, msg *activity.Message) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	// 1. Create message
	if err := u.msgRepo.CreateMessage(ctx, msg); err != nil {
		return err
	}

	// 2. Save tracking values if any
	if len(msg.TrackingValues) > 0 {
		for i := range msg.TrackingValues {
			msg.TrackingValues[i].MessageID = msg.ID
		}
		if err := u.msgRepo.CreateTrackingValues(ctx, msg.TrackingValues); err != nil {
			return err
		}
	}

	// 3. Notify followers based on subtypes
	if msg.ResModel != "" && msg.ResID != nil {
		subtypeID := int64(0)
		if msg.SubtypeID != nil {
			subtypeID = *msg.SubtypeID
		}

		followers, err := u.followerRepo.GetFollowersForNotification(ctx, msg.ResModel, *msg.ResID, subtypeID)
		if err == nil {
			for _, f := range followers {
				// Avoid notifying the author
				if f.UserID != nil && (msg.AuthorID == nil || *msg.AuthorID != *f.UserID) {
					u.notifyUser(ctx, msg, *f.UserID)
				}
				// TODO: Handle partner_id notifications (external email)
			}
		}
	}

	return nil
}

func (u *UseCase) notifyUser(ctx context.Context, msg *activity.Message, userID int64) {
	notif := &activity.Notification{
		MessageID:        &msg.ID,
		RecipientUserID:  userID,
		NotificationType: activity.NotificationTypeInbox,
		Status:           activity.NotificationStatusUnread,
		CompanyID:        msg.CompanyID,
	}
	if err := u.notifRepo.CreateNotification(ctx, notif); err == nil {
		if u.bus != nil {
			_ = u.bus.NotifyUser(ctx, userID, "mail.notification", notif)
		}
	}
}

// Subscribe adds a follower to a resource thread.
func (u *UseCase) Subscribe(ctx context.Context, f *activity.Follower) error {
	if err := f.Validate(); err != nil {
		return err
	}
	return u.followerRepo.CreateFollower(ctx, f)
}

// Unsubscribe removes a follower from a resource thread.
func (u *UseCase) Unsubscribe(ctx context.Context, id int64) error {
	return u.followerRepo.DeleteFollower(ctx, id)
}

// ListFollowers returns all followers of a resource.
func (u *UseCase) ListFollowers(ctx context.Context, resModel string, resID int64) ([]activity.Follower, error) {
	return u.followerRepo.ListFollowers(ctx, resModel, resID)
}

// GetSubtypeByName finds a message subtype for a model or global.
func (u *UseCase) GetSubtypeByName(ctx context.Context, resModel, name string) (*activity.MessageSubtype, error) {
	return u.subtypeRepo.GetSubtypeByName(ctx, resModel, name)
}
