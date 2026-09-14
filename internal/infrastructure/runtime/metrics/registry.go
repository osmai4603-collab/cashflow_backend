package metrics

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Metric names exported by the registry. Everything is prefixed with
// "cashflow_" so moving to a dedicated exporter later does not clash with the
// default Go/process collectors.
const (
	MetricHTTPRequests        = "cashflow_http_requests_total"
	MetricHTTPRequestDuration = "cashflow_http_request_duration_seconds"
	MetricHTTPResponseSize    = "cashflow_http_response_size_bytes"
	MetricProbeDuration       = "cashflow_probe_duration_seconds"
	MetricUptimeSeconds       = "cashflow_process_uptime_seconds"
	MetricGoGoroutines        = "cashflow_go_goroutines"
	MetricGoGCSecondsTotal    = "cashflow_go_gc_seconds_total"
	MetricGoMemoryAlloc       = "cashflow_go_memory_alloc_bytes"
	MetricGoHeapInuse         = "cashflow_go_heap_inuse_bytes"
	MetricGoHeapIdle          = "cashflow_go_heap_idle_bytes"
	MetricDBMaxConns          = "cashflow_db_pool_max_connections"
	MetricDBActiveConns       = "cashflow_db_pool_active_connections"
	MetricDBIdleConns         = "cashflow_db_pool_idle_connections"
	MetricDBWaitCount         = "cashflow_db_pool_wait_count_total"
	MetricDBEmptyAcquireCount = "cashflow_db_pool_empty_acquire_count_total"
	MetricDBWaitDuration      = "cashflow_db_pool_wait_duration_seconds_total"
)

// Registry is the single source of truth for runtime, HTTP and database
// metrics. Both the JSON endpoint and (later) the Prometheus exporter read
// from the same underlying prometheus.Registry, so there are never two
// competing counters for the same event.
type Registry struct {
	raw *prometheus.Registry

	dbProvider atomic.Value // holds DBStatsProvider (nil-safe)
	serverURL  atomic.Value // holds string
	startTime  time.Time

	httpRequests  *prometheus.CounterVec
	httpDuration  *prometheus.HistogramVec
	httpBytes     *prometheus.HistogramVec
	probeDuration *prometheus.HistogramVec
	rum           map[string]*prometheus.HistogramVec

	window *WindowTracker
	probes *ProbeTracker

	errorsMu     sync.RWMutex
	recentErrors []HTTPErrorEvent
}

