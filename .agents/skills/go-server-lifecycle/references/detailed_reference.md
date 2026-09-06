# Go Server Lifecycle — Detailed Reference

This document provides in-depth explanations of each lifecycle phase,
including edge cases, common mistakes, and production considerations.

---

## 1. Timeout Architecture

### How Timeouts Map to the Connection Lifecycle

```
Client                                      Server
  |                                            |
  |──── TCP Connect ──────────────────────────>|  ← Connection accepted
  |                                            |     (ReadTimeout starts)
  |──── Send Headers ─────────────────────────>|  ← ReadHeaderTimeout scope
  |                                            |
  |──── Send Body ────────────────────────────>|  ← ReadTimeout scope
  |                                            |
  |                                   [Handler executes]  ← NOT covered by timeouts!
  |                                            |
  |<─── Receive Response ─────────────────────|  ← WriteTimeout scope
  |                                            |
  |              [Keep-Alive Wait]             |  ← IdleTimeout scope
  |                                            |
  |──── Next Request ─────────────────────────>|  ← ReadTimeout restarts
```

### Key Insight: Handler Timeout Gap

The four server timeouts do NOT cancel handler logic. If your handler makes
a slow database query or external API call, it will continue running even
after `WriteTimeout` expires.

**Solutions:**

1. **`http.TimeoutHandler`** — wraps a handler with a deadline:
   ```go
   handler := http.TimeoutHandler(myHandler, 30*time.Second, "timeout")
   ```

2. **`context.WithTimeout`** — use inside the handler:
   ```go
   func myHandler(w http.ResponseWriter, r *http.Request) {
       ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
       defer cancel()
       result, err := db.QueryContext(ctx, "SELECT ...")
   }
   ```

### HTTPS Considerations

When using TLS (`ListenAndServeTLS`):
- `WriteTimeout` includes the TLS handshake time
- Consider increasing `WriteTimeout` slightly for HTTPS servers
- `ReadHeaderTimeout` does NOT include TLS handshake

---

## 2. ErrServerClosed Deep Dive

### Why errors.Is() Instead of ==

```go
// BAD — fragile, won't work with wrapped errors
if err == http.ErrServerClosed {

// GOOD — works with wrapped errors too
if errors.Is(err, http.ErrServerClosed) {
```

### When ErrServerClosed Occurs

`ListenAndServe()` returns `http.ErrServerClosed` immediately when either
`Shutdown()` or `Close()` is called. The key point: this error is returned
BEFORE the shutdown process completes.

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

This is why the server MUST run in a goroutine — the main function needs
to wait for `Shutdown()` to return, not just for `ListenAndServe()`.

---

## 3. Signal Handling Edge Cases

### Double Signal (Force Kill)

Users sometimes press Ctrl+C twice when shutdown seems slow. Handle this:

```go
// First signal: graceful shutdown
// Second signal: force exit
sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

<-sigCtx.Done()
stop() // Stop catching signals — next Ctrl+C will force kill

// Proceed with graceful shutdown...
```

### SIGQUIT for Debug Dumps

Consider also handling `SIGQUIT` to dump goroutine stacks:

```go
go func() {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGQUIT)
    <-quit
    pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
}()
```

---

## 4. Drain Phase — Load Balancer Timing

### Why the Wait is Necessary

```
Without drain:                    With drain:
  T0: SIGTERM received              T0: SIGTERM received
  T0: Shutdown() called             T0: /readyz → 503
  T0: Listener closed               T5: LB removes pod
  T1: LB still routing here!        T5: Shutdown() called
  T1: New request → REJECTED        T5: Listener closed
  ❌ Request dropped                 ✅ No drops
```

### Calculating Drain Duration

The drain wait must be ≥ the load balancer's health check interval:

| Platform              | Default Health Check Interval | Recommended Drain |
|:----------------------|:------------------------------|:------------------|
| Kubernetes            | 10s                           | 10-15s            |
| AWS ALB               | 30s                           | 30-35s            |
| Google Cloud LB       | 10s                           | 10-15s            |
| Nginx upstream        | Configurable                  | interval × 2     |

---

## 5. Hijacked Connections (WebSocket)

### The Problem

From the official docs:
> "Shutdown does not attempt to close nor wait for hijacked connections
> such as WebSockets."

This means `Shutdown()` will return while WebSocket connections are still
active. If you `os.Exit()` immediately after, WebSocket clients will be
disconnected without a proper close handshake.

### Solution Pattern

```go
type ConnTracker struct {
    mu    sync.Mutex
    conns map[string]net.Conn
    wg    sync.WaitGroup
}

func (ct *ConnTracker) Add(id string, conn net.Conn) {
    ct.mu.Lock()
    ct.conns[id] = conn
    ct.wg.Add(1)
    ct.mu.Unlock()
}

func (ct *ConnTracker) Remove(id string) {
    ct.mu.Lock()
    delete(ct.conns, id)
    ct.wg.Done()
    ct.mu.Unlock()
}

func (ct *ConnTracker) CloseAll() {
    ct.mu.Lock()
    for id, conn := range ct.conns {
        conn.Close()
        delete(ct.conns, id)
    }
    ct.mu.Unlock()
}

func (ct *ConnTracker) Wait() {
    ct.wg.Wait()
}
```

---

## 6. Testing the Lifecycle

### Test: Graceful Shutdown Completes In-Flight Requests

```go
func TestGracefulShutdown(t *testing.T) {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(2 * time.Second) // Simulate slow request
        w.WriteHeader(http.StatusOK)
    })

    srv := &http.Server{Addr: ":0", Handler: handler}
    ln, _ := net.Listen("tcp", ":0")

    go srv.Serve(ln)

    // Start a slow request
    done := make(chan int, 1)
    go func() {
        resp, _ := http.Get("http://" + ln.Addr().String())
        done <- resp.StatusCode
    }()

    time.Sleep(500 * time.Millisecond) // Let request start

    // Shutdown while request is in-flight
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(ctx)

    // The in-flight request should complete successfully
    status := <-done
    if status != http.StatusOK {
        t.Errorf("expected 200, got %d", status)
    }
}
```

### Test: Shutdown Timeout Forces Exit

```go
func TestShutdownTimeout(t *testing.T) {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(10 * time.Second) // Very slow — will exceed timeout
        w.WriteHeader(http.StatusOK)
    })

    srv := &http.Server{Addr: ":0", Handler: handler}
    ln, _ := net.Listen("tcp", ":0")

    go srv.Serve(ln)

    // Start a very slow request
    go http.Get("http://" + ln.Addr().String())
    time.Sleep(500 * time.Millisecond)

    // Shutdown with short timeout — should fail
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    err := srv.Shutdown(ctx)

    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("expected DeadlineExceeded, got %v", err)
    }
}
```
