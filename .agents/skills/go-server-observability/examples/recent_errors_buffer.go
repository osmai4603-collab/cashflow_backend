package observability

import (
	"sync"
	"time"
)

// =============================================================================
// Live In-Memory HTTP Error Diagnostics Ring Buffer
// =============================================================================

// HTTPErrorEvent captures actionable diagnostic context on failed requests.
type HTTPErrorEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMs int64     `json:"duration_ms"`
	ClientIP   string    `json:"client_ip,omitempty"`
}

// ErrorRingBuffer maintains the last N HTTP errors in a thread-safe circular buffer.
type ErrorRingBuffer struct {
	mu       sync.RWMutex
	capacity int
	events   []HTTPErrorEvent
	cursor   int
	isFull   bool
}

// NewErrorRingBuffer initializes a circular buffer with the given capacity.
func NewErrorRingBuffer(capacity int) *ErrorRingBuffer {
	if capacity <= 0 {
		capacity = 10
	}
	return &ErrorRingBuffer{
		capacity: capacity,
		events:   make([]HTTPErrorEvent, capacity),
	}
}

// Push records an error event into the ring buffer.
func (b *ErrorRingBuffer) Push(event HTTPErrorEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events[b.cursor] = event
	b.cursor = (b.cursor + 1) % b.capacity
	if b.cursor == 0 {
		b.isFull = true
	}
}

// Recent returns the recorded error events in reverse chronological order (newest first).
func (b *ErrorRingBuffer) Recent() []HTTPErrorEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := b.cursor
	if b.isFull {
		count = b.capacity
	}

	result := make([]HTTPErrorEvent, 0, count)
	if !b.isFull {
		for i := b.cursor - 1; i >= 0; i-- {
			result = append(result, b.events[i])
		}
		return result
	}

	// Buffer is full: newest starts at cursor - 1 wrapping around
	for i := 0; i < b.capacity; i++ {
		idx := (b.cursor - 1 - i + b.capacity) % b.capacity
		result = append(result, b.events[idx])
	}
	return result
}
