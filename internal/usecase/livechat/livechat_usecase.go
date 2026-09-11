package livechatusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/livechat"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo livechat.Repository
}

func New(repo livechat.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) InitSession(ctx context.Context, channelID int64, visitorUUID, name, email string) (*livechat.Session, error) {
	if channelID <= 0 || strings.TrimSpace(visitorUUID) == "" {
		return nil, platformerrors.Validation("channel_id and visitor_uuid are required", nil)
	}
	channel, err := u.repo.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, livechat.ErrChannelNotFound
	}
	if !channel.Active {
		return nil, platformerrors.Conflict("livechat channel is inactive")
	}
	if existing, err := u.repo.GetSessionByUUID(ctx, visitorUUID); err == nil && existing != nil && existing.Status == livechat.SessionStatusActive {
		return existing, nil
	}
	session := &livechat.Session{ChannelID: channelID, VisitorUUID: strings.TrimSpace(visitorUUID), VisitorName: strings.TrimSpace(name), VisitorEmail: strings.TrimSpace(email), Status: livechat.SessionStatusActive, CreatedAt: time.Now().UTC()}
	if session.VisitorName == "" {
		session.VisitorName = "Visitor"
	}
	if err := u.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (u *UseCase) SendMessage(ctx context.Context, sessionID int64, senderType livechat.SenderType, senderID *int64, body, fileURL string) (*livechat.Message, error) {
	session, err := u.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, livechat.ErrSessionNotFound
	}
	if session.Status == livechat.SessionStatusClosed {
		return nil, livechat.ErrSessionClosed
	}
	message := &livechat.Message{SessionID: sessionID, SenderType: senderType, SenderID: senderID, Body: strings.TrimSpace(body), FileURL: strings.TrimSpace(fileURL), CreatedAt: time.Now().UTC()}
	if err := message.Validate(); err != nil {
		return nil, err
	}
	if err := u.repo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}
	return message, nil
}

func (u *UseCase) CloseSession(ctx context.Context, sessionID int64) error {
	session, err := u.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return livechat.ErrSessionNotFound
	}
	if err := session.Close(); err != nil {
		return err
	}
	return u.repo.UpdateSession(ctx, session)
}

func (u *UseCase) GetChannel(ctx context.Context, id int64) (*livechat.Channel, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("channel_id is required", nil)
	}
	channel, err := u.repo.GetChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, livechat.ErrChannelNotFound
	}
	return channel, nil
}

func (u *UseCase) GetActiveSessions(ctx context.Context) ([]livechat.Session, error) {
	sessions, err := u.repo.ListSessions(ctx, 0)
	if err != nil {
		return nil, err
	}
	result := make([]livechat.Session, 0, len(sessions))
	for _, session := range sessions {
		if session.Status == livechat.SessionStatusActive {
			result = append(result, session)
		}
	}
	return result, nil
}

func (u *UseCase) RateSession(ctx context.Context, sessionID int64, score int, comment string) error {
	session, err := u.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return livechat.ErrSessionNotFound
	}
	if err := session.AddRating(score, comment); err != nil {
		return err
	}
	return u.repo.UpdateSession(ctx, session)
}

func (u *UseCase) JoinSession(ctx context.Context, sessionID int64, operatorID int64) error {
	session, err := u.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return livechat.ErrSessionNotFound
	}
	channel, err := u.repo.GetChannel(ctx, session.ChannelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return livechat.ErrChannelNotFound
	}
	if !channel.IsOperator(operatorID) {
		return livechat.ErrNotOperator
	}
	if err := session.AssignOperator(operatorID); err != nil {
		return err
	}
	return u.repo.UpdateSession(ctx, session)
}

func (u *UseCase) ConvertToTicket(ctx context.Context, sessionID int64) (int64, error) {
	if sessionID <= 0 {
		return 0, platformerrors.Validation("session_id is required", nil)
	}
	session, err := u.repo.GetSession(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	if session == nil {
		return 0, livechat.ErrSessionNotFound
	}
	if session.ConvertedTicketID == nil {
		return 0, platformerrors.NotImplemented(fmt.Sprintf("conversion of livechat session %d requires helpdesk integration", sessionID))
	}
	return *session.ConvertedTicketID, nil
}
