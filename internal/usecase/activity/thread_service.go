package activityusecase

import (
	"context"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/activity"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

// PostMessageInput specifies parameters for adding a new message to a Chatter thread.
type PostMessageInput struct {
	Subject        string
	Body           string
	MessageType    activity.MessageType
	SubtypeID      *int64
	ParentID       *int64
	AuthorID       *int64
	TrackingValues []activity.TrackingValue
}

// ThreadService provides high-level Chatter and Thread orchestration.
// Corresponds to Odoo 19 mail.thread mixin.
type ThreadService struct {
	uc           *UseCase
	msgRepo      activity.MessageRepository
	followerRepo activity.FollowerRepository
	subtypeRepo  activity.SubtypeRepository
	notifRepo    activity.NotificationRepository
	bus          activity.Bus
	logger       *slog.Logger
}

// NewThreadService creates a new ThreadService instance.
func NewThreadService(
	uc *UseCase,
	msgRepo activity.MessageRepository,
	followerRepo activity.FollowerRepository,
	subtypeRepo activity.SubtypeRepository,
	notifRepo activity.NotificationRepository,
	bus activity.Bus,
	logger *slog.Logger,
) *ThreadService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ThreadService{
		uc:           uc,
		msgRepo:      msgRepo,
		followerRepo: followerRepo,
		subtypeRepo:  subtypeRepo,
		notifRepo:    notifRepo,
		bus:          bus,
		logger:       logger,
	}
}

// PostMessage posts a message onto a threadable entity, attaches tracking values if any,
// and distributes notifications to subscribed followers.
func (s *ThreadService) PostMessage(ctx context.Context, thread activity.Threadable, in PostMessageInput) (*activity.Message, error) {
	if thread == nil {
		return nil, platformerrors.Validation("thread record cannot be nil", nil)
	}

	resModel := thread.ThreadModel()
	resID := thread.ThreadID()
	companyID := thread.ThreadCompanyID()
	if companyID <= 0 {
		companyID = 1
	}

	msgType := in.MessageType
	if msgType == "" {
		msgType = activity.MessageTypeComment
	}

	msg := &activity.Message{
		Subject:        in.Subject,
		Body:           in.Body,
		MessageType:    msgType,
		ResModel:       resModel,
		ResID:          &resID,
		SubtypeID:      in.SubtypeID,
		ParentID:       in.ParentID,
		AuthorID:       in.AuthorID,
		TrackingValues: in.TrackingValues,
		CompanyID:      companyID,
		CreatedAt:      time.Now().UTC(),
	}

	if s.uc != nil {
		if err := s.uc.MessagePost(ctx, msg); err != nil {
			return nil, err
		}
	} else if s.msgRepo != nil {
		if err := s.msgRepo.CreateMessage(ctx, msg); err != nil {
			return nil, err
		}
		if len(msg.TrackingValues) > 0 {
			for i := range msg.TrackingValues {
				msg.TrackingValues[i].MessageID = msg.ID
			}
			_ = s.msgRepo.CreateTrackingValues(ctx, msg.TrackingValues)
		}
	}

	s.logger.InfoContext(ctx, "chatter message posted",
		"res_model", resModel,
		"res_id", resID,
		"message_id", msg.ID,
		"type", msg.MessageType,
	)

	return msg, nil
}

// Subscribe adds a follower to the thread.
func (s *ThreadService) Subscribe(ctx context.Context, thread activity.Threadable, partnerID *int64, userID *int64, subtypeIDs []int64) error {
	if thread == nil {
		return platformerrors.Validation("thread cannot be nil", nil)
	}

	companyID := thread.ThreadCompanyID()
	if companyID <= 0 {
		companyID = 1
	}

	follower := &activity.Follower{
		ResModel:   thread.ThreadModel(),
		ResID:      thread.ThreadID(),
		PartnerID:  partnerID,
		UserID:     userID,
		SubtypeIDs: subtypeIDs,
		CompanyID:  companyID,
	}

	if s.uc != nil {
		return s.uc.Subscribe(ctx, follower)
	}
	if s.followerRepo != nil {
		return s.followerRepo.CreateFollower(ctx, follower)
	}
	return nil
}

// Unsubscribe removes a follower from the thread by their follower record ID.
func (s *ThreadService) Unsubscribe(ctx context.Context, followerID int64) error {
	if s.uc != nil {
		return s.uc.Unsubscribe(ctx, followerID)
	}
	if s.followerRepo != nil {
		return s.followerRepo.DeleteFollower(ctx, followerID)
	}
	return nil
}

// ListThread retrieves the paginated history of messages in a thread.
func (s *ThreadService) ListThread(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[activity.Message], error) {
	if s.msgRepo == nil {
		return pagination.PageResult[activity.Message]{}, nil
	}
	return s.msgRepo.ListMessagesByResource(ctx, resModel, resID, page)
}

// ListFollowers retrieves all followers registered to the given entity.
func (s *ThreadService) ListFollowers(ctx context.Context, resModel string, resID int64) ([]activity.Follower, error) {
	if s.followerRepo == nil {
		return nil, nil
	}
	return s.followerRepo.ListFollowers(ctx, resModel, resID)
}
