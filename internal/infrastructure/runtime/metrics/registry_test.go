package metrics

import (
	"math"
	"sync"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

func TestRegistry_ZeroValues(t *testing.T) {
	reg := NewRegistry()

	total, byClass, avgMs := reg.HTTPTotals()
	if total != 0 {
		t.Errorf("expected zero requests, got %d", total)
	}
	for _, cls := range []string{Status2xx, Status3xx, Status4xx, Status5xx} {
		if byClass[cls] != 0 {
			t.Errorf("expected zero %s requests, got %d", cls, byClass[cls])
		}
	}
	if avgMs != 0 {
		t.Errorf("expected zero average latency, got %v", avgMs)
	}

	// Fresh registry middleware/JSON must still emit the full empty DB block.
	if got := reg.serverURLValue(); got != "" {
		t.Errorf("expected empty server url, got %q", got)
	}
}

func TestRegistry_LatencyBuckets(t *testing.T) {
	reg := NewRegistry()
	reg.ObserveHTTP("GET", "/api/v1/partners/{id}", Status2xx, 50*time.Millisecond, 512)

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	found := false
	for _, f := range families {
		if f.GetName() != MetricHTTPRequestDuration {
			continue
		}
		found = true
		m := f.GetMetric()[0]
		buckets := m.GetHistogram().GetBucket()
		// client_golang emits one bucket per configured bound; observations
		// beyond the last bound are captured by SampleCount (the +Inf bucket).
		if len(buckets) != len(LatencyBuckets) {
			t.Fatalf("expected %d buckets, got %d", len(LatencyBuckets), len(buckets))
		}
		for i, want := range LatencyBuckets {
			if got := buckets[i].GetUpperBound(); math.Abs(got-want) > 1e-12 {
				t.Errorf("bucket %d: expected upper bound %v, got %v", i, want, got)
			}
		}
		if got := m.GetHistogram().GetSampleCount(); got != 1 {
			t.Errorf("expected 1 sample, got %d", got)
		}
		if got := buckets[len(buckets)-1].GetCumulativeCount(); got != 1 {
			t.Errorf("expected last bucket cumulative count 1, got %d", got)
		}
	}
	if !found {
		t.Fatalf("histogram family %s not found", MetricHTTPRequestDuration)
	}
}

func TestRegistry_ConcurrentCounting(t *testing.T) {
	reg := NewRegistry()
	const workers, perWorker = 8, 1000

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				reg.ObserveHTTP("GET", "/api/v1/partners/{id}", Status2xx, time.Millisecond, 1024)
			}
		}()
	}
	wg.Wait()

	total, byClass, _ := reg.HTTPTotals()
	want := uint64(workers * perWorker)
	if total != want {
		t.Errorf("expected %d total requests, got %d", want, total)
	}
	if byClass[Status2xx] != want {
		t.Errorf("expected %d 2xx requests, got %d", want, byClass[Status2xx])
	}
}

// TestRegistry_ConcurrentRegisterAndGather exercises the atomic dbProvider /
// serverURL store under the race detector while the registry is gathered.
func TestRegistry_ConcurrentRegisterAndGather(t *testing.T) {
	reg := NewRegistry()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				reg.RegisterServerURL("http://cashflow.test:8080")
				reg.RegisterDB(&fakeDB{})
				reg.ObserveHTTP("POST", "/api/v1/transactions", Status4xx, time.Microsecond, 64)
			}
		}()
	}

	for i := 0; i < 500; i++ {
		if _, err := reg.Gather(); err != nil {
			t.Fatalf("gather: %v", err)
		}
		reg.HTTPTotals()
	}

	close(stop)
	wg.Wait()
}

func TestRegistry_DBGaugesReflectProvider(t *testing.T) {
	reg := NewRegistry()

	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if v := gaugeValue(families, MetricDBMaxConns); v != 0 {
		t.Errorf("expected 0 max conns without provider, got %v", v)
	}

	reg.RegisterDB(&fakeDB{})
	families, err = reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if v := gaugeValue(families, MetricDBActiveConns); v != 5 {
		t.Errorf("expected 5 active conns, got %v", v)
	}
	if v := gaugeValue(families, MetricDBEmptyAcquireCount); v != 2 {
		t.Errorf("expected 2 empty acquire count, got %v", v)
	}
	if v := gaugeValue(families, MetricDBWaitDuration); v != 2.5 {
		t.Errorf("expected 2.5s wait duration, got %v", v)
	}
}

func gaugeValue(families []*dto.MetricFamily, name string) float64 {
	for _, f := range families {
		if f.GetName() == name {
			for _, m := range f.GetMetric() {
				return m.GetGauge().GetValue()
			}
		}
	}
	return math.NaN()
}

type fakeDB struct{}

func (f *fakeDB) Stats() DBStats {
	return DBStats{MaxConns: 10, ActiveConns: 5, IdleConns: 3, WaitCount: 12, EmptyAcquireCount: 2, WaitDuration: 2500 * time.Millisecond}
}

func TestRegistry_RecentErrorsRingBuffer(t *testing.T) {
	reg := NewRegistry()

	// Initial state must be empty
	if errs := reg.RecentErrors(); len(errs) != 0 {
		t.Fatalf("expected nil or empty recent errors, got %d", len(errs))
	}

	// Record 7 errors to verify ring buffer caps at maxRecentErrors (5)
	for i := 1; i <= 7; i++ {
		reg.RecordHTTPError("GET", "/test/error", 400+i, time.Duration(i)*time.Millisecond)
	}

	errs := reg.RecentErrors()
	if len(errs) != 5 {
		t.Fatalf("expected exactly 5 errors in buffer, got %d", len(errs))
	}

	// First error should be the 3rd one (status 403), last should be 7th (status 407)
	if errs[0].Status != 403 {
		t.Errorf("expected first error status 403, got %d", errs[0].Status)
	}
	if errs[4].Status != 407 {
		t.Errorf("expected last error status 407, got %d", errs[4].Status)
	}
}

