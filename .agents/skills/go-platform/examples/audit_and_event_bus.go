package examples

import (
	"context"
	"sync"
	"time"
)

// Unexported contextKey type avoids key collisions across packages (official Go recommendation).
type contextKey struct{ name string }

var (
	userContextKey   = contextKey{name: "user_id"}
	tenantContextKey = contextKey{name: "tenant_id"}
	traceContextKey  = contextKey{name: "trace_id"}
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

// WithTenant adds the tenant ID to the request context.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}

// TenantFromContext extracts the tenant ID if present.
func TenantFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(tenantContextKey).(string); ok {
		return v
	}
	return ""
}

// WithTraceID adds the distributed trace/correlation ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceContextKey, traceID)
}

// TraceIDFromContext extracts the trace ID if present.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(traceContextKey).(string); ok {
		return v
	}
	return ""
}

// Event represents a generic in-process application event.
type Event struct {
	Topic     string
	Payload   any
	Timestamp time.Time
}

// EventBus provides in-process, thread-safe, non-blocking topic-based publish/subscribe.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan Event]struct{}
}

// NewEventBus instantiates an empty bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]map[chan Event]struct{}),
	}
}

// Publish dispatches an event to all subscribers listening to the topic.
// Non-blocking: if a subscriber channel buffer is full, the message is dropped
// to guarantee that the publisher is never blocked by slow consumers.
func (b *EventBus) Publish(ctx context.Context, topic string, payload any) error {
	event := Event{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[topic] {
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

// Subscribe opens a subscription for a given topic, returning a receive channel and cancel func.
func (b *EventBus) Subscribe(topic string, bufferSize int) (<-chan Event, func()) {
	if bufferSize < 1 {
		bufferSize = 16
	}
	events := make(chan Event, bufferSize)

	b.mu.Lock()
	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[chan Event]struct{})
	}
	b.subscribers[topic][events] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			delete(b.subscribers[topic], events)
			if len(b.subscribers[topic]) == 0 {
				delete(b.subscribers, topic)
			}
			close(events)
		})
	}

	return events, unsubscribe
}