// NewRegistry builds an empty registry pre-wired with the cashflow metric
// families. All collectors are registered immediately; afterwards the
// registry may be gathered and updated concurrently.
func NewRegistry() *Registry {
	r := &Registry{
		raw:       prometheus.NewRegistry(),
		startTime: time.Now(),
		window:    NewWindowTracker(),
		probes:    NewProbeTracker(),
	}
	r.serverURL.Store("")

	r.httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: MetricHTTPRequests,
		Help: "Total number of HTTP requests processed, partitioned by method, route template and status class.",
	}, []string{LabelMethod, LabelRoute, LabelStatusClass})

	r.httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricHTTPRequestDuration,
		Help:    "HTTP request latency in seconds, partitioned by method and route template.",
		Buckets: LatencyBuckets,
	}, []string{LabelMethod, LabelRoute})

	r.httpBytes = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricHTTPResponseSize,
		Help:    "HTTP response size in bytes, partitioned by method and route template.",
		Buckets: ByteBuckets,
	}, []string{LabelMethod, LabelRoute})

	r.probeDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    MetricProbeDuration,
		Help:    "Health probe latency in seconds, measured outside business traffic.",
		Buckets: LatencyBuckets,
	}, []string{LabelProbe})

	r.rum = newRUMHistograms()

	start := r.startTime
	dbGauge := func(name, help string, pick func(DBStats) float64) prometheus.GaugeFunc {
		return prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: help}, func() float64 {
			p, ok := r.dbProvider.Load().(DBStatsProvider)
			if !ok || p == nil {
				return 0
			}
			return pick(p.Stats())
		})
	}
	memoryGauge := func(name, help string, pick func(*runtime.MemStats) uint64) prometheus.GaugeFunc {
		return prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: help}, func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(pick(&m))
		})
	}

	gcPauseSeconds := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: MetricGoGCSecondsTotal,
		Help: "Cumulative seconds spent in GC stop-the-world pauses since process start.",
	}, func() float64 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return float64(m.PauseTotalNs) / float64(time.Second)
	})

	collectors := []prometheus.Collector{
		r.httpRequests,
		r.httpDuration,
		r.httpBytes,
		r.probeDuration,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: MetricUptimeSeconds,
			Help: "Time in seconds since the process started.",
		}, func() float64 { return time.Since(start).Seconds() }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: MetricGoGoroutines,
			Help: "Current number of goroutines.",
		}, func() float64 { return float64(runtime.NumGoroutine()) }),
		gcPauseSeconds,
		memoryGauge(MetricGoMemoryAlloc, "Bytes allocated and still in use.", func(m *runtime.MemStats) uint64 { return m.Alloc }),
		memoryGauge(MetricGoHeapInuse, "Bytes in in-use spans.", func(m *runtime.MemStats) uint64 { return m.HeapInuse }),
		memoryGauge(MetricGoHeapIdle, "Bytes in idle (unused) spans.", func(m *runtime.MemStats) uint64 { return m.HeapIdle }),
		dbGauge(MetricDBMaxConns, "Maximum configured size of the database connection pool.", func(s DBStats) float64 { return float64(s.MaxConns) }),
		dbGauge(MetricDBActiveConns, "Number of active database connections.", func(s DBStats) float64 { return float64(s.ActiveConns) }),
		dbGauge(MetricDBIdleConns, "Number of idle database connections.", func(s DBStats) float64 { return float64(s.IdleConns) }),
		dbGauge(MetricDBWaitCount, "Total number of waits for a database connection.", func(s DBStats) float64 { return float64(s.WaitCount) }),
		dbGauge(MetricDBEmptyAcquireCount, "Total times the pool was exhausted and a caller had to wait.", func(s DBStats) float64 { return float64(s.EmptyAcquireCount) }),
		dbGauge(MetricDBWaitDuration, "Total time spent waiting for a database connection in seconds.", func(s DBStats) float64 { return s.WaitDuration.Seconds() }),
	}
	for _, vec := range r.rum {
		collectors = append(collectors, vec)
	}
	r.raw.MustRegister(collectors...)

	return r
}

// MustRegister registers additional collectors, delegating to the underlying
// prometheus registry. It panics on a naming conflict.
func (r *Registry) MustRegister(collectors ...prometheus.Collector) {
	r.raw.MustRegister(collectors...)
}

// RegisterDB stores the database stats provider. It is safe to call after the
// server has started; reads are atomic.
func (r *Registry) RegisterDB(provider DBStatsProvider) {
	r.dbProvider.Store(provider)
}

// RegisterServerURL stores the advertised base URL. It is safe to call after
// the server has started; reads are atomic.
func (r *Registry) RegisterServerURL(url string) {
	r.serverURL.Store(url)
}

func (r *Registry) serverURLValue() string {
	s, _ := r.serverURL.Load().(string)
	return s
}

// ObserveHTTP records one completed request. route must already be normalised
// to the closed label contract; bytes is the number of body bytes written.
func (r *Registry) ObserveHTTP(method, route, statusClass string, duration time.Duration, bytes int64) {
	r.httpRequests.WithLabelValues(method, route, statusClass).Inc()
	r.httpDuration.WithLabelValues(method, route).Observe(duration.Seconds())
	r.httpBytes.WithLabelValues(method, route).Observe(float64(bytes))
	if r.window != nil {
		r.window.Observe(duration, bytes)
	}
}

// ObserveProbe records the latency of a health probe (livez/readyz) outside
// the business traffic histogram.
func (r *Registry) ObserveProbe(probe string, duration time.Duration) {
	r.probeDuration.WithLabelValues(probe).Observe(duration.Seconds())
	if r.probes != nil {
		r.probes.Observe(probe, duration)
	}
}

