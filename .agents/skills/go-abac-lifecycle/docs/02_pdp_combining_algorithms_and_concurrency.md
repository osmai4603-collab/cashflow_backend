# PDP Combining Algorithms and High-Concurrency Evaluation in Go

This document details the architecture and implementation of the **Policy Decision Point (PDP)** in Go, focusing on combining algorithms, thread safety, lock-free evaluation, and zero-downtime policy hot-reloading.

---

## 1. The Role of the Policy Decision Point (PDP)

The PDP is the mathematical engine of an ABAC system. Given an `EvaluationContext` $(S, R, A, E)$ and a collection of active policy rules $\{R_1, R_2, \dots, R_n\}$, the PDP evaluates applicable rules and combines their outputs into a single, definitive decision: **Permit** or **Deny**.

```text
               ┌────────────────────────────────────────┐
               │    Incoming Evaluation Context (C)     │
               │        (Subject, Resource, ...)        │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │         Active Policy Rules            │
               │   Rule 1 ───▶ Rule 2 ───▶ Rule 3 ...   │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │         Combining Algorithm            │
               │   (Deny-Overrides / Permit-Overrides)  │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │             Final Decision             │
               │         Permit  OR  Strict Deny        │
               └────────────────────────────────────────┘
```

---

## 2. Policy Combining Algorithms

When multiple rules target the same request, conflicts can occur (e.g., Rule A allows, but Rule B forbids). The PDP resolves these conflicts using a deterministic **Combining Algorithm**.

### 2.1 Deny-Overrides (Financial & High-Security Standard)

- **Rule**: If **any** applicable rule evaluates to `Deny`, the overall decision is immediately and strictly `Deny`.
- **Condition for Permit**: Access is granted if and only if at least one applicable rule evaluates to `Permit`, and **no** applicable rule evaluates to `Deny`.
- **Default Fallback**: If no rules are applicable, the decision is `Deny` (Default Deny).
- **Use Case**: Core financial ledgers, critical infrastructure, data exfiltration defense.

```go
func evaluateDenyOverrides(rules []PolicyRule, ctx EvaluationContext) (Decision, string) {
    hasPermit := false
    permitReason := ""

    for _, rule := range rules {
        if !rule.Target(ctx) {
            continue
        }
        dec, reason := rule.Evaluate(ctx)
        if dec == DecisionDeny {
            return DecisionDeny, fmt.Sprintf("denied by rule %q: %s", rule.ID(), reason)
        }
        if dec == DecisionPermit {
            hasPermit = true
            permitReason = reason
        }
    }

    if hasPermit {
        return DecisionPermit, permitReason
    }
    return DecisionDeny, "default deny: no applicable rule permitted the action"
}
```

### 2.2 Permit-Overrides

- **Rule**: If **any** applicable rule evaluates to `Permit`, the decision is `Permit`, even if other rules evaluate to `Deny`.
- **Condition for Deny**: Access is denied if all applicable rules evaluate to `Deny`, or if no rules apply.
- **Use Case**: Public read endpoints with emergency bypass grants, open collaboration portals.

### 2.3 First-Applicable

- **Rule**: The rules are evaluated sequentially in order of registration. The decision of the first rule whose `Target(ctx)` matches and produces a `Permit` or `Deny` is adopted immediately.
- **Use Case**: Firewall-style ordered rule chains where performance requires short-circuit evaluation.

---

## 3. Thread-Safe In-Memory Caching & Hot-Reloading

Because authorization checks sit directly in the critical request path, network calls or disk reads to fetch policies on every request introduce intolerable latency. Policies must reside in-memory.

### 3.1 Pattern A: Read-Heavy Concurrent Scalability with `sync.RWMutex`

For systems where policies are reloaded periodically or via administrative mutations:

```go
type Engine struct {
    mu        sync.RWMutex
    algorithm CombiningAlgorithm
    rules     []PolicyRule
    logger    *slog.Logger
}

func (e *Engine) Evaluate(ctx EvaluationContext) (Decision, string) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    switch e.algorithm {
    case DenyOverrides:
        return evaluateDenyOverrides(e.rules, ctx)
    case PermitOverrides:
        return evaluatePermitOverrides(e.rules, ctx)
    case FirstApplicable:
        return evaluateFirstApplicable(e.rules, ctx)
    default:
        return DecisionDeny, "unknown combining algorithm"
    }
}

// ReloadRules atomically swaps the active policy rule set under a write lock.
func (e *Engine) ReloadRules(newRules []PolicyRule) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.rules = make([]PolicyRule, len(newRules))
    copy(e.rules, newRules)
}
```

### 3.2 Pattern B: Lock-Free Atomic State Swapping with `sync/atomic`

For ultra-high throughput environments (hundreds of thousands of requests per second), read locks can incur CPU cache-line bouncing. Using `atomic.Pointer` provides zero read synchronization overhead:

```go
import "sync/atomic"

type PolicySnapshot struct {
    algorithm CombiningAlgorithm
    rules     []PolicyRule
}

type AtomicEngine struct {
    snapshot atomic.Pointer[PolicySnapshot]
}

func NewAtomicEngine(alg CombiningAlgorithm, rules []PolicyRule) *AtomicEngine {
    eng := &AtomicEngine{}
    eng.snapshot.Store(&PolicySnapshot{
        algorithm: alg,
        rules:     rules,
    })
    return eng
}

func (e *AtomicEngine) Evaluate(ctx EvaluationContext) (Decision, string) {
    snap := e.snapshot.Load() // Atomic pointer load: 0 lock contention
    switch snap.algorithm {
    case DenyOverrides:
        return evaluateDenyOverrides(snap.rules, ctx)
    // ...
    }
    return DecisionDeny, "unknown algorithm"
}

func (e *AtomicEngine) SwapPolicies(newAlg CombiningAlgorithm, newRules []PolicyRule) {
    e.snapshot.Store(&PolicySnapshot{
        algorithm: newAlg,
        rules:     newRules,
    })
}
```

---

## 4. Performance Guidelines: Sub-Microsecond Evaluation

1. **Avoid Heap Allocations in Rules**: Do not allocate new slices or maps inside `rule.Evaluate()`. Pass references or use existing struct fields.
2. **Pre-Filter Targets**: Implement `rule.Target(ctx) bool` as a cheap, early-exit check (e.g., checking `ctx.Resource.Type == "invoice"`) before running complex multi-attribute comparisons.
3. **Validate with `-race`**: Always run tests with `go test -race` to verify that concurrent evaluation and policy reloading never race on shared state.
