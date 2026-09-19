package examples

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"
)

// Clock provides a time abstraction interface to allow deterministic unit testing
// without depending on actual wall-clock time.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
	Since(t time.Time) time.Duration
}

// RealClock implements Clock using standard library time functions.
type RealClock struct{}

func NewRealClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now().UTC()
}

func (c *RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (c *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// MockClock provides a thread-safe deterministic clock for unit and integration tests.
type MockClock struct {
	mu      sync.RWMutex
	current time.Time
}

func NewMockClock(initial time.Time) *MockClock {
	return &MockClock{current: initial.UTC()}
}

func (m *MockClock) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *MockClock) Sleep(d time.Duration) {
	m.Advance(d)
}

func (m *MockClock) Since(t time.Time) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current.Sub(t)
}

func (m *MockClock) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = t.UTC()
}

func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.current = m.current.Add(d)
}

// IDGenerator provides an abstract interface for generating unique system identifiers.
type IDGenerator interface {
	NewID() string
}

// TimeOrderedIDGenerator generates 128-bit time-ordered unique IDs (UUIDv7-compatible format).
// Encodes high 48-bit UNIX milliseconds timestamp for B-Tree index locality,
// followed by 80 bits of cryptographically secure random entropy.
type TimeOrderedIDGenerator struct {
	clock Clock
}

func NewTimeOrderedIDGenerator(clock Clock) *TimeOrderedIDGenerator {
	if clock == nil {
		clock = NewRealClock()
	}
	return &TimeOrderedIDGenerator{clock: clock}
}

func (g *TimeOrderedIDGenerator) NewID() string {
	ms := uint64(g.clock.Now().UnixMilli())

	var raw [16]byte
	// 48-bit timestamp in big-endian
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)

	// Fill remaining 10 bytes with cryptographic randomness
	_, _ = rand.Read(raw[6:])

	// Version 7: set bits 48-51 to 0111
	raw[6] = (raw[6] & 0x0F) | 0x70
	// Variant: set bits 64-65 to 10
	raw[8] = (raw[8] & 0x3F) | 0x80

	var buf [36]byte
	hex.Encode(buf[0:8], raw[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], raw[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], raw[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], raw[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], raw[10:16])

	return string(buf[:])
}

// WorkerPool provides bounded concurrent task execution for in-process background jobs.
type WorkerPool struct {
	tasks chan func(ctx context.Context)
	wg    sync.WaitGroup
}

func NewWorkerPool(workers, queueSize int) *WorkerPool {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 1 {
		queueSize = workers * 2
	}

	p := &WorkerPool{
		tasks: make(chan func(ctx context.Context), queueSize),
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.tasks {
				task(context.Background())
			}
		}()
	}

	return p
}

func (p *WorkerPool) Submit(task func(ctx context.Context)) bool {
	select {
	case p.tasks <- task:
		return true
	default:
		return false // Queue full: backpressure applied
	}
}

func (p *WorkerPool) Shutdown() {
	close(p.tasks)
	p.wg.Wait()
}

// Suppress unused import warning for binary package
var _ = binary.BigEndian