// Gather returns the current metrics in the OpenMetrics DTO shape.
func (r *Registry) Gather() ([]*dto.MetricFamily, error) {
	return r.raw.Gather()
}

type histBucket struct {
	upperBound float64
	count      float64
}

func calculateQuantile(q float64, totalCount float64, buckets []histBucket) float64 {
	if totalCount <= 0 || len(buckets) == 0 {
		return 0
	}
	rank := q * totalCount
	var prevBound float64
	var prevCount float64

	for _, b := range buckets {
		if rank <= b.count {
			countDiff := b.count - prevCount
			if countDiff <= 0 {
				return b.upperBound
			}
			fraction := (rank - prevCount) / countDiff
			return prevBound + fraction*(b.upperBound-prevBound)
		}
		prevBound = b.upperBound
		prevCount = b.count
	}
	return buckets[len(buckets)-1].upperBound
}

// ExtendedStats gathers and computes detailed traffic statistics including
// status classes, latency percentiles, average response size, probe latencies,
// and RUM metrics.
func (r *Registry) ExtendedStats() ExtendedStats {
	stats := ExtendedStats{
		ByClass: map[string]uint64{
			Status2xx: 0,
			Status3xx: 0,
			Status4xx: 0,
			Status5xx: 0,
		},
	}

	families, err := r.raw.Gather()
	if err != nil {
		return stats
	}

	var latSum, latCount float64
	latBucketMap := make(map[float64]float64)

	var bytesSum, bytesCount float64

	var readyzSum, readyzCount float64
	var livezSum, livezCount float64

	var rumCount uint64
	var ttfbSum, ttfbCount float64
	var lcpSum, lcpCount float64
	var inpSum, inpCount float64
	var clsSum, clsCount float64
	var domSum, domCount float64

	for _, f := range families {
		switch f.GetName() {
		case MetricHTTPRequests:
			for _, m := range f.GetMetric() {
				v := uint64(m.GetCounter().GetValue())
				stats.Total += v
				var cls string
				for _, lp := range m.GetLabel() {
					if lp.GetName() == LabelStatusClass {
						cls = lp.GetValue()
					}
				}
				if cls == "" {
					cls = Status2xx
				}
				stats.ByClass[cls] += v
			}
		case MetricHTTPRequestDuration:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				latSum += h.GetSampleSum()
				latCount += float64(h.GetSampleCount())
				for _, b := range h.GetBucket() {
					if !math.IsInf(b.GetUpperBound(), 0) {
						latBucketMap[b.GetUpperBound()] += float64(b.GetCumulativeCount())
					}
				}
			}
		case MetricHTTPResponseSize:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				bytesSum += h.GetSampleSum()
				bytesCount += float64(h.GetSampleCount())
			}
		case MetricProbeDuration:
			for _, m := range f.GetMetric() {
				var probe string
				for _, lp := range m.GetLabel() {
					if lp.GetName() == LabelProbe {
						probe = lp.GetValue()
					}
				}
				h := m.GetHistogram()
				cnt := float64(h.GetSampleCount())
				if probe == ProbeReadyz {
					readyzSum += h.GetSampleSum()
					readyzCount += cnt
				} else if probe == ProbeLivez {
					livezSum += h.GetSampleSum()
					livezCount += cnt
				}
			}
		case MetricRUMTTFB:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				ttfbSum += h.GetSampleSum()
				ttfbCount += float64(h.GetSampleCount())
				rumCount += h.GetSampleCount()
			}
		case MetricRUMLCP:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				lcpSum += h.GetSampleSum()
				lcpCount += float64(h.GetSampleCount())
				rumCount += h.GetSampleCount()
			}
		case MetricRUMINP:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				inpSum += h.GetSampleSum()
				inpCount += float64(h.GetSampleCount())
				rumCount += h.GetSampleCount()
			}
		case MetricRUMCLS:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				clsSum += h.GetSampleSum()
				clsCount += float64(h.GetSampleCount())
				rumCount += h.GetSampleCount()
			}
		case MetricRUMDOMInteractive:
			for _, m := range f.GetMetric() {
				h := m.GetHistogram()
				domSum += h.GetSampleSum()
				domCount += float64(h.GetSampleCount())
				rumCount += h.GetSampleCount()
			}
		}
	}

	if latCount > 0 {
		stats.LifetimeAvgMs = latSum / latCount * 1000

		var sortedBuckets []histBucket
		for bound, count := range latBucketMap {
			sortedBuckets = append(sortedBuckets, histBucket{upperBound: bound, count: count})
		}
		sort.Slice(sortedBuckets, func(i, j int) bool {
			return sortedBuckets[i].upperBound < sortedBuckets[j].upperBound
		})

		stats.LifetimeP95Ms = calculateQuantile(0.95, latCount, sortedBuckets) * 1000
	}

	stats.WindowSeconds = DefaultWindowSeconds
	if r.window != nil {
		win := r.window.Snapshot()
		stats.AvgMs = win.AvgMs
		stats.P50Ms = win.P50Ms
		stats.P95Ms = win.P95Ms
		stats.P99Ms = win.P99Ms
		stats.AvgResponseBytes = win.AvgResponseBytes
	} else {
		if latCount > 0 {
			stats.AvgMs = stats.LifetimeAvgMs
			stats.P95Ms = stats.LifetimeP95Ms
		}
		if bytesCount > 0 {
			stats.AvgResponseBytes = bytesSum / bytesCount
		}
	}

	if r.probes != nil {
		stats.ReadyzMs, stats.LivezMs = r.probes.LatenciesMs()
	}
	if stats.ReadyzMs == 0 && readyzCount > 0 {
		stats.ReadyzMs = readyzSum / readyzCount * 1000
	}
	if stats.LivezMs == 0 && livezCount > 0 {
		stats.LivezMs = livezSum / livezCount * 1000
	}

	if rumCount > 0 {
		rum := &RUMSummary{
			SamplesCount: rumCount,
		}
		if ttfbCount > 0 {
			rum.AvgTTFBMs = ttfbSum / ttfbCount * 1000
		}
		if lcpCount > 0 {
			rum.AvgLCPMs = lcpSum / lcpCount * 1000
		}
		if inpCount > 0 {
			rum.AvgINPMs = inpSum / inpCount * 1000
		}
		if clsCount > 0 {
			rum.AvgCLS = clsSum / clsCount
		}
		if domCount > 0 {
			rum.AvgDOMIntMs = domSum / domCount * 1000
		}
		stats.RUM = rum
	}

	stats.RecentErrors = r.RecentErrors()

	return stats
}

