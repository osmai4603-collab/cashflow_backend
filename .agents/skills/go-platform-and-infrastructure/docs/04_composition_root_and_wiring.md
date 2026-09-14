# Composition Root & Explicit Dependency Wiring

The **Composition Root** is the single location in the application where the complete dependency graph is instantiated and assembled. In clean Go design, this resides in `internal/platform/app` (or `cmd/server/main.go`).

---

## 1. Why Explicit Wiring Over DI Frameworks?

Many frameworks (e.g. reflection-based dependency injection) obscure the startup lifecycle, introduce runtime panics, and make tracing call graphs difficult.

### Benefits of Explicit Wiring in Go:
- **Compile-Time Safety**: If a component needs a new dependency, the compiler immediately flags missing parameters.
- **Clear Lifecycle Order**: Dependencies are created strictly in topological order:
  $$\text{Config} \rightarrow \text{Logger} \rightarrow \text{Database Pool} \rightarrow \text{Platform} \rightarrow \text{Infrastructure} \rightarrow \text{Repositories} \rightarrow \text{Use Cases} \rightarrow \text{Handlers}$$
- **Zero Hidden Magic**: New team members can inspect `app.go` to understand how the entire system connects.

---

## 2. Bootstrapping Sequence

```text
Step 1: Configuration & Observability
  ├── LoadConfig()
  └── InitLogger()

Step 2: Core Storage & Connection Pools
  └── InitDatabasePool()

Step 3: Platform Primitives
  ├── InitNotificationBus()
  ├── InitAuditRecorder()
  ├── InitCurrencyConverter()
  └── InitSequenceGenerator()

Step 4: Infrastructure Gateways
  ├── InitPaymentRegistry(Stripe, PayPal)
  └── InitEmailGateway()

Step 5: Storage Adapters (Repositories)
  └── InitRepositories(dbPool)

Step 6: Application Use Cases
  └── InitUseCases(repos, gateways, platformServices)

Step 7: Presentation & Routing
  └── InitRouter(handlers, middlewares)

Step 8: Background Workers & Scheduler
  └── StartWorkers(ctx, wg)

Step 9: Server Lifecycle Execution
  └── RunServer(ctx, router)
```

---

## 3. Reverse Teardown Integration

The composition root must manage orderly shutdown:
1. Stop accepting new incoming requests (Drain phase).
2. Stop background workers and wait for active jobs (`wg.Wait()`).
3. Flush in-flight audit logs and notification events.
4. Close database connection pools and network clients in **reverse order of creation**.
