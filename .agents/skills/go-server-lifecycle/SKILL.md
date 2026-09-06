---
name: go-server-lifecycle
description: >-
  Manages the complete lifecycle of an HTTP server in Go, from initialization
  to graceful shutdown, integrated with Clean Architecture, Chi router, and
  production background workers. Covers all 7 lifecycle phases: Initialization,
  Configuration, Startup, Serving, Drain, Graceful Shutdown, and Cleanup, plus
  process management (Makefile/Signals) and testing strategies.
---

# Go Server Lifecycle Management Skill

This skill provides a complete, standards-based approach to managing the
lifecycle of an HTTP server in Go, structured cleanly according to **Clean Architecture**
principles, powered by the lightweight and idiomatic **Chi router (`go-chi/chi/v5`)**,
and following official Go standard library patterns.

## Official Sources & References

- [`net/http.Server`](https://pkg.go.dev/net/http#Server)
- [`Server.Shutdown`](https://pkg.go.dev/net/http#Server.Shutdown)
- [`os/signal`](https://pkg.go.dev/os/signal)
- [`context`](https://pkg.go.dev/context)
- [`sync.WaitGroup`](https://pkg.go.dev/sync#WaitGroup)
- [`go-chi/chi/v5`](https://github.com/go-chi/chi)

---

## Architectural Context: Clean Architecture & Lifecycle

In production systems, server lifecycle logic should be cleanly separated from
business rules and data access:

```
┌──────────────────────────────────────────────────────────┐
│                      Infrastructure                      │
│  ┌─────────────────┐ ┌────────────────┐ ┌─────────────┐  │
│  │ HTTP Server     │ │ WorkerManager  │ │ Health Svc  │  │
│  │ (7 Lifecycle    │ │ (sync.WG, ctx) │ │ (/livez,    │  │
│  │  Phases)        │ │                │ │  /readyz)   │  │
│  └─────────────────┘ └────────────────┘ └─────────────┘  │
│                            │                             │
│       ┌────────────────────┼─────────────────────┐       │
│       ▼                    ▼                     ▼       │
│  ┌────────────────────────────────────────────────────┐  │
│  │                     Adapters                       │  │
│  │  ┌──────────────────────┐  ┌────────────────────┐  │  │
│  │  │ HTTP (Chi Router,    │  │ Storage / DB       │  │  │
│  │  │ Handlers, DTOs)      │  │ (SQL, Memory Repo) │  │  │
│  │  └──────────────────────┘  └────────────────────┘  │  │
│  │                             ▲                      │  │
│  └─────────────────────────────┼──────────────────────┘  │
│                                │                         │
│  ┌─────────────────────────────┼──────────────────────┐  │
│  │                         Usecases                   │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │ Business Logic & Orchestration               │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  │                             │                      │  │
│  └─────────────────────────────┼──────────────────────┘  │
│                                │                         │
│  ┌─────────────────────────────▼──────────────────────┐  │
│  │                          Domain                    │  │
│  │  ┌──────────────────────┐  ┌────────────────────┐  │  │
│  │  │ Entities & Rules     │  │ Repository Ports   │  │  │
│  │  └──────────────────────┘  └────────────────────┘  │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### Lifecycle Overview (7 Phases)

```
Phase 1: Initialization  → Load config, create DB/repos, usecases, Chi handlers
Phase 2: Configuration   → Server timeouts (Read, ReadHeader, Write, Idle), Chi middlewares
Phase 3: Startup         → Non-blocking ListenAndServe in goroutine, start background workers
Phase 4: Serving         → Health probes: /livez (process alive), /readyz (dependencies ready)
Phase 5: Drain           → Mark not-ready (503), wait for Load Balancer propagation
Phase 6: Graceful Shutdown → server.Shutdown(ctx) with deadline, wait for in-flight HTTP
Phase 7: Cleanup         → Stop workers (WorkerManager), close DB/storage, flush logger
```

---

## Step-by-Step Instructions

### Step 1: Initialize Dependencies (Clean Architecture Wiring)

In the composition root (`cmd/server/main.go`), initialize all layers following
the Clean Architecture dependency rule (inward dependency flow):

1. **Config & Logger**: Load environment variables and initialize structured logging (`slog`).
2. **Driven Adapters (Storage/DB)**: Connect to database or create repository implementations.
3. **Use Cases**: Instantiate application logic with repository interfaces (Ports).
4. **Driving Adapters (HTTP & Chi Router)**: Instantiate handlers with use cases, configure Chi router.
5. **Infrastructure**: Create `HealthService`, `WorkerManager`, and `Server`.

> **CRITICAL**: If any critical dependency fails during initialization, **exit immediately** with `os.Exit(1)`. Never start a broken server.

Reference: [examples/01_initialization.go](./examples/01_initialization.go) and [examples/09_clean_arch_chi_lifecycle.go](./examples/09_clean_arch_chi_lifecycle.go)

---

### Step 2: Configure Server with Timeouts & Chi Middlewares

#### A. HTTP Server Timeouts
**CRITICAL**: Never use a zero-value `http.Server`. Always set network timeouts to protect against Slowloris and connection leaks:

| Field                 | Recommended Value | Purpose                          |
|:----------------------|:------------------|:---------------------------------|
| `ReadTimeout`         | 5s                | Max time to read full request    |
| `ReadHeaderTimeout`   | 2s                | Slowloris attack protection      |
| `WriteTimeout`        | 10s               | Max time to write response       |
| `IdleTimeout`         | 120s              | Keep-alive wait for next request |
| `MaxHeaderBytes`      | 1 MB (`1 << 20`)  | Prevent header flood attacks     |

#### B. Chi Router & Middleware Pipeline
Use `go-chi/chi/v5` for lightweight, standard-library-compatible HTTP routing. Configure standard production middlewares:

```go
r := chi.NewRouter()

// 1. Request tracking & client IP
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)

// 2. Structured logging & panic recovery
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)

// 3. Handler-level timeout (Closes the "Handler Timeout Gap")
r.Use(middleware.Timeout(30 * time.Second))
```

> **NOTE**: Server network timeouts (`WriteTimeout`) do not terminate long-running handler goroutines. Using Chi's `middleware.Timeout` ensures request contexts are cancelled when handlers take too long.

Reference: [examples/02_configuration.go](./examples/02_configuration.go)

---

### Step 3: Start Server & Background Workers (Non-blocking)

Run `srv.ListenAndServe()` in a **separate goroutine** so that `main()` remains free to listen for operating system signals.

Concurrently, launch background workers through a dedicated `WorkerManager`:

```go
// Start background worker manager
workerManager := worker.NewWorkerManager(logger)
workerManager.Start("stats-reporter", func(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // worker job
        }
    }
})

// Start HTTP Server in background goroutine
errChan := make(chan error, 1)
go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        errChan <- err
    }
}()
```

Rules:
- Always check the error returned by `ListenAndServe()`.
- Filter out `http.ErrServerClosed` — it is the **expected** error when `Shutdown()` is invoked.
- Always use `errors.Is(err, http.ErrServerClosed)`, never `err == http.ErrServerClosed`.

Reference: [examples/03_startup.go](./examples/03_startup.go)

---

### Step 4: Implement Health Check Endpoints

Register separate endpoints on the Chi router for distinct container lifecycle probes:

| Probe | Route | Purpose | Rule |
|:------|:------|:--------|:-----|
| **Liveness** | `/livez` | Verifies the process is alive | **Never** check external DBs (prevents cascading pod crashes) |
| **Readiness** | `/readyz` | Verifies the app is ready for traffic | Check DB connectivity and the server `isReady` atomic flag |

```go
r.Get("/livez", healthHandler.Livez)
r.Get("/readyz", healthHandler.Readyz)
```

Reference: [examples/04_health_checks.go](./examples/04_health_checks.go)

---

### Step 5: Handle OS Signals Cleanly

Listen for termination signals using `signal.NotifyContext` (Go 1.16+):

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

select {
case <-ctx.Done():
    logger.Info("shutdown signal received")
case err := <-errChan:
    logger.Error("server startup failed", "error", err)
    return
}
```

