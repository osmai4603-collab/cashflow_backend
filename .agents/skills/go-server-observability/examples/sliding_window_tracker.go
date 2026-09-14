package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Zero-Allocation 60-Second Sliding Window Latency Tracker
// =============================================================================

const windowBuckets = 60

type secondBucket struct {
	timestamp int64
	count     int64
	sumMicros int64
	// Logarithmic latency histogram buckets (in microseconds):
	// <1ms, <5ms, <10ms, <25ms, <50ms, <100ms, <250ms, <500ms, <1s, <2s, >=2s
	hist [11]int64
}

// WindowTracker provides a zero-allocation rolling 60-second latency window.
type WindowTracker struct {
	mu      sync.RWMutex
	buckets [windowBuckets]secondBucket
}

func NewWindowTracker() *WindowTracker {
	return &WindowTracker{}
}

// Observe records a single duration in the current 1-second slice.
func (w *WindowTracker) Observe(d time.Duration) {
	now := time.Now()
	sec := now.Unix()
	idx := sec % windowBuckets
	micros := d.Microseconds()

	bucket := &w.buckets[idx]

	// Reset bucket if it belongs to a past minute
	if atomic.LoadInt64(&bucket.timestamp) != sec {
		w.mu.Lock()
		if bucket.timestamp != sec {
			bucket.timestamp = sec
			bucket.count = 0
			bucket.sumMicros = 0
			for i := range bucket.hist {
				bucket.hist[i] = 0
			}
		}
		w.mu.Unlock()
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	bucket.count++
	bucket.sumMicros += micros

	switch {
	case micros < 1000:
		bucket.hist[0]++
	case micros < 5000:
		bucket.hist[1]++
	case micros < 10000:
		bucket.hist[2]++
	case micros < 25000:
		bucket.hist[3]++
	case micros < 50000:
		bucket.hist[4]++
	case micros < 100000:
		bucket.hist[5]++
	case micros < 250000:
		bucket.hist[6]++
	case micros < 500000:
		bucket.hist[7]++
	case micros < 1000000:
		bucket.hist[8]++
	case micros < 2000000:
		bucket.hist[9]++
	default:
		bucket.hist[10]++
	}
}

// WindowStats holds the aggregated metrics for the last 60 seconds.
type WindowStats struct {
	TotalRequests int64   `json:"total_requests"`
	RPS           float64 `json:"rps"`
	AverageMs     float64 `json:"avg_ms"`
	P50Ms         float64 `json:"p50_ms"`
	P95Ms         float64 `json:"p95_ms"`
	P99Ms         float64 `json:"p99_ms"`
	IsIdle        bool    `json:"is_idle"`
}

// Snapshot aggregates the active buckets within the last 60 seconds.
func (w *WindowTracker) Snapshot() WindowStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	nowSec := time.Now().Unix()
	minSec := nowSec - windowBuckets

	var totalReq int64
	var totalMicros int64
	var totalHist [11]int64

	for i := 0; i < windowBuckets; i++ {
		b := &w.buckets[i]
		if b.timestamp > minSec && b.timestamp <= nowSec {
			totalReq += b.count
			totalMicros += b.sumMicros
			for h := range b.hist {
				totalHist[h] += b.hist[h]
			}
		}
	}

	if totalReq == 0 {
		return WindowStats{IsIdle: true}
	}

	avgMs := float64(totalMicros) / float64(totalReq) / 1000.0
	rps := float64(totalReq) / float64(windowBuckets)

	return WindowStats{
		TotalRequests: totalReq,
		RPS:           rps,
		AverageMs:     avgMs,
		P50Ms:         estimateQuantile(totalHist, totalReq, 0.50),
		P95Ms:         estimateQuantile(totalHist, totalReq, 0.95),
		P99Ms:         estimateQuantile(totalHist, totalReq, 0.99),
		IsIdle:        false,
	}
}

func estimateQuantile(hist [11]int64, total int64, q float64) float64 {
	target := int64(float64(total) * q)
	var accumulated int64

	thresholdsMs := [11]float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000}
	for i, count := range hist {
		accumulated += count
		if accumulated >= target {
			return thresholdsMs[i]
		}
	}
	return thresholdsMs[10]
}
