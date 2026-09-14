package examples

import (
	"context"
	"sync"
	"time"
)

type auditContextKey struct{ name string }

var (
	userContextKey   = &auditContextKey{name: "user_id"}
	tenantContextKey = &auditContextKey{name: "tenant_id"}
)

// AuditFields encapsulates standard change tracking embedded into entities.
type AuditFields struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy *int64    `json:"created_by,omitempty"`
	UpdatedBy *int64    `json:"updated_by,omitempty"`
}

// NewAuditFields creates initialized audit metadata using context identity.
func NewAuditFields(ctx context.Context) AuditFields {
	now := time.Now().UTC()
	uid := UserFromContext(ctx)
	return AuditFields{
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: uid,
		UpdatedBy: uid,
	}
}

// Touch refreshes the modification timestamp and modifier ID.
func (f *AuditFields) Touch(ctx context.Context) {
	f.UpdatedAt = time.Now().UTC()
	if uid := UserFromContext(ctx); uid != nil {
		f.UpdatedBy = uid
	}
}

// WithUser adds the actor ID to the request context.
func WithUser(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}

// UserFromContext extracts the actor ID if present.
func UserFromContext(ctx context.Context) *int64 {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(userContextKey).(int64); ok {
		return &v
	}
	return nil
}

// NotificationEvent represents an internal application event.
type NotificationEvent struct {
	Channel   string
	Payload   any
	Timestamp time.Time
}

// NotificationBus provides in-process, thread-safe non-blocking event distribution.
type NotificationBus struct {
	mu          sync.RWMutex
	subscribers map[int64]map[chan NotificationEvent]struct{}
}

// NewNotificationBus instantiates an empty bus.
func NewNotificationBus() *NotificationBus {
	return &NotificationBus{
		subscribers: make(map[int64]map[chan NotificationEvent]struct{}),
	}
}

// NotifyUser dispatches an event to all subscriber channels belonging to the user.
// Non-blocking: if a subscriber buffer is full, the message is skipped to prevent blocking.
func (b *NotificationBus) NotifyUser(ctx context.Context, userID int64, channel string, payload any) error {
	if userID <= 0 {
		return nil
	}

	event := NotificationEvent{
		Channel:   channel,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[userID] {
		select {
		case ch <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Buffer full - drop message to preserve publisher throughput
		}
	}

	return nil
}

// Subscribe opens a subscription for a given user ID, returning a receive channel and cancel func.
func (b *NotificationBus) Subscribe(userID int64) (<-chan NotificationEvent, func()) {
	events := make(chan NotificationEvent, 16)

	b.mu.Lock()
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = make(map[chan NotificationEvent]struct{})
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
