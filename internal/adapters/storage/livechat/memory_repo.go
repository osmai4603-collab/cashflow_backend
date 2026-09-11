package livechatstorage

import (
	"cashflow_backend/internal/domain/livechat"
	"context"
	"sort"
	"sync"
)

type MemoryRepo struct {
	mu       sync.RWMutex
	channels map[int64]*livechat.Channel
	sessions map[int64]*livechat.Session
	messages map[int64]*livechat.Message
	nextID   int64
}

func (r *MemoryRepo) allocateID() int64 { r.nextID++; return r.nextID }

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
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]livechat.Channel, 0)
	for _, channel := range r.channels {
		if companyID == 0 || channel.CompanyID == companyID {
			result = append(result, *channel)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateChannel(ctx context.Context, channel *livechat.Channel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if channel.ID == 0 {
		channel.ID = r.allocateID()
	}
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
		if s.VisitorUUID == uuid {
			return s, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) CreateSession(ctx context.Context, session *livechat.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session.ID == 0 {
		session.ID = r.allocateID()
	}
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]livechat.Session, 0)
	for _, session := range r.sessions {
		if channelID == 0 || session.ChannelID == channelID {
			result = append(result, *session)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateMessage(ctx context.Context, msg *livechat.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if msg.ID == 0 {
		msg.ID = r.allocateID()
	}
	r.messages[msg.ID] = msg
	return nil
}

func (r *MemoryRepo) ListMessages(ctx context.Context, sessionID int64) ([]livechat.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]livechat.Message, 0)
	for _, message := range r.messages {
		if message.SessionID == sessionID {
			result = append(result, *message)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) ListCannedResponses(ctx context.Context, companyID int64) ([]livechat.CannedResponse, error) {
	return nil, nil
}