// HTTPTotals aggregates the labelled HTTP families back into the legacy
// flat counters (total, per status class, average latency in ms).
func (r *Registry) HTTPTotals() (total uint64, byClass map[string]uint64, avgMs float64) {
	s := r.ExtendedStats()
	return s.Total, s.ByClass, s.AvgMs
}


// uptimeSeconds mirrors the uptime gauge for the JSON endpoint without
// triggering an extra gather.
func (r *Registry) uptimeSeconds() float64 {
	return time.Since(r.startTime).Seconds()
}

const maxRecentErrors = 5

// RecordHTTPError records a single 4xx or 5xx HTTP error in the ring buffer.
func (r *Registry) RecordHTTPError(method, path string, status int, duration time.Duration) {
	r.errorsMu.Lock()
	defer r.errorsMu.Unlock()

	ev := HTTPErrorEvent{
		Timestamp:  time.Now(),
		Method:     method,
		Path:       path,
		Status:     status,
		DurationMs: duration.Milliseconds(),
	}

	if len(r.recentErrors) >= maxRecentErrors {
		copy(r.recentErrors, r.recentErrors[1:])
		r.recentErrors[len(r.recentErrors)-1] = ev
	} else {
		r.recentErrors = append(r.recentErrors, ev)
	}
}

// RecentErrors returns a snapshot of the most recent HTTP error events.
func (r *Registry) RecentErrors() []HTTPErrorEvent {
	r.errorsMu.RLock()
	defer r.errorsMu.RUnlock()

	if len(r.recentErrors) == 0 {
		return nil
	}
	out := make([]HTTPErrorEvent, len(r.recentErrors))
	copy(out, r.recentErrors)
	return out
}

