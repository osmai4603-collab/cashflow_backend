# مواصفات مخطط المقاييس التشغيلية (Metrics Specification)

توثق هذه الصفحة المخطط البرمجي الكامل للبيانات التي يتم تصديرها عبر مساري المراقبة:

1. `GET /metrics/json` (اللقطة الحظية بصيغة JSON)
2. `GET /metrics` (معيار OpenMetrics / Prometheus)

---

## 1. مواصفات لقطة JSON (`GET /metrics/json`)

### مخطط البيانات (JSON Schema Example)

```json
{
  "uptime_seconds": 1433.83,
  "timestamp": "2026-09-14T05:31:01Z",
  "traffic": {
    "total_requests": 25410,
    "total_2xx": 25380,
    "total_4xx": 28,
    "total_5xx": 2,
    "window_rps": 142.5,
    "window_avg_ms": 12.4,
    "p50_ms": 8.0,
    "p95_ms": 45.0,
    "p99_ms": 120.0,
    "is_idle": false
  },
  "db": {
    "active_conns": 4,
    "idle_conns": 16,
    "max_conns": 25,
    "wait_count": 89400,
    "empty_acquire_count": 0,
    "wait_duration_ms": 14.2,
    "saturation_percent": 16.0,
    "is_saturated": false,
    "has_blocked_acquires": false
  },
  "runtime": {
    "alloc_mb": 42.15,
    "heap_sys_mb": 64.0,
    "num_goroutines": 38,
    "num_gc_cycles": 112
  },
  "recent_errors": [
    {
      "timestamp": "2026-09-14T05:30:15Z",
      "method": "GET",
      "path": "/api/v1/unknown",
      "status": 404,
      "duration_ms": 2
    }
  ]
}
```

---

## 2. مواصفات معيار OpenMetrics (`GET /metrics`)

```text
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{status="2xx"} 25380
http_requests_total{status="4xx"} 28
http_requests_total{status="5xx"} 2

# HELP db_connections Active and idle database connections
# TYPE db_connections gauge
db_connections{state="active"} 4
db_connections{state="idle"} 16
db_connections{state="max"} 25

# HELP db_empty_acquire_total Count of times callers blocked waiting for DB pool slot
# TYPE db_empty_acquire_total counter
db_empty_acquire_total 0
```
