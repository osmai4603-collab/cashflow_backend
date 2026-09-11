package livechatstorage

import (
	"context"
	"cashflow_backend/internal/domain/livechat"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	channels map[int64]*livechat.Channel
	sessions map[int64]*livechat.Session
	messages map[int64]*livechat.Message
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		channels: make(map[int64]*livechat.Channel),
		sessions: make(map[int64]*livechat.Session),
		messages: make(map[int64]*livechat.Message),
	}
}

func (r *MemoryRepo) GetChannel(ctx context.Context, id int64) (*livechat.Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.channels[id], nil
}

func (r *MemoryRepo) ListChannels(ctx context.Context, companyID int64) ([]livechat.Channel, error) {
	return nil, nil
}

func (r *MemoryRepo) CreateChannel(ctx context.Context, channel *livechat.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[channel.ID] = channel
	return nil
}

func (r *MemoryRepo) UpdateChannel(ctx context.Context, channel *livechat.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.channels[channel.ID] = channel
	return nil
}

func (r *MemoryRepo) GetSession(ctx context.Context, id int64) (*livechat.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessions[id], nil
}

func (r *MemoryRepo) GetSessionByUUID(ctx context.Context, uuid string) (*livechat.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.sessions {
		if s.UUID == uuid {
			return s, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) CreateSession(ctx context.Context, session *livechat.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *MemoryRepo) UpdateSession(ctx context.Context, session *livechat.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *MemoryRepo) ListSessions(ctx context.Context, channelID int64) ([]livechat.Session, error) {
	return nil, nil
}

func (r *MemoryRepo) CreateMessage(ctx context.Context, msg *livechat.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[msg.ID] = msg
	return nil
}

func (r *MemoryRepo) ListMessages(ctx context.Context, sessionID int64) ([]livechat.Message, error) {
	return nil, nil
}

func (r *MemoryRepo) ListCannedResponses(ctx context.Context, companyID int64) ([]livechat.CannedResponse, error) {
	return nil, nil
}
