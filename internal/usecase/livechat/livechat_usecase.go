package livechatusecase

import (
	"context"
	"cashflow_backend/internal/domain/livechat"
)

type UseCase struct {
	repo livechat.Repository
}

func New(repo livechat.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) InitSession(ctx context.Context, channelID int64, visitorUUID, name, email string) (*livechat.Session, error) {
	return nil, nil
}

func (u *UseCase) SendMessage(ctx context.Context, sessionID int64, senderType livechat.SenderType, senderID *int64, body, fileURL string) (*livechat.Message, error) {
	return nil, nil
}

func (u *UseCase) CloseSession(ctx context.Context, sessionID int64) error {
	return nil
}

func (u *UseCase) GetChannel(ctx context.Context, id int64) (*livechat.Channel, error) {
	return nil, nil
}

func (u *UseCase) GetActiveSessions(ctx context.Context) ([]livechat.Session, error) {
	return nil, nil
}

func (u *UseCase) RateSession(ctx context.Context, sessionID int64, score int, comment string) error {
	return nil
}

func (u *UseCase) JoinSession(ctx context.Context, sessionID int64, operatorID int64) error {
	return nil
}

func (u *UseCase) ConvertToTicket(ctx context.Context, sessionID int64) (int64, error) {
	return 0, nil
}
