package metrics

import (
	"sync"
	"time"
)

const (
	// DefaultWindowSeconds is the duration of the rolling window.
	DefaultWindowSeconds = 60
	// numLatencyBuckets is len(LatencyBuckets) + 1 (for the +Inf overflow bucket).
	numLatencyBuckets = 12
)

// windowSlot stores raw counts, latency sum, response bytes, and bucket counts for a single second.
type windowSlot struct {
	timestamp int64
	count     uint64
	sum       float64 // duration in seconds
	bytesSum  uint64
	buckets   [numLatencyBuckets]uint64
}

// WindowSnapshot contains instantaneous aggregated traffic stats over the active window.
type WindowSnapshot struct {
	Count            uint64
	AvgMs            float64
	P50Ms            float64
	P95Ms            float64
	P99Ms            float64
	AvgResponseBytes float64
}

// WindowTracker maintains a circular buffer of 1-second slots over a 60-second sliding window.
// It is fully thread-safe and allocates zero memory on the hot request path.
type WindowTracker struct {
	mu    sync.RWMutex
	slots [DefaultWindowSeconds]windowSlot
	nowFn func() time.Time
}

// NewWindowTracker constructs an initialized WindowTracker.
func NewWindowTracker() *WindowTracker {
	return &WindowTracker{}
}

func (w *WindowTracker) timeNow() time.Time {
	if w.nowFn != nil {
		return w.nowFn()
	}
	return time.Now()
}

// bucketIndex maps a duration in seconds to its index in [0, 11].
func bucketIndex(val float64) int {
	for i, bound := range LatencyBuckets {
		if val <= bound {
			return i
		}
	}
	return len(LatencyBuckets)
}

// Observe records a single completed request duration and response size into the active slot.
func (w *WindowTracker) Observe(duration time.Duration, bytes int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	sec := w.timeNow().Unix()
	idx := sec % DefaultWindowSeconds

	slot := &w.slots[idx]
	if slot.timestamp != sec {
		*slot = windowSlot{timestamp: sec}
	}

	dSec := duration.Seconds()
	slot.count++
	slot.sum += dSec
	if bytes > 0 {
		slot.bytesSum += uint64(bytes)
	}
	slot.buckets[bucketIndex(dSec)]++
}

// Snapshot computes the current windowed statistics for all requests within the last 60 seconds.
// If no requests occurred in the active window, it returns zeroed values (baseline idle state).
func (w *WindowTracker) Snapshot() WindowSnapshot {
	w.mu.RLock()
	defer w.mu.RUnlock()

	nowSec := w.timeNow().Unix()
	var totalCount uint64
	var totalSum float64
	var totalBytes uint64
	var bucketCounts [numLatencyBuckets]uint64

	for i := range w.slots {
		slot := &w.slots[i]
		age := nowSec - slot.timestamp
		if age >= 0 && age < DefaultWindowSeconds {
			totalCount += slot.count
			totalSum += slot.sum
			totalBytes += slot.bytesSum
			for b := 0; b < numLatencyBuckets; b++ {
				bucketCounts[b] += slot.buckets[b]
			}
		}
	}

	if totalCount == 0 {
		return WindowSnapshot{}
	}

	snap := WindowSnapshot{
		Count:            totalCount,
		AvgMs:            (totalSum / float64(totalCount)) * 1000,
		AvgResponseBytes: float64(totalBytes) / float64(totalCount),
	}

	// Convert non-cumulative slot bucket counts to cumulative histogram buckets
	var cumulative float64
	var sortedBuckets []histBucket
	for i, bound := range LatencyBuckets {
		cumulative += float64(bucketCounts[i])
		sortedBuckets = append(sortedBuckets, histBucket{upperBound: bound, count: cumulative})
	}

	snap.P50Ms = calculateQuantile(0.50, float64(totalCount), sortedBuckets) * 1000
	snap.P95Ms = calculateQuantile(0.95, float64(totalCount), sortedBuckets) * 1000
	snap.P99Ms = calculateQuantile(0.99, float64(totalCount), sortedBuckets) * 1000

	return snap
}

// ProbeTracker tracks the most recent latency of health probes (readyz, livez).
type ProbeTracker struct {
	mu     sync.RWMutex
	readyz float64
	livez  float64
}

// NewProbeTracker creates a new ProbeTracker.
func NewProbeTracker() *ProbeTracker {
	return &ProbeTracker{}
}

// Observe records the latest duration of a probe.
func (p *ProbeTracker) Observe(probe string, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sec := duration.Seconds()
	switch probe {
	case ProbeReadyz:
		p.readyz = sec
	case ProbeLivez:
		p.livez = sec
	}
}

// LatenciesMs returns the latest measured probe latencies in milliseconds.
func (p *ProbeTracker) LatenciesMs() (readyzMs, livezMs float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.readyz * 1000, p.livez * 1000
}