- Catches both `os.Interrupt` (`SIGINT` from Ctrl+C) and `syscall.SIGTERM` (sent by Kubernetes, Docker, systemd).

Reference: [examples/05_signal_handling.go](./examples/05_signal_handling.go)

---

### Step 6: Implement Drain Phase (Production Load Balancer Sync)

When receiving a shutdown signal:
1. Atomically set readiness flag to `false` (making `/readyz` return `503 Service Unavailable`).
2. Sleep for the configured drain duration (e.g., 5 to 15 seconds) so that external load balancers and Kubernetes ingress controllers detect the 503 and remove the pod from their active routing tables.
3. Only after the drain period completes, proceed to close the HTTP listener.

```go
healthSvc.SetReady(false)
time.Sleep(drainDuration)
```

Reference: [examples/06_drain_phase.go](./examples/06_drain_phase.go)

---

### Step 7: Graceful HTTP Shutdown with Timeout

Stop accepting new connections and allow in-flight HTTP requests to complete:

```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()

if err := srv.Shutdown(shutdownCtx); err != nil {
    logger.Error("server forced to shutdown", "error", err)
}
```

- Always use `srv.Shutdown(ctx)` instead of `srv.Close()`.
- Always bound the shutdown with a reasonable timeout (e.g., 10s to 30s).

Reference: [examples/07_graceful_shutdown.go](./examples/07_graceful_shutdown.go)

---

### Step 8: Cleanup Resources in Reverse Order

After HTTP connections have drained and closed, tear down remaining resources in **strict reverse order of creation**:

