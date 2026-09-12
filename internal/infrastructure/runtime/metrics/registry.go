package metrics

import (
	"runtime"
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
}

// NewRegistry builds an empty registry pre-wired with the cashflow metric
// families. All collectors are registered immediately; afterwards the
// registry may be gathered and updated concurrently.
func NewRegistry() *Registry {
	r := &Registry{
		raw:       prometheus.NewRegistry(),
		startTime: time.Now(),
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
}

// ObserveProbe records the latency of a health probe (livez/readyz) outside
// the business traffic histogram.
func (r *Registry) ObserveProbe(probe string, duration time.Duration) {
	r.probeDuration.WithLabelValues(probe).Observe(duration.Seconds())
}

// Gather returns the current metrics in the OpenMetrics DTO shape.
func (r *Registry) Gather() ([]*dto.MetricFamily, error) {
	return r.raw.Gather()
}

// HTTPTotals aggregates the labelled HTTP families back into the legacy
// flat counters (total, per status class, average latency in ms).
func (r *Registry) HTTPTotals() (total uint64, byClass map[string]uint64, avgMs float64) {
	byClass = map[string]uint64{
		Status2xx: 0,
		Status3xx: 0,
		Status4xx: 0,
		Status5xx: 0,
	}

	var sum, count float64
	families, err := r.raw.Gather()
	if err != nil {
		return 0, byClass, 0
	}
	for _, f := range families {
		switch f.GetName() {
		case MetricHTTPRequests:
			for _, m := range f.GetMetric() {
				v := uint64(m.GetCounter().GetValue())
				total += v
				var cls string
				for _, lp := range m.GetLabel() {
					if lp.GetName() == LabelStatusClass {
						cls = lp.GetValue()
					}
				}
				if cls == "" {
					cls = Status2xx
				}
				byClass[cls] += v
			}
		case MetricHTTPRequestDuration:
			for _, m := range f.GetMetric() {
				sum += m.GetHistogram().GetSampleSum()
				count += float64(m.GetHistogram().GetSampleCount())
			}
		}
	}
	if count > 0 {
		avgMs = sum / count * 1000
	}
	return total, byClass, avgMs
}

// uptimeSeconds mirrors the uptime gauge for the JSON endpoint without
// triggering an extra gather.
func (r *Registry) uptimeSeconds() float64 {
	return time.Since(r.startTime).Seconds()
}
