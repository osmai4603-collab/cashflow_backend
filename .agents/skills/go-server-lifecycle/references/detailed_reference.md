# Go Server Lifecycle — Detailed Reference

This document provides in-depth explanations of each lifecycle phase,
Clean Architecture integration, Chi router architecture, process signal propagation,
and production considerations.

---

## 1. Clean Architecture & Lifecycle Decoupling

In enterprise Go backends, the HTTP server lifecycle is an **infrastructure concern**. It must not bleed into core business rules or domain models.

### Directory Structure & Responsibility Boundary

```
cmd/
  server/
    main.go                 # Composition Root: Wires all layers, triggers Lifecycle
internal/
  domain/                   # Pure Domain: Entities, Value Objects, Domain Ports
    transaction.go          # Business rules (zero 3rd-party dependencies)
    ports.go                # Interfaces: Repository and external service contracts
  usecase/                  # Application Logic: Orchestrates domain entities & ports
    transaction_usecase.go
  adapters/                 # Interface Adapters: Converts external formats to internal
    http/                   # Driving Adapter: Chi router, HTTP handlers, DTOs
      router.go
      handler.go
      dto.go
    storage/                # Driven Adapter: Implements domain.Repository (SQL/Memory)
      memory_repo.go
  infrastructure/           # Frameworks & Drivers: Lifecycle, Config, OS, Containers
    config/                 # Environment & timeout configuration
    health/                 # Liveness & Readiness health monitoring service
    worker/                 # Background task manager (sync.WaitGroup + context)
    server/                 # 7-phase HTTP Server implementation
```

### The Composition Root Pattern (`main.go`)

`main.go` is the single place where dependencies are constructed in order:

```go
func main() {
    // 1. Config & Logger
    cfg := config.Load()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // 2. Driven Adapters (Storage/DB)
    repo := storage.NewMemoryTransactionRepo()

    // 3. Application Use Cases
    uc := usecase.NewTransactionUseCase(repo)

    // 4. Driving Adapters (HTTP / Chi)
    handler := httpAdapter.NewTransactionHandler(uc, logger)
    healthSvc := health.NewService(repo)
    router := httpAdapter.NewRouter(handler, healthSvc, logger)

    // 5. Infrastructure: Worker Manager & Server
    wm := worker.NewWorkerManager(logger)
    srv := server.New(cfg, router, healthSvc, wm, logger)

    // 6. Execute Lifecycle
    if err := srv.Run(); err != nil {
        logger.Error("server terminated with error", "error", err)
        os.Exit(1)
    }
}
```

---

## 2. Timeout Architecture & Chi Router Integration

### How Timeouts Map to Connection Lifecycle

```
Client                                      Server
  |                                            |
  |──── TCP Connect ──────────────────────────>|  ← Connection accepted
  |                                            |     (ReadTimeout starts)
  |──── Send Headers ─────────────────────────>|  ← ReadHeaderTimeout scope
  |                                            |
  |──── Send Body ────────────────────────────>|  ← ReadTimeout scope
  |                                            |
  |                                   [Handler executes]  ← Handled by Chi middleware.Timeout!
  |                                            |
  |                                            |
  |<─── Receive Response ─────────────────────|  ← WriteTimeout scope
  |                                            |
  |              [Keep-Alive Wait]             |  ← IdleTimeout scope
  |                                            |
  |──── Next Request ─────────────────────────>|  ← ReadTimeout restarts
```

### Closing the "Handler Timeout Gap" with Chi

`http.Server.WriteTimeout` controls the network connection write deadline. If a handler hangs on an unindexed database query or slow third-party API, `WriteTimeout` will eventually close the connection, but the handler goroutine **continues executing**, leaking CPU and database connections.

Chi provides standard, idiomatic context propagation via `middleware.Timeout`:

```go
r := chi.NewRouter()

// Production Middleware Pipeline
r.Use(middleware.RequestID)      // Injects unique X-Request-Id for distributed tracing
r.Use(middleware.RealIP)         // Parses X-Forwarded-For / X-Real-IP safely
r.Use(middleware.Logger)         // Logs incoming requests
r.Use(middleware.Recoverer)      // Catches panics and returns 500 Internal Server Error
r.Use(middleware.Timeout(30*time.Second)) // Injects context deadline into r.Context()
```

When the timeout fires, Chi cancels `r.Context()`. Any database query or HTTP client using `r.Context()` immediately aborts:

```go
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
    // Passes r.Context() down to usecase and repository
    summary, err := h.uc.GetSummary(r.Context())
    if err != nil {
        if errors.Is(r.Context().Err(), context.DeadlineExceeded) {
            http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
            return
        }
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    // write JSON response...
}
```

---

## 3. Background Worker Lifecycle (`WorkerManager`)

Production servers frequently run background jobs (e.g., metric reporting, cache warming, message consumption). These must be managed cleanly alongside HTTP connections.

### Worker Manager Implementation Pattern

```go
type WorkerManager struct {
    wg     sync.WaitGroup
    ctx    context.Context
    cancel context.CancelFunc
    logger *slog.Logger
}

func NewWorkerManager(logger *slog.Logger) *WorkerManager {
    ctx, cancel := context.WithCancel(context.Background())
    return &WorkerManager{
        ctx:    ctx,
        cancel: cancel,
        logger: logger,
    }
}

func (wm *WorkerManager) Start(name string, fn func(ctx context.Context)) {
    wm.wg.Add(1)
    go func() {
        defer wm.wg.Done()
        wm.logger.Info("background worker started", "name", name)
        fn(wm.ctx)
        wm.logger.Info("background worker stopped", "name", name)
    }()
}

func (wm *WorkerManager) StopAndWait(timeout time.Duration) error {
    wm.cancel() // signal all workers to terminate
    done := make(chan struct{})
    go func() {
        wm.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-time.After(timeout):
        return errors.New("timed out waiting for background workers to exit")
    }
}
```

