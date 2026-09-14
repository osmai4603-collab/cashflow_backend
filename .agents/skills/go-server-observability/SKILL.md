---
name: go-server-observability
description: "Production-ready runtime observability, telemetry, and performance monitoring for Go HTTP services. Covers sliding-window latency engines (p50/p95/p99), database connection pool contention monitoring (empty_acquire_count), recent error circular buffers, dual exposition (OpenMetrics vs single-pass JSON snapshot), RUM ingestion, SLO alerting thresholds, high-performance CLI dashboards, and load-test baseline delta verification."
---

# Go Server Observability & Performance Skill

This skill defines production-grade telemetry, runtime observability, and performance monitoring standards
for Go HTTP services. It transforms services from "black boxes" into observable "glass boxes", providing
instantaneous visibility into real-time latency distributions, connection pool saturation, and operational health.

---

## Core Observability Principles

A production-ready observability architecture is characterized by:

1. **Instantaneous over Lifetime Averages**:
   Lifetime cumulative averages hide tail latency spikes and fail to recover after traffic bursts. Real-time operations require sliding-window quantiles (`p50`, `p95`, `p99`) that automatically return to baseline when idle.
2. **True Resource Contention Tracking**:
   Monotonically climbing counters (like total DB acquires) do not indicate failures. Monitoring must isolate true saturation events—such as **`empty_acquire_count`** and wait durations.
3. **Zero-Allocation Hot Path**:
   Collecting metrics inside the HTTP request/response hot path must impose near-zero CPU and memory overhead (`0 allocs/op`, `< 100 ns/op`).
4. **Dual Exposition Strategy**:
   - Standard **OpenMetrics / Prometheus** (`/metrics`) for centralized long-term scraping and time-series aggregation.
   - Low-latency **JSON Snapshot** (`/metrics/json`) for single-pass CLI dashboards and local diagnostic tools.
5. **Actionable SLO-Driven Alerts**:
   Clear thresholds for availability, latency, pool utilization, and heap growth with documented operational responses.
6. **Isolated Management Plane**:
   Telemetry and profiling endpoints reside on an isolated management listener (`:8066`), returning 404 on the public business port (`:8070`).

---

