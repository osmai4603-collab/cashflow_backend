# High-Throughput Thread-Safe Caching & Zero-Downtime Hot-Reload

Authorization evaluations happen on almost every incoming request. Querying a relational database on each evaluation is an anti-pattern that creates a bottleneck. This document explains how to implement sub-microsecond in-memory caching with zero-downtime hot-reloading in Go.

---

## 1. Concurrency Patterns: `sync.RWMutex` vs `atomic.Pointer`

Depending on the write frequency of role definitions, Go offers two idiomatic concurrency models:

```text
Pattern A: sync.RWMutex (High Concurrency, Periodic Updates)
┌────────────────────────────────────────────────────────┐
│ Multiple Reader Goroutines (RLock) ────► Shared Memory  │
│ Single Writer Goroutine (Lock) ────────► Shared Memory  │
└────────────────────────────────────────────────────────┘

Pattern B: atomic.Pointer (Read-Heavy, Zero Lock Contention)
┌────────────────────────────────────────────────────────┐
│ Readers ───────────────────────────────► Snapshot A    │
│ Writer prepares new Snapshot B in memory               │
│ Writer atomically swaps pointer: Snapshot A ──► B      │
└────────────────────────────────────────────────────────┘
```

### Pattern A: `sync.RWMutex` (Standard In-Memory Engine)

```go
type Engine struct {
 mu          sync.RWMutex
 rolePerms   map[Role]map[Permission]bool
 inheritance map[Role][]Role
}

func (e *Engine) HasPermission(roles []Role, required Permission) bool {
 e.mu.RLock()
 defer e.mu.RUnlock()
 // Read-safe traversal
 return e.evaluate(roles, required)
}
```

### Pattern B: Lock-Free Snapshot Swap via `atomic.Pointer`

For ultra-high-throughput architectures ($> 100,000\text{ req/s}$) where write contention during policy reload is unacceptable, use Go 1.19+ `sync/atomic.Pointer`:

```go
package rbac

import (
 "sync/atomic"
)

type PolicySnapshot struct {
 RolePerms   map[Role]map[Permission]bool
 Inheritance map[Role][]Role
}

type AtomicEngine struct {
 current atomic.Pointer[PolicySnapshot]
}

func NewAtomicEngine(initial *PolicySnapshot) *AtomicEngine {
 e := &AtomicEngine{}
 e.current.Store(initial)
 return e
}

// HasPermission performs zero-lock evaluation
func (e *AtomicEngine) HasPermission(roles []Role, required Permission) bool {
 snap := e.current.Load()
 if snap == nil {
  return false // Default Deny
 }
 return evaluateSnapshot(snap, roles, required)
}

// Reload replaces the entire policy snapshot atomically without blocking any readers
func (e *AtomicEngine) Reload(newSnapshot *PolicySnapshot) {
 e.current.Store(newSnapshot)
}
```

---

## 2. Dynamic Hot-Reloading Triggering Strategies

In production, policies change when admins add permissions or modify roles. The in-memory cache should reload without service restarts:

1. **Database Change CDC / Polling**:
   A background worker queries the `roles` and `permissions` tables every $N$ seconds, compares an updated version timestamp, and rebuilds the in-memory snapshot.
2. **In-Process or Distributed Pub/Sub Notification**:
   When an admin executes a policy change via the management API, a message is published to an internal notification channel or Redis/NATS topic (`rbac.policy.updated`), prompting all server instances to trigger a cache reload.
3. **POSIX SIGHUP Reload**:
   Listening for `syscall.SIGHUP` to trigger a reload from a local configuration file:

```go
func ListenForPolicyReloadSignal(engine *AtomicEngine, loader func() (*PolicySnapshot, error)) {
 c := make(chan os.Signal, 1)
 signal.Notify(c, syscall.SIGHUP)
 go func() {
  for range c {
   newSnap, err := loader()
   if err == nil {
    engine.Reload(newSnap)
   }
  }
 }()
}
```
