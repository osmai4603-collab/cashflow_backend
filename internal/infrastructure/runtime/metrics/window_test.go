package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestWindowTracker_ZeroValuesWhenEmpty(t *testing.T) {
	w := NewWindowTracker()
	snap := w.Snapshot()

	if snap.Count != 0 {
		t.Errorf("expected 0 count, got %d", snap.Count)
	}
	if snap.AvgMs != 0 {
		t.Errorf("expected 0 avg, got %v", snap.AvgMs)
	}
	if snap.P50Ms != 0 || snap.P95Ms != 0 || snap.P99Ms != 0 {
		t.Errorf("expected 0 percentiles, got p50=%v p95=%v p99=%v", snap.P50Ms, snap.P95Ms, snap.P99Ms)
	}
	if snap.AvgResponseBytes != 0 {
		t.Errorf("expected 0 avg bytes, got %v", snap.AvgResponseBytes)
	}
}

func TestWindowTracker_ObserveAndQuantiles(t *testing.T) {
	baseTime := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	currentTime := baseTime

	w := NewWindowTracker()
	w.nowFn = func() time.Time { return currentTime }

	// Record 90 requests of 10ms (0.010s) and 10 requests of 200ms (0.200s)
	for i := 0; i < 90; i++ {
		w.Observe(10*time.Millisecond, 500)
	}
	for i := 0; i < 10; i++ {
		w.Observe(200*time.Millisecond, 2000)
	}

	snap := w.Snapshot()
	if snap.Count != 100 {
		t.Fatalf("expected 100 requests, got %d", snap.Count)
	}

	// Average: (90 * 10 + 10 * 200) / 100 = (900 + 2000) / 100 = 29.0 ms
	if snap.AvgMs < 28.0 || snap.AvgMs > 30.0 {
		t.Errorf("expected AvgMs around 29.0, got %v", snap.AvgMs)
	}

	// P50 should be around 10ms
	if snap.P50Ms < 5.0 || snap.P50Ms > 25.0 {
		t.Errorf("expected P50Ms in 5-25ms range, got %v", snap.P50Ms)
	}

	// P95 / P99 should reflect the 200ms requests
	if snap.P95Ms < 50.0 {
		t.Errorf("expected P95Ms >= 50ms, got %v", snap.P95Ms)
	}
	if snap.P99Ms < 100.0 {
		t.Errorf("expected P99Ms >= 100ms, got %v", snap.P99Ms)
	}
}

func TestWindowTracker_WindowExpirationAndBaselineRecovery(t *testing.T) {
	baseTime := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	currentTime := baseTime

	w := NewWindowTracker()
	w.nowFn = func() time.Time { return currentTime }

	// T = 0: High load of 500ms requests
	for i := 0; i < 50; i++ {
		w.Observe(500*time.Millisecond, 1024)
	}

	snap := w.Snapshot()
	if snap.Count != 50 || snap.AvgMs < 400 {
		t.Fatalf("expected 50 requests with ~500ms avg, got count=%d avg=%v", snap.Count, snap.AvgMs)
	}

	// Advance time by 30 seconds (still within window)
	currentTime = currentTime.Add(30 * time.Second)
	snap30 := w.Snapshot()
	if snap30.Count != 50 {
		t.Fatalf("expected count=50 at T=30s, got %d", snap30.Count)
	}

	// Advance time past the 60s window (T = 65s) -> idle baseline!
	currentTime = baseTime.Add(65 * time.Second)
	snapExpired := w.Snapshot()
	if snapExpired.Count != 0 {
		t.Errorf("expected 0 count after window expiration, got %d", snapExpired.Count)
	}
	if snapExpired.AvgMs != 0 || snapExpired.P50Ms != 0 || snapExpired.P95Ms != 0 || snapExpired.P99Ms != 0 {
		t.Errorf("expected all metrics to return to baseline 0, got avg=%v p50=%v p95=%v p99=%v",
			snapExpired.AvgMs, snapExpired.P50Ms, snapExpired.P95Ms, snapExpired.P99Ms)
	}

	// Now record new fast requests at T = 70s
	currentTime = baseTime.Add(70 * time.Second)
	for i := 0; i < 20; i++ {
		w.Observe(2*time.Millisecond, 100)
	}

	snapNew := w.Snapshot()
	if snapNew.Count != 20 {
		t.Fatalf("expected 20 requests, got %d", snapNew.Count)
	}
	if snapNew.AvgMs > 10.0 {
		t.Errorf("expected new fast avg < 10ms, got %v (old 500ms requests should not pollute!)", snapNew.AvgMs)
	}
}

func TestWindowTracker_ConcurrentSafety(t *testing.T) {
	w := NewWindowTracker()
	const workers = 10
	const perWorker = 1000

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				w.Observe(time.Duration(j%50)*time.Millisecond, int64(j*10))
				if j%100 == 0 {
					_ = w.Snapshot()
				}
			}
		}(i)
	}
	wg.Wait()

	snap := w.Snapshot()
	want := uint64(workers * perWorker)
	if snap.Count != want {
		t.Errorf("expected %d total count, got %d", want, snap.Count)
	}
}

func TestProbeTracker_ObservesLatest(t *testing.T) {
	p := NewProbeTracker()

	r, l := p.LatenciesMs()
	if r != 0 || l != 0 {
		t.Errorf("expected zero latencies, got readyz=%v livez=%v", r, l)
	}

	p.Observe(ProbeReadyz, 15*time.Millisecond)
	p.Observe(ProbeLivez, 3*time.Millisecond)

	r, l = p.LatenciesMs()
	if r != 15 || l != 3 {
		t.Errorf("expected readyz=15ms livez=3ms, got readyz=%v livez=%v", r, l)
	}

	// Update with new values
	p.Observe(ProbeReadyz, 2*time.Millisecond)
	r, _ = p.LatenciesMs()
	if r != 2 {
		t.Errorf("expected updated readyz=2ms, got %v", r)
	}
}

func BenchmarkWindowTracker_Observe(b *testing.B) {
	w := NewWindowTracker()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Observe(15*time.Millisecond, 512)
	}
}
