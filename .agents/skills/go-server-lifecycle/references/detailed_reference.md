# Go Server Lifecycle — Detailed Reference

This document provides in-depth explanations of lifecycle phases, professional logging, and runtime observability.

---

## 1. Professional Structured Logging

In production, logs must be machine-readable (JSON) for aggregators like ELK or Datadog. However, for local development, humans need readable, colored output.

### TTY Detection & Dual-Mode
Use `github.com/mattn/go-isatty` to detect if the output is a terminal.

- **TTY Mode**: Use a custom `slog.Handler` that formats output with ANSI colors.
- **JSON Mode**: Use `slog.JSONHandler` with a custom `ReplaceAttr` to enforce consistent time formatting (e.g., `2006-01-02 03:04:05 PM`).

---

## 2. Runtime Observability & Metrics

A production-ready server is a "glass box". You must be able to see inside without restarting it.

### The `/metrics` Endpoint
Expose vital signals in a structured format (JSON):
- **Memory**: Allocated bytes, Heap status, System memory.
- **Runtime**: Number of goroutines, GC cycles, Uptime.
- **Traffic**: Success rate (`2xx`), Client errors (`4xx`), Server errors (`5xx`), and Average Latency (ms).

### Database Pool Monitoring
Exporting connection pool stats (like `pgxpool.Stat()`) is critical for detecting:
- **Connection Leaks**: Active connections growing indefinitely.
- **Pool Saturation**: High wait counts for new connections.

---

## 3. Log Suppression for Probes

Observability probes (`/metrics`, `/livez`) run frequently (e.g., every 5 seconds). Logging every success creates massive "noise".

**Best Practice**:
- Do NOT log healthy (Status < 400) hits to observability endpoints.
- DO log errors (Status >= 400) to ensure outages are visible.

---

## 4. The 7-Phase Operational Flow (Refresher)

| Phase | Action | Purpose |
|:---|:---|:---|
| **1. Init** | Config, Logging, Deps | Fail-Fast if broken |
| **2. Config** | Router, Timeouts, Metrics | Prepare the engine |
| **3. Startup** | Non-blocking Listen | Start the listener |
| **4. Serving** | Handle requests | Main operational state |
| **5. Drain** | Ready = false | Notify Load Balancer |
| **6. Shutdown** | server.Shutdown | Wait for in-flight work |
| **7. Cleanup** | Close resources | Safe reverse-order exit |

---

## 5. The "go run" Signal Trap

Never use `go run` in production or Makefiles. It wraps the process and swallows exit signals, making graceful shutdown impossible to verify from the outside. Always build the binary and execute it directly.
