package observability

import (
	"time"
)

// =============================================================================
// Database Connection Pool Contention Telemetry
// =============================================================================

// DBPoolMetrics models granular connection pool statistics.
type DBPoolMetrics struct {
	ActiveConns        int32         `json:"active_conns"`
	IdleConns          int32         `json:"idle_conns"`
	MaxConns           int32         `json:"max_conns"`
	WaitCount          int64         `json:"wait_count"`
	EmptyAcquireCount  int64         `json:"empty_acquire_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	WaitDurationMs     float64       `json:"wait_duration_ms"`
	SaturationPercent  float64       `json:"saturation_percent"`
	IsSaturated        bool          `json:"is_saturated"`
	HasBlockedAcquires bool          `json:"has_blocked_acquires"`
}

// DBPoolStatsProvider represents any pool implementation (e.g. pgxpool.Pool).
type DBPoolStatsProvider interface {
	TotalConns() int32
	IdleConns() int32
	MaxConns() int32
	AcquireCount() int64
	EmptyAcquireCount() int64
	AcquireDuration() time.Duration
}

// CollectPoolMetrics transforms raw pool counters into actionable contention metrics.
func CollectPoolMetrics(p DBPoolStatsProvider) DBPoolMetrics {
	if p == nil {
		return DBPoolMetrics{}
	}

	total := p.TotalConns()
	idle := p.IdleConns()
	active := total - idle
	if active < 0 {
		active = 0
	}
	max := p.MaxConns()
	waitCount := p.AcquireCount()
	emptyAcquires := p.EmptyAcquireCount()
	waitDur := p.AcquireDuration()

	var saturation float64
	if max > 0 {
		saturation = (float64(active) / float64(max)) * 100.0
	}

	return DBPoolMetrics{
		ActiveConns:        active,
		IdleConns:          idle,
		MaxConns:           max,
		WaitCount:          waitCount,
		EmptyAcquireCount:  emptyAcquires,
		WaitDuration:       waitDur,
		WaitDurationMs:     float64(waitDur.Microseconds()) / 1000.0,
		SaturationPercent:  saturation,
		IsSaturated:        saturation >= 80.0,
		HasBlockedAcquires: emptyAcquires > 0,
	}
}
