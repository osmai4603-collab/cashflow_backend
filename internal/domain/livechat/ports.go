package livechat

import (
	"context"
)

type Repository interface {
	// Channel operations
	GetChannel(ctx context.Context, id int64) (*Channel, error)
	ListChannels(ctx context.Context, companyID int64) ([]Channel, error)
	CreateChannel(ctx context.Context, channel *Channel) error
	UpdateChannel(ctx context.Context, channel *Channel) error

	// Session operations
	GetSession(ctx context.Context, id int64) (*Session, error)
	GetSessionByUUID(ctx context.Context, uuid string) (*Session, error)
	CreateSession(ctx context.Context, session *Session) error
	UpdateSession(ctx context.Context, session *Session) error
	ListSessions(ctx context.Context, channelID int64) ([]Session, error)

	// Message operations
	CreateMessage(ctx context.Context, msg *Message) error
	ListMessages(ctx context.Context, sessionID int64) ([]Message, error)

	// Canned Response
	ListCannedResponses(ctx context.Context, companyID int64) ([]CannedResponse, error)
}

type Service interface {
	// Public Visitor API
	InitSession(ctx context.Context, channelID int64, visitorUUID, name, email string) (*Session, error)
	SendMessage(ctx context.Context, sessionID int64, senderType SenderType, senderID *int64, body, fileURL string) (*Message, error)
	CloseSession(ctx context.Context, sessionID int64) error
	RateSession(ctx context.Context, sessionID int64, score int, comment string) error

	// Operator API
	JoinSession(ctx context.Context, sessionID int64, operatorID int64) error
	ConvertToTicket(ctx context.Context, sessionID int64) (int64, error)
}