```
1. HTTP Server stopped             (srv.Shutdown completed)
2. Background Workers cancelled    (WorkerManager context cancelled)
3. Background Workers completed    (WorkerManager.Wait())
4. Storage / DB connections closed (repo.Close() / db.Close())
5. File handles closed             (file.Close())
6. Logger flushed                  (logger.Sync())
7. Process terminates with code 0
```

Reference: [examples/08_cleanup.go](./examples/08_cleanup.go)

---

### Step 9: Process Execution & Signal Propagation (Makefile Gotcha)

#### The `go run` Trap
When using `make run`, never use `go run ./cmd/...` in the Makefile target:
```makefile
# BAD — DO NOT DO THIS
run:
	go run ./cmd/server
```
**Why?** `go run` compiles the binary to a temporary folder and launches a sub-process. When you press `Ctrl+C` (`SIGINT`), `go run` intercepts the signal and immediately exits with exit code 1 / signal 2 (`make: *** [Makefile: run] Error 1`), interrupting the graceful shutdown output and causing Make to report a failure.

#### The Correct Pattern
Always compile the binary first and execute it directly:
```makefile
# GOOD — Production & Local Development
BINARY_NAME=bin/server

build:
	@mkdir -p bin
	go build -o $(BINARY_NAME) ./cmd/server

run: build
	@./$(BINARY_NAME)
```
Executing the binary directly allows the Go runtime to capture the signal cleanly, execute all 7 phases, and exit with code `0`.

---

### Step 10: Multi-Layer Testing Strategy

Ensure production reliability by testing every architectural layer:

1. **Domain Layer**: Pure unit tests verifying business rules and constraints (`go test ./internal/domain/...`).
2. **Use Case Layer**: Unit tests mocking repository ports (`go test ./internal/usecase/...`).
3. **Storage Adapters**: Concurrent read/write safety tests using `-race` (`go test -v -race ./internal/adapters/storage/...`).
4. **HTTP Adapters & Chi Router**: Route assertions, middleware behaviors, and status codes using `httptest.NewRecorder` and `httptest.NewServer`.
5. **Infrastructure & Lifecycle**:
   - Verify non-blocking startup.
   - Verify `/livez` returns 200 while `/readyz` returns 503 during drain.
   - Verify graceful shutdown completes in-flight requests under timeout context.

---

## Verification Checklist

After implementing the server lifecycle, verify these criteria:

```
[ ] Clean Architecture layers: Domain has 0 external dependencies
[ ] Ports defined in domain, adapters implement them
[ ] Server timeouts set (ReadTimeout, ReadHeaderTimeout, WriteTimeout, IdleTimeout, MaxHeaderBytes)
[ ] Chi router used with production middlewares (RequestID, RealIP, Logger, Recoverer, Timeout)
[ ] ListenAndServe runs in a separate goroutine
[ ] ErrServerClosed handled with errors.Is (not treated as fatal)
[ ] OS signals (SIGINT, SIGTERM) captured via signal.NotifyContext
[ ] Drain phase marks server not-ready and waits for LB table updates
[ ] Shutdown uses context.WithTimeout and server.Shutdown (never Close)
[ ] Health endpoints exist (/livez process-only, /readyz dependency-checked)
[ ] WorkerManager cancels workers via context and waits with sync.WaitGroup
[ ] Resources cleaned up in reverse creation order (Workers -> Storage -> Logger)
[ ] Makefile compiles binary before executing (avoids go run signal exit code 1)
[ ] All tests pass with race detector: go test -v -race ./...
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It's Wrong | Correct Approach |
|:---|:---|:---|
| `go run` in `Makefile run` | Returns exit status 1 / signal 2 on `Ctrl+C` | `go build -o bin/server ...` then run binary |
| `ListenAndServe` in `main()` without goroutine | Blocks signal handling and graceful shutdown | Run `srv.ListenAndServe()` in goroutine |
| Zero-value `http.Server{}` | No timeouts = resource exhaustion and Slowloris vulnerability | Set all 4 timeouts + `MaxHeaderBytes` |
| Using `Close()` instead of `Shutdown()` | Immediately terminates in-flight client requests | Use `srv.Shutdown(ctx)` with timeout |
| Handler timeout gap | Network timeouts do not stop slow handlers | Use Chi `middleware.Timeout(duration)` |
| Liveness checking DB | Temporary DB glitch restarts all containers (cascading failure) | `/livez` checks process only; `/readyz` checks DB |
| `err == http.ErrServerClosed` | Fragile comparison, fails on wrapped errors | `errors.Is(err, http.ErrServerClosed)` |
| Unmanaged background goroutines | Leaks memory, loses data during termination | Use `WorkerManager` with `sync.WaitGroup` & context |
| Mixing business logic in HTTP handlers | Violates Clean Architecture, impedes testability | Handlers call UseCases, UseCases call Domain Ports |
