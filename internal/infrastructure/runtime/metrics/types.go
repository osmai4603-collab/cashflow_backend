package metrics

// Locked label keys for HTTP traffic metrics. These are the only label
// dimensions allowed; route cardinality is kept bounded by normalising to
// route templates (see RoutePattern) in P2.
const (
	LabelMethod      = "method"
	LabelRoute       = "route"
	LabelStatusClass = "status_class"
	LabelProbe       = "probe"
	LabelClientType  = "client_type"
)

// Closed set of status classes (2xx/3xx/4xx/5xx). 1xx falls back to 2xx.
const (
	Status2xx = "2xx"
	Status3xx = "3xx"
	Status4xx = "4xx"
	Status5xx = "5xx"

	// UnknownRoute is used when a request could not be matched to a route template.
	UnknownRoute = "unknown"
)

// Closed set of probe names whose latency is measured outside business traffic.
const (
	ProbeLivez  = "livez"
	ProbeReadyz = "readyz"
)

// Closed set of RUM client types. client_type is the only dimension on the
// RUM families; the set is bounded so label cardinality cannot grow and the
// handler rejects anything outside it (P7).
const (
	ClientTypeWeb     = "web"
	ClientTypeDesktop = "desktop"
	ClientTypeMobile  = "mobile"
)

// ValidClientType reports whether s is a member of the closed client set.
func ValidClientType(s string) bool {
	switch s {
	case ClientTypeWeb, ClientTypeDesktop, ClientTypeMobile:
		return true
	default:
		return false
	}
}

// LatencyBuckets defines the fixed histogram buckets (in seconds) for HTTP
// request latency. The +Inf bucket is added implicitly by Prometheus.
var LatencyBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// ByteBuckets defines the histogram buckets (in bytes) for HTTP response sizes.
var ByteBuckets = []float64{
	64, 256, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304,
}

// CLSBuckets defines the buckets for the cumulative layout shift score, a
// dimensionless number in [0, 1] for typical sessions. The +Inf bucket is
// added implicitly.
var CLSBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1,
}

// statusClassFor maps an HTTP status code onto the closed label set.
func statusClassFor(code int) string {
	switch {
	case code >= 500:
		return Status5xx
	case code >= 400:
		return Status4xx
	case code >= 300:
		return Status3xx
	default:
		return Status2xx
	}
}

// DBStatsProvider allows different storage backends to report their pool health.
type DBStatsProvider interface {
	Stats() DBStats
}

type DBStats struct {
	MaxConns    int32 `json:"max_conns"`
	ActiveConns int32 `json:"active_conns"`
	IdleConns   int32 `json:"idle_conns"`
	WaitCount   int64 `json:"wait_count"`
}
