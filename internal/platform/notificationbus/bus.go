package notificationbus

import (
	"context"
	"sync"
)

// Event is a user-scoped real-time notification.
type Event struct {
	Channel string `json:"channel"`
	Payload any    `json:"payload"`
}

// Bus delivers notification events to subscribers in the current process.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[int64]map[chan Event]struct{}
}

func New() *Bus {
	return &Bus{subscribers: make(map[int64]map[chan Event]struct{})}
}

func (b *Bus) NotifyUser(ctx context.Context, userID int64, channel string, payload any) error {
	if userID <= 0 {
		return nil
	}

	event := Event{Channel: channel, Payload: payload}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for subscriber := range b.subscribers[userID] {
		select {
		case subscriber <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// A slow client must not block activity creation for other clients.
		}
	}
	return nil
}

func (b *Bus) Subscribe(userID int64) (<-chan Event, func()) {
	events := make(chan Event, 16)
	b.mu.Lock()
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = make(map[chan Event]struct{})
	}
	b.subscribers[userID][events] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			delete(b.subscribers[userID], events)
			if len(b.subscribers[userID]) == 0 {
				delete(b.subscribers, userID)
			}
			close(events)
		})
	}
	return events, unsubscribe
}