## Architecture: The Telemetry Plane

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                             HTTP Middlewares                             │
│     Latency Histogram ─── Status Classifier ─── Error Event Buffer       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      Metrics Registry & Engine                           │
│  ┌─────────────────────────┐              ┌───────────────────────────┐  │
│  │ 60s Sliding Window      │              │ In-Memory Recent Errors   │  │
│  │ (p50 / p95 / p99 / RPS) │              │ (Last N 4xx / 5xx Events) │  │
│  └─────────────────────────┘              └───────────────────────────┘  │
│  ┌─────────────────────────┐              ┌───────────────────────────┐  │
│  │ Database Pool Monitor   │              │ Go Runtime & Memory       │  │
│  │ (Active, Wait, Empty)   │              │ (Alloc, Heap, Goroutines) │  │
│  └─────────────────────────┘              └───────────────────────────┘  │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                 Management Exposition Router (:8066)                     │
│  GET /metrics       → Standard Prometheus OpenMetrics Text               │
│  GET /metrics/json  → Compact JSON Snapshot for CLI Tooling              │
│  POST /rum          → Core Web Vitals Real User Monitoring Ingestion     │
│  /debug/pprof/*     → On-demand CPU, Heap, and Goroutine Profiles        │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Section 1: Sliding Window Latency Engine

### Why Lifetime Averages Fail
Standard cumulative metrics (`total_duration / total_requests`) suffer from two critical flaws:
1. **Masking Spikes**: A 2-second latency spike across 100 requests becomes mathematically invisible if averaged over 1,000,000 previous requests.
2. **Permanent Distortion**: After a temporary degradation, lifetime average remains elevated for hours or days, preventing automated verification of recovery.

### The 60-Second Ring Buffer (`WindowTracker`)
- Time is sliced into 60 1-second rolling buckets.
- Each bucket records: request count, duration sum, and calibrated latency histograms.
- As time advances, buckets older than 60 seconds roll out automatically.
- When traffic ceases, percentiles cleanly return to `0.00 ms` with an `idle` state indicator.

### Zero-Allocation Hot Path Benchmark
```go
// Benchmark target: 0 allocs/op, < 100 ns/op
func (w *WindowTracker) Observe(d time.Duration) {
    bucketIdx := time.Now().Unix() % 60
    // Thread-safe atomic or striped bucket increment
}
```

---

## Section 2: Database Connection Pool Contention

In PostgreSQL connection pools (e.g. `pgxpool`), distinguish general traffic from true contention:

| Pool Metric | Diagnostic Meaning | Healthy State | Critical Alarm |
| :--- | :--- | :--- | :--- |
| `ActiveConns` | Connections currently executing queries | Fluctuates with RPS | N/A |
| `IdleConns` | Ready connections awaiting requests | > 0 during normal ops | 0 (All conns active) |
| `MaxConns` | Maximum configured pool limit | Static configuration | Static configuration |
| `WaitCount` | Cumulative count of total acquisitions | Monotonically climbs | Monotonically climbs |
| **`EmptyAcquireCount`** | Pool was 100% full; caller blocked | **0 or static** | **Climbing** (Pool starved) |
| **`WaitDuration`** | Cumulative time spent waiting for pool slot | Low / Constant | Exponentially increasing |

---

## Section 3: Live Error Diagnostics Buffer

Counters (`4xx: 12`, `5xx: 1`) indicate failures but lack context. Maintain a thread-safe circular ring buffer of the last N (e.g. 5–10) HTTP errors:

```go
type HTTPErrorEvent struct {
    Timestamp  time.Time `json:"timestamp"`
    Method     string    `json:"method"`
    Path       string    `json:"path"`
    Status     int       `json:"status"`
    DurationMs int64     `json:"duration_ms"`
}
```

- When `status >= 400`, capture event metadata into the ring buffer.
- Expose `recent_errors` inside `/metrics/json`.
- Display directly on the CLI dashboard:
  `Last Error: [500] POST /api/v1/payments (240ms) - 2s ago`.

---

## Section 4: Production SLO Thresholds & Alerting Rules

| Alert Rule | Condition | Operational Rationale & Action |
| :--- | :--- | :--- |
| `High5xxShare` | 5xx share > 0.5% for 5m | Exceeds 99.5% availability SLO. Check recent error buffer. |
| `HighLatencyP95` | p95 > 400ms (sliding window) | 20% margin below 500ms SLO. Inspect slow queries / pprof. |
| `DBPoolSaturated` | active / max conns > 80% | Pool near exhaustion. Review query durations or increase pool. |
| `DBPoolBlocked` | `empty_acquire_count` climbing | Callers are blocked waiting for DB slots. Immediate bottleneck. |
| `HeapGrowthAbnormal`| heap delta > 128 MB in 10m | Potential memory leak. Capture heap profile via `/debug/pprof`. |

---

## Section 5: High-Performance CLI Monitor Dashboard

### Architecture of `tool/monitor.sh`
- **Single-Pass Parsing**: Parse all JSON snapshot metrics in a single `jq -r '[...] | @tsv'` call to prevent spawning multiple processes per tick.
- **Dynamic RPS & Acquire Rates**: Compute differential rates per second from previous and current tick values.
- **Terminal Stability**: Use ANSI top-left repositioning (`\033[H`) and line clearing (`\033[K`) to prevent terminal flickering or ghosting.
- **Instant Diagnostics**: High-contrast display of recent errors and database pool health indicators.

---

## Section 6: Load Testing Baseline Delta Verification

When executing load tests (`tool/loadtest` or `tool/calibrate_baseline.sh`):
1. **Capture Baseline Snapshot**: Query `/metrics/json` before generating traffic to record pre-test counters.
2. **Generate Traffic**: Run load test scenarios.
3. **Compute Session Net Deltas**:
   $$\Delta \text{Requests} = \text{CurrentRequests} - \text{BaselineRequests}$$
   $$\Delta \text{Errors} = \text{CurrentErrors} - \text{BaselineErrors}$$
4. **Evaluate Session SLOs**: Verify percentiles and error rates strictly within the test window rather than the service's lifetime totals.

---

## Observability Verification Checklist

```text
[ ] Sliding window engine divides time into 60 1s buckets with automatic recovery to 0.00ms on idle
[ ] Zero-allocation verified: WindowTracker benchmarks show 0 allocs/op in hot path
[ ] DB pool observability captures active, idle, max, and critically empty_acquire_count
[ ] Recent errors buffer captures method, path, status, and duration for 4xx and 5xx
[ ] Management router exposes /metrics (OpenMetrics) and /metrics/json (compact snapshot)
[ ] Public port returns 404 for /metrics, /metrics/json, and /debug/pprof
[ ] CLI monitor uses single-pass jq parsing and renders without terminal flicker
[ ] Load testing tools sample baseline snapshots and assert against session deltas
```
