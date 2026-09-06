---
name: go-server-lifecycle
description: >-
  Manages the complete lifecycle of an HTTP server in Go, from initialization
  to graceful shutdown. Use this skill when creating a new Go HTTP server,
  adding graceful shutdown, configuring server timeouts, implementing health
  checks (liveness/readiness), handling OS signals, cleaning up resources,
  or reviewing an existing server for production readiness. Covers all 7
  lifecycle phases: Initialization, Configuration, Startup, Serving,
  Drain, Graceful Shutdown, and Cleanup.
---

# Go Server Lifecycle Management Skill

This skill provides a complete, standards-based approach to managing the
lifecycle of an HTTP server in Go. All patterns are derived from the official
Go standard library documentation.

## Official Sources

- [`net/http.Server`](https://pkg.go.dev/net/http#Server)
- [`Server.Shutdown`](https://pkg.go.dev/net/http#Server.Shutdown)
- [`os/signal`](https://pkg.go.dev/os/signal)
- [`context`](https://pkg.go.dev/context)
- [`sync.WaitGroup`](https://pkg.go.dev/sync#WaitGroup)

---

## Lifecycle Overview (7 Phases)

```
Phase 1: Initialization  → Load config, create dependencies
Phase 2: Configuration   → Set server timeouts and TLS
Phase 3: Startup         → Non-blocking ListenAndServe in goroutine
Phase 4: Serving         → Health checks: /livez, /readyz
Phase 5: Drain           → Mark not-ready, wait for LB removal
Phase 6: Graceful Shutdown → server.Shutdown(ctx) with timeout
Phase 7: Cleanup         → Close DB, workers, files (reverse order)
```

---

## Step-by-Step Instructions

### Step 1: Initialize Dependencies

Before starting the server, initialize all dependencies that the server
needs. This includes database connections, caches, message queues, etc.

- Create dependencies in order (config → logger → DB → cache → services)
- Store them in a struct or pass via dependency injection
- If initialization fails, **exit immediately** — do not start the server

Reference: [examples/01_initialization.go](./examples/01_initialization.go)

### Step 2: Configure the Server with Timeouts

**CRITICAL**: Never use a zero-value `http.Server`. Always set timeouts.

| Field                 | Recommended Value | Purpose                          |
|:----------------------|:------------------|:---------------------------------|
| `ReadTimeout`         | 5s                | Max time to read full request    |
| `ReadHeaderTimeout`   | 2s                | Slowloris attack protection      |
| `WriteTimeout`        | 10s               | Max time to write response       |
| `IdleTimeout`         | 120s              | Keep-alive wait for next request |

> **WARNING**: These timeouts control **network I/O** only. They do NOT stop
> handler logic. Use `context.WithTimeout` or `http.TimeoutHandler` for
> handler-level timeouts.

Reference: [examples/02_configuration.go](./examples/02_configuration.go)

### Step 3: Start the Server (Non-blocking)

**CRITICAL**: Run `ListenAndServe()` in a **separate goroutine** so that
`main()` remains free to handle OS signals.

Rules:
- Always check the returned error
- Filter out `http.ErrServerClosed` — it is **expected** when `Shutdown()` is called
- Use `errors.Is()` for comparison, not `==`
- Log startup confirmation after the goroutine is launched

Reference: [examples/03_startup.go](./examples/03_startup.go)

### Step 4: Implement Health Check Endpoints

Register **separate** endpoints for each health probe:

| Probe       | Path        | What to Check                    | What NOT to Check       |
|:------------|:------------|:---------------------------------|:------------------------|
| **Liveness**  | `/livez`  | Process is alive (return 200)    | External dependencies   |
| **Readiness** | `/readyz` | Critical dependencies (DB, etc.) | Non-critical services   |
| **Startup**   | `/startupz` | Initialization complete         | —                       |

> **DANGER**: A liveness probe that checks external databases can cause
> **cascading failures** — all pods restart due to a temporary DB flicker.

Reference: [examples/04_health_checks.go](./examples/04_health_checks.go)

### Step 5: Handle OS Signals

Listen for termination signals using `signal.NotifyContext` (Go 1.16+):

Required signals:
- `os.Interrupt` (`SIGINT`) — user presses Ctrl+C
- `syscall.SIGTERM` — sent by Kubernetes, systemd, Docker

Reference: [examples/05_signal_handling.go](./examples/05_signal_handling.go)

### Step 6: Implement Drain Phase (Production)

When running behind a load balancer:

1. Receive shutdown signal
2. Mark the app as "not ready" (return 503 on `/readyz`)
3. Wait briefly (e.g., 5 seconds) for the load balancer to stop routing
4. Proceed to shutdown

This prevents request drops during rolling deployments.

Reference: [examples/06_drain_phase.go](./examples/06_drain_phase.go)

### Step 7: Graceful Shutdown with Timeout

Use `server.Shutdown(ctx)` — **NOT** `server.Close()`.

| Method     | Behavior                              |
|:-----------|:--------------------------------------|
| `Shutdown` | Waits for in-flight requests to finish |
| `Close`    | Immediately kills all connections      |

Rules:
- Always pass a `context.WithTimeout` (5–30 seconds)
- Handle hijacked connections (WebSockets) separately
- `Shutdown` does NOT close hijacked connections

Reference: [examples/07_graceful_shutdown.go](./examples/07_graceful_shutdown.go)

### Step 8: Cleanup Resources (Reverse Order)

Close resources in **reverse order of creation**:

```
1. Stop accepting new requests     (Shutdown)
2. Wait for in-flight requests     (Shutdown)
3. Cancel background workers       (context cancel)
4. Wait for workers to finish      (sync.WaitGroup)
5. Close database connections      (db.Close)
6. Close file handles              (file.Close)
7. Flush and close logger          (logger.Sync)
8. Exit process
```

Reference: [examples/08_cleanup.go](./examples/08_cleanup.go)

---

## Verification Checklist

After implementing the server lifecycle, verify these criteria:

```
[ ] Server timeouts are set (ReadTimeout, ReadHeaderTimeout, WriteTimeout, IdleTimeout)
[ ] ListenAndServe runs in a goroutine
[ ] ErrServerClosed is handled correctly (not treated as fatal)
[ ] OS signals (SIGINT, SIGTERM) are captured
[ ] Shutdown uses context.WithTimeout (not infinite wait)
[ ] Shutdown is used instead of Close
[ ] Health endpoints exist (/livez, /readyz)
[ ] Liveness probe does NOT check external dependencies
[ ] Resources are cleaned up in reverse creation order
[ ] sync.WaitGroup is used for background workers
[ ] Hijacked connections (WebSocket) are handled manually if applicable
```

---

## Complete Reference Implementation

For the full production-ready implementation combining all phases,
see: [examples/complete_server.go](./examples/complete_server.go)

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It's Wrong | Correct Approach |
|:---|:---|:---|
| `ListenAndServe` in `main()` without goroutine | Blocks signal handling | Run in goroutine |
| Zero-value `http.Server{}` | No timeouts = resource exhaustion | Set all 4 timeouts |
| Using `Close()` instead of `Shutdown()` | Drops in-flight requests | Use `Shutdown(ctx)` |
| `Shutdown` without timeout context | Can hang forever | Use `context.WithTimeout` |
| Liveness checking DB | Cascading pod restarts | Check process health only |
| `err == http.ErrServerClosed` | Fragile comparison | `errors.Is(err, http.ErrServerClosed)` |
| Ignoring `SIGTERM` | Kubernetes kills ungracefully | Handle both SIGINT and SIGTERM |
| Not waiting for background workers | Data loss, race conditions | Use `sync.WaitGroup` |