---

## 4. Process Execution & The `go run` Signal Trap

### The Problem with `go run` in Makefiles and Scripts

A common developer mistake is executing `go run ./cmd/server` inside a Makefile:

```makefile
# DANGEROUS:
run:
	go run ./cmd/server
```

When you hit `Ctrl+C` (`SIGINT`):
1. The shell sends `SIGINT` to the process group.
2. `go run` is a wrapper process that compiled a temporary binary and spawned it as a child.
3. `go run` catches `SIGINT` and immediately exits with `exit status 1` or `signal: interrupt`.
4. `make` detects the non-zero exit code of `go run` and outputs:
   ```
   make: *** [Makefile:12: run] Error 1
   ```
5. Even if your Go code caught the signal and performed graceful shutdown, the terminal reports an error!

### The Solution: Direct Binary Execution

Always build the binary and execute it directly:

```makefile
BINARY_NAME=bin/server

build:
	@mkdir -p bin
	go build -o $(BINARY_NAME) ./cmd/server

run: build
	@./$(BINARY_NAME)
```

**Why this works:**
- `./bin/server` is the direct process attached to the terminal.
- When `Ctrl+C` is pressed, the Go runtime's `signal.NotifyContext` receives `SIGINT`.
- The server performs Phase 5 (Drain), Phase 6 (Graceful Shutdown), and Phase 7 (Cleanup).
- `main()` exits with status `0` (`os.Exit(0)` or normal return).
- `make` sees exit code `0` and exits cleanly without error banners.

---

## 5. ErrServerClosed Deep Dive

### Why `errors.Is()` Instead of `==`

```go
// BAD — fragile, fails with wrapped errors
if err == http.ErrServerClosed {

// GOOD — correctly unwraps errors
if errors.Is(err, http.ErrServerClosed) {
```

### When `ErrServerClosed` Occurs

`ListenAndServe()` returns `http.ErrServerClosed` immediately when either `Shutdown()` or `Close()` is called. It is returned **before** the shutdown process completes:

```
Time ──────────────────────────────────────────────>
     │                                              │
     │ Shutdown() called                            │ Shutdown() returns
     │       │                                      │
     │       ▼                                      │
     │  ListenAndServe returns ErrServerClosed      │
     │  (immediately)                               │
     │                                              │
     │  ────── In-flight requests finishing ──────  │
     │                                              │
```

This is why `ListenAndServe()` must run in a separate goroutine.

---

## 6. Drain Phase — Load Balancer Timing

### Why Drain is Essential

```
Without Drain:                    With Drain:
  T0: SIGTERM received              T0: SIGTERM received
  T0: Shutdown() called             T0: /readyz → 503 Service Unavailable
  T0: Listener closed               T5: Ingress / LB drops pod from routing pool
  T1: LB still routing here!        T5: Shutdown() called
  T1: Client request → DROPPED      T5: Listener closed (in-flight finish)
  ❌ 502 Bad Gateway to users       ✅ 0 dropped client requests
```

### Recommended Drain Intervals

| Platform              | Health Check Interval | Recommended Drain Duration |
|:----------------------|:----------------------|:---------------------------|
| Kubernetes Ingress    | 5s - 10s              | 10s - 15s                  |
| AWS ALB               | 15s - 30s             | 25s - 35s                  |
| Google Cloud HTTP LB  | 5s - 10s              | 10s - 15s                  |
| Local Development     | Immediate             | 1s - 5s                    |

---

## 7. Multi-Layer Testing Architecture

### 1. Domain Tests
Verify core business logic without network or database dependencies:
```go
func TestNewTransaction_InvalidAmount(t *testing.T) {
    _, err := domain.NewTransaction("tx1", -50.0, "DEBIT", "test")
    if !errors.Is(err, domain.ErrInvalidAmount) {
        t.Fatalf("expected ErrInvalidAmount, got %v", err)
    }
}
```

### 2. Storage Race Tests (`go test -race`)
Verify that concurrent reads and writes to repository do not cause data races:
```go
func TestMemoryRepo_ConcurrentAccess(t *testing.T) {
    repo := storage.NewMemoryTransactionRepo()
    var wg sync.WaitGroup
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            tx, _ := domain.NewTransaction(fmt.Sprintf("%d", id), 10.0, "CREDIT", "note")
            repo.Save(context.Background(), tx)
        }(i)
    }
    wg.Wait()
}
```

### 3. HTTP Handler Tests with Chi
Test routing and middlewares without starting a real TCP listener:
```go
func TestChiRouter_HealthEndpoints(t *testing.T) {
    r := httpAdapter.NewRouter(nil, healthSvc, logger)
    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/livez", nil)
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
}
```

### 4. Lifecycle Graceful Shutdown Tests
Verify that in-flight requests complete during shutdown:
```go
func TestServer_GracefulShutdown(t *testing.T) {
    // Start server on random port (:0)
    // Send slow request in goroutine
    // Trigger shutdown
    // Verify slow request returns 200 OK
}
```
