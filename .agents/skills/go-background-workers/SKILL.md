---
name: go-background-workers
description: "Production-ready background worker supervision, recurring task execution, and job processing for Go HTTP services. Covers panic recovery, graceful context cancellation, memory-safe ticker management, exponential backoff with jitter, and reverse-order lifecycle integration."
---

# Go Background Workers & Scheduled Tasks Skill

This skill defines production standards for executing asynchronous work, recurring cron-like tasks,
and queue processors in Go services. It ensures background processes are supervised, resilient to
panics, cleanly coordinated with OS shutdown signals, and strictly isolated from leaking goroutines.

---

## Production Worker Principles

1. **Supervision & Panic Isolation**:
   No worker goroutine should ever terminate the main process due to an unhandled panic. Every worker loop must have a deferred recovery handler (`recover()`) that logs the stack trace, alerts telemetry, and safely restarts the worker if appropriate.
2. **Context-Driven Cooperative Cancellation**:
   Workers must never run unbounded loops. Every sleep, queue fetch, or batch execution must listen to `ctx.Done()`. When cancellation is triggered (Phase 6/7 of the server lifecycle), workers must terminate promptly without dropping in-flight jobs.
3. **Memory-Safe Ticker Management**:
   Never use `time.Tick()` in long-running services (it cannot be garbage collected). Always create an explicit `time.NewTicker()` and guarantee its closure via `defer ticker.Stop()`.
4. **Exponential Backoff with Random Jitter**:
   When external dependencies (database, SMTP, message broker) fail, workers must back off exponentially and apply random jitter to prevent "thundering herd" floods.
5. **Reverse-Order Lifecycle Coordination**:
   Workers must start **after** database pools and readiness are established (Phase 4), and must stop **before** database connections are closed (Phase 8).

---

## Architecture: Worker Supervision Topology

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                         Main Server Lifecycle                            │
│                 context.Context (with SIGINT/SIGTERM)                    │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                         Worker Pool Supervisor                           │
│   sync.WaitGroup ─── Shared Cancel Context ─── Error Notification Chan   │
├───────────────────┬──────────────────────┬───────────────────────────────┤
│ Recurring Worker  │ Queue Processor      │ Maintenance Worker            │
│ (time.NewTicker)  │ (Batch / Channel)    │ (Cron Schedule)               │
│ - defer recover() │ - defer recover()    │ - defer recover()             │
│ - select ctx.Done │ - select ctx.Done    │ - select ctx.Done             │
│ - ticker.Stop()   │ - flush in-flight    │ - safe exit                   │
└───────────────────┴──────────────────────┴───────────────────────────────┘
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| `go worker()` without `recover` | Unhandled panic crashes the entire server | Wrap worker loop in `defer func() { if r := recover() ... }` |
| `for range time.Tick(d)` | Leaks the ticker channel; cannot be stopped | Use `ticker := time.NewTicker(d)` and `defer ticker.Stop()` |
| `time.Sleep()` in worker loop | Ignores shutdown signals; hangs during graceful teardown | Use `select { case <-ctx.Done(): ... case <-ticker.C: ... }` |
| Stopping DB before workers | Workers executing queries crash with connection closed errors | Stop workers first via `sync.WaitGroup`, then close DB pools |
| Fixed retry without backoff | Overwhelms recovering external services (thundering herd) | Apply exponential backoff with randomized jitter |

---

## Worker Verification Checklist

```text
[ ] Every background goroutine has a defer/recover safety block
[ ] All worker loops check ctx.Done() and exit cleanly on shutdown signals
[ ] All time.NewTicker instances are stopped with defer ticker.Stop()
[ ] Worker pool tracks goroutines with sync.WaitGroup and blocks until in-flight work finishes
[ ] In-flight database transactions are committed or rolled back before worker exits
[ ] Failed external requests apply exponential backoff with jitter
[ ] Workers stop strictly before database connection pools are closed
```
