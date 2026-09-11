---
name: go-server-lifecycle
description: "Production-ready lifecycle management for Go HTTP services. Covers fail-fast initialization, server configuration, startup, liveness/readiness probes, graceful shutdown, drain behavior, background workers, signal handling, and runtime verification."
---

# Go Server Lifecycle Management Skill

This skill defines a production-ready lifecycle for Go HTTP services. It is intentionally
framework-agnostic, but it assumes a standard architecture with a composition root,
application/services layer, HTTP adapter layer, infrastructure integration, and a runtime
control layer for health, workers, and graceful shutdown.

The guidance below is designed for real services that must survive startup failures,
traffic spikes, shutdown signals, and deployment rollouts without leaving requests hanging
or leaving background workers running in the background.

## Production Principles

A correctly managed Go server is not just a server that boots. It is a service that:

- fails fast when required configuration or dependencies are invalid
- exposes correct liveness and readiness semantics
- starts background work only after the app is ready
- handles OS termination signals reliably
- drains traffic before shutting down
- closes resources in a safe order
- logs and surfaces failures in a way that supports operations
- **exposes deep observability signals for runtime monitoring**
- **provides human-friendly logs during development without sacrificing production structure**

## Official Sources & References

- [`net/http.Server`](https://pkg.go.dev/net/http#Server)
- [`Server.Shutdown`](https://pkg.go.dev/net/http#Server.Shutdown)
- [`os/signal`](https://pkg.go.dev/os/signal)
- [`context`](https://pkg.go.dev/context)
- [`sync.WaitGroup`](https://pkg.go.dev/sync#WaitGroup)
- [`go-chi/chi/v5`](https://github.com/go-chi/chi)

---

## Architectural Context

The following layering is a common and safe pattern for Go services:

- composition/bootstrap layer: config loading, logger setup, dependency wiring,
  lifecycle orchestration
- domain layer: business rules and core entities
- application/use-case layer: orchestration and service logic
- adapter/HTTP layer: router, middleware, handlers, request validation, JSON response
- infrastructure layer: database access, queues, cache, SMTP, storage, external APIs
- runtime services: health checks, worker manager, metrics, shutdown coordinator

The key invariant is simple:

- the domain layer must not depend on transport, DB driver internals, or HTTP-specific concerns
- the server lifecycle sits outside the business logic and coordinates runtime behavior
- startup, readiness, and shutdown are operational concerns and should be explicit, not implicit

A typical production layout is:

```text
┌────────────────────────────────────────────────────────────────────┐
│                      Composition / Bootstrap                        │
│  config, env, logger, signal handling, dependency wiring           │
├────────────────────────────────────────────────────────────────────┤
│                   HTTP / Adapter Layer                              │
│  router, middleware, handlers, JSON serialization, request timeout │
├────────────────────────────────────────────────────────────────────┤
│                  Application / Use Case Layer                      │
│  orchestration, validation, domain services, business rules       │
├────────────────────────────────────────────────────────────────────┤
│                        Domain Layer                                  │
│  entities, interfaces, core behaviors                              │
├────────────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                              │
│  DB, queue, email, cache, storage, external services               │
└────────────────────────────────────────────────────────────────────┘
```

---

## Seven-Phase Lifecycle

The lifecycle should be treated as an ordered operational flow.

```text
Phase 1: Initialization  → validate config, initialize logger, open dependencies
Phase 2: Configuration   → set server timeouts, define middleware, register observability providers
Phase 3: Startup         → start server in goroutine, initialize workers, expose readiness
Phase 4: Serving         → accept traffic, answer probes, process requests
Phase 5: Drain           → set not-ready, stop sending new traffic, give ingress time to react
Phase 6: Graceful Shutdown → call http.Server.Shutdown with timeout and wait for in-flight work
Phase 7: Cleanup         → stop workers, close DB/storage clients, flush logs, exit cleanly
```

This flow must be deterministic and explicit. The service should never appear “running” before readiness is complete, and it should never terminate abruptly without a drain or shutdown phase.

---

## Step 1: Fail-Fast Initialization & Dual-Mode Logging

The process should be built in a strict order, starting with a structured logger that is environment-aware.

### Professional Structured Logging

A production service needs `JSON` logs, but developers need readable logs. Use a "Dual-Mode" pattern:

- **Terminal Mode**: Use a custom `slog.Handler` (Pretty Handler) for colored, human-friendly output.
- **Production Mode**: Fallback to standard `JSONHandler`.
- **Formatting**: Always include a consistent time format (e.g., `2006-01-02 03:04:05 PM`).

Example `initLogger` pattern:

```go
func initLogger() *slog.Logger {
    out := os.Stderr
    isTerminal := isatty.IsTerminal(out.Fd())
    
    if isTerminal {
        // Return custom handler with colors and AM/PM time
        return slog.New(&prettyHandler{w: out})
    }
    
    // Production JSON
    return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey {
                return slog.String(slog.TimeKey, a.Value.Time().Format("2006-01-02 03:04:05 PM"))
            }
            return a
        },
    }))
}
```

---

## Step 2: Configure the HTTP server with Strict Timeouts

A production-ready `http.Server` must always set timeouts. The zero-value server is not safe for production.

Recommended defaults:

| Field | Recommended value | Purpose |
|:---|:---:|:---|
| `ReadTimeout` | 5s | Maximum time to read a request |
| `ReadHeaderTimeout` | 2s | Protect against slowloris and stalled headers |
| `WriteTimeout` | 10s | Maximum time to write a response |
| `IdleTimeout` | 120s | Idle keep-alive timeout |
| `MaxHeaderBytes` | 1 MB | Prevent oversized header flooding |

---

## Step 3: Use Chi with Observability Middleware

Use `go-chi/chi/v5` with a middleware stack that captures traffic quality and performance.

```go
r := chi.NewRouter()

r.Use(middleware.RequestID)
r.Use(metrics.Middleware) // Track status codes (2xx, 4xx, 5xx) and latency
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
```

### Observability Standards

- **Log Suppression**: Suppress logs for healthy observability probes (`/metrics`, `/livez`, `/readyz`) to reduce noise in logs.
- **Latency Tracking**: Track average response time in milliseconds.
- **Traffic Quality**: Distinguish between successful requests and errors.

---

## Step 4: Implement Liveness, Readiness, and Metrics Correctly

Health and metrics endpoints are operational API contracts.

| Probe | Purpose | Rules |
|:---|:---|:---|
| `/livez` | process health | process-level only; no external deps |
| `/readyz` | traffic readiness | verify required dependencies (DB, etc.) |
| `/metrics` | runtime telemetry | expose CPU, Mem, Goroutines, HTTP stats, and DB pool stats |

### Database Pool Monitoring

If using a connection pool (like `pgxpool`), export its stats:
- `ActiveConns`: Current busy connections.
- `MaxConns`: Configured limit.
- `WaitCount`: Total connections that had to wait for a slot (indicates saturation).

---

## Step 5: Start the HTTP Server in a Non-Blocking Way

The server must start in a goroutine so the main control loop can watch for signals.

```go
errCh := make(chan error, 1)

go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        errCh <- err
    }
}()
```

---

## Step 6: Handle Signals with `signal.NotifyContext`

Use `signal.NotifyContext` as the primary shutdown trigger.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

---

## Step 7: Drain Before Shutdown

When the service receives a termination signal, it must stop accepting new traffic before stopping the listener.

1. mark readiness as false.
2. wait for a configured drain period.
3. begin `http.Server.Shutdown`.

---

## Step 8: Graceful Shutdown, Not Forced Close

The server must use `http.Server.Shutdown`, not `Close`, for normal teardown.

---

## Step 9: Worker Lifecycle Management

Background workers should be treated as part of the application lifecycle.
- Start only after dependencies are ready.
- Stop before closing DB/storage dependencies.
- Use `sync.WaitGroup` to wait for clean exit.

---

## Step 10: Cleanup in Reverse Order

Cleanup should be deterministic and in strict reverse order of construction.

1. Stop HTTP server.
2. Stop background workers.
3. Close DB pools and storage clients.
4. Flush logs.

---

## Step 11: Runtime Monitoring Dashboard

For production-ready operations, provide a real-time monitor (e.g., `make monitor`) that scans the `/metrics` endpoint and renders:
- **System Health**: CPU, Memory, Goroutines.
- **Traffic Health**: Success rate, 4xx/5xx counts, Avg Latency.
- **Database Health**: Pool saturation and wait counts.
- **Alerting**: Use color-coded indicators (Red for 5xx errors, Yellow for high latency).

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It's Wrong | Correct Approach |
|:---|:---|:---|
| `go run` in Production | Signal handling fails; messy exit codes | Build binary, then execute |
| JSON logs in Development | Hard for humans to debug | Use environment-aware "Pretty Logger" |
| Logging every `/metrics` hit | Floods logs with noise | Suppress healthy probe logs |
| Missing DB pool metrics | Blind to connection leaks or saturation | Export `pgxpool` stats |
| Zero-value `http.Server{}` | Resource exhaustion vulnerability | Set all 4 timeouts |
| Using `Close()` for shutdown | Terminates in-flight requests abruptly | Use `srv.Shutdown(ctx)` |

---

## Production Verification Checklist

```text
[ ] Required dependencies fail fast
[ ] Server timeouts are explicit and non-zero
[ ] Signal.NotifyContext handles SIGINT and SIGTERM
[ ] Liveness and Readiness are separate and correct
[ ] /metrics exposes CPU, Memory, Goroutines, GC, and HTTP stats
[ ] DB pool stats (Active, Max, Wait) are exported to /metrics
[ ] Logs are colored in terminal and JSON in production
[ ] Logs for healthy /metrics and /readyz probes are suppressed
[ ] Drain phase is implemented before shutdown
[ ] Cleanup order is reverse of creation
[ ] `make monitor` renders live metrics with status indicators
[ ] All tests pass with race detector: go test -v -race ./...
```
