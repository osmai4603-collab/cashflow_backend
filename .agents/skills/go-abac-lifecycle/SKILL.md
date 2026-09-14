---
name: go-abac-lifecycle
description: "Production-ready, framework-agnostic Attribute-Based Access Control (ABAC) lifecycle architecture for Go applications. Covers NIST SP 800-162 standards, 4-dimensional attribute modeling (Subject, Resource, Action, Environment), dynamic PIP enrichment, PDP combining algorithms (Deny-Overrides, Permit-Overrides), context-scoped identity propagation, PEP middleware guards, RFC 7807 problem details, and tamper-evident audit logging."
---

# Go ABAC Lifecycle Architecture Skill

This skill defines a comprehensive, production-ready, and completely framework-agnostic **Attribute-Based Access Control (ABAC) Security Lifecycle** for Go applications. It is strictly decoupled from any specific business domain (e.g., banking, healthcare, CRM, e-commerce), making it directly applicable to any Go microservice, monolith, REST API, gRPC backend, or distributed cloud-native daemon.

The architecture synthesizes:

- Official Go language security standards and idioms from [go.dev](https://go.dev/doc/security) and [go.dev/security/best-practices](https://go.dev/security/best-practices).
- Formal international access control standards from **NIST SP 800-162** (*Guide to Attribute Based Access Control Definition and Considerations*) and **OASIS XACML 3.0**.
- The battle-tested authorization decoupling patterns pioneered by **Kubernetes** (`k8s.io/apiserver`) and **Google CEL** (`google/cel-go`).

---

## Production Security Principles

1. **Strict Default Deny (Fail-Closed Architecture)**:
   Access is denied by default. An action is permitted if and only if an explicit policy or combination of policies allows it. If an attribute is missing, a PIP lookup fails, a context is unauthenticated, or an internal error occurs, the decision MUST immediately evaluate to `Deny`.

2. **The 4-Dimensional Attribute Quadruple**:
   Authorization decisions evaluate four atomic dimensions in real-time:
   - **Subject ($S$)**: Principal attributes (ID, TenantID, Roles, Clearance, Department, Attributes map).
   - **Resource ($R$)**: Target entity attributes (ID, Type, OwnerID, TenantID, Sensitivity, Status, Attributes map).
   - **Action ($A$)**: Operation attributes (Verb, Method, TargetScope).
   - **Environment ($E$)**: Contextual/ambient attributes (RequestTime, ClientIP, NetworkZone, TLSVersion, SecurityPosture).

3. **Domain-Agnostic Separation of Concerns**:
   - **PAP / PRP (Policy Administration & Retrieval Point)**: Declares, validates, and stores policies (Go pure functions, CEL, or JSON rules).
   - **PIP (Policy Information Point)**: Dynamically retrieves and enriches resource and environmental attributes from databases or caches.
   - **PDP (Policy Decision Point)**: Evaluates attributes against loaded policies using deterministic combining algorithms (`Deny-Overrides`, `Permit-Overrides`).
   - **PEP (Policy Enforcement Point)**: Intercepts requests, propagates context safely, and enforces the PDP's decision at the protocol boundary (HTTP/gRPC).

4. **Type-Safe, Unexported Context Keying**:
   Caller identity and verified subject attributes MUST be injected and propagated through `context.Context` using unexported private key types (`type contextKey struct{}`) to make key collision between packages mathematically impossible.

5. **Thread-Safe, Sub-Microsecond Policy Evaluation**:
   Policy evaluation happens in the critical path of incoming requests. The PDP must evaluate rules with zero locking contention during read paths (using `sync.RWMutex` or `atomic.Pointer`) and minimal memory allocations.

6. **Information Disclosure Prevention on Rejection**:
   Forbidden responses MUST follow **RFC 7807 (Problem Details for HTTP APIs)** with HTTP status code `403 Forbidden`. The public error message MUST NOT reveal internal attribute values, sensitive rule names, or infrastructure layout.

7. **Tamper-Evident Audit Trails & Observability**:
   Every authorization decision (both permitted and denied) must emit structured logs (`log/slog`) and export real-time counters (e.g., Prometheus `authz_evaluations_total{decision="allow|deny"}`).

---

## Architectural Topology: The 7-Phase ABAC Lifecycle

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 1: Attribute Modeling       Define Subject, Resource, Action, and Environment Schemas       │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 2: Policy Authoring (PAP)    Define Policy Rules, Targets, Conditions, & Combining Algorithm │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 3: Ingress & Context (PEP)   Intercept Request, verify AuthN, inject Subject & Env to ctx  │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 4: Dynamic PIP Enrichment    Query DB/Cache to resolve Resource & ambient context           │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 5: PDP Evaluation Engine     Execute Combining Algorithm (Deny-Overrides / Permit-Overrides)│
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 6: Enforcement & Rejection   Permitted: ServeHTTP (200) | Denied: RFC 7807 Problem (403)   │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 7: Audit & Observability     Structured slog logging, Prometheus OpenMetrics, Security SIEM │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase-by-Phase Engineering Guide

### Phase 1: Domain-Agnostic Attribute Modeling

Represent the 4 core dimensions using strongly-typed Go structs with generic attribute maps for dynamic extensions:

```go
package abac

import "time"

// Subject represents the authenticated caller requesting access.
type Subject struct {
    ID         string            `json:"id"`
    TenantID   string            `json:"tenant_id,omitempty"`
    Roles      []string          `json:"roles,omitempty"`
    Department string            `json:"department,omitempty"`
    Clearance  int               `json:"clearance,omitempty"`
    Attributes map[string]any    `json:"attributes,omitempty"`
}

// Resource represents the target entity being accessed or mutated.
type Resource struct {
    ID          string            `json:"id"`
    Type        string            `json:"type"` // e.g., "document", "transaction", "profile"
    OwnerID     string            `json:"owner_id,omitempty"`
    TenantID    string            `json:"tenant_id,omitempty"`
    Department  string            `json:"department,omitempty"`
    Sensitivity int               `json:"sensitivity,omitempty"`
    Status      string            `json:"status,omitempty"`
    Attributes  map[string]any    `json:"attributes,omitempty"`
}

// Action represents the verb or operation requested on the resource.
type Action struct {
    Verb   string `json:"verb"`             // e.g., "read", "create", "update", "delete", "approve"
    Method string `json:"method,omitempty"` // HTTP method or RPC method
}

// Environment represents ambient contextual metadata at evaluation time.
type Environment struct {
    RequestTime time.Time         `json:"request_time"`
    ClientIP    string            `json:"client_ip,omitempty"`
    NetworkZone string            `json:"network_zone,omitempty"`
    Attributes  map[string]any    `json:"attributes,omitempty"`
}

// EvaluationContext encapsulates the complete attribute quadruple.
type EvaluationContext struct {
    Subject     Subject
    Resource    Resource
    Action      Action
    Environment Environment
}
```

### Phase 2: Policy Authoring and Combining Algorithms (PAP/PDP)

A policy rule implements the `PolicyRule` interface. The PDP combines multiple rules using a deterministic algorithm:

```go
package abac

type Decision int

const (
    DecisionNotApplicable Decision = iota
    DecisionPermit
    DecisionDeny
)

type PolicyRule interface {
    ID() string
    Description() string
    Target(ctx EvaluationContext) bool
    Evaluate(ctx EvaluationContext) (Decision, string)
}

type CombiningAlgorithm int

const (
    DenyOverrides CombiningAlgorithm = iota
    PermitOverrides
    FirstApplicable
)
```

### Phase 3: Request Interception and Type-Safe Context Propagation (PEP)

Never use raw strings for context keys. Always declare an unexported type:

```go
package abac

import "context"

type contextKey struct{}

var subjectContextKey = contextKey{}

func WithSubject(ctx context.Context, sub Subject) context.Context {
    return context.WithValue(ctx, subjectContextKey, sub)
}

func GetSubject(ctx context.Context) (Subject, bool) {
    sub, ok := ctx.Value(subjectContextKey).(Subject)
    return sub, ok
}
```

### Phase 4: Dynamic Policy Information Point (PIP) Enrichment

In real-world applications, resource attributes cannot be trusted from client tokens; they must be retrieved from the persistent state:

```go
type PIPResolver interface {
    ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error)
}
```

### Phase 5: High-Performance, Thread-Safe PDP Engine

The engine manages policy rules safely under concurrent reads:

```go
type Engine struct {
    mu         sync.RWMutex
    algorithm  CombiningAlgorithm
    rules      []PolicyRule
    logger     *slog.Logger
}

func (e *Engine) Evaluate(ctx EvaluationContext) (Decision, string) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    switch e.algorithm {
    case DenyOverrides:
        // If any applicable rule denies, immediate Deny
        // Must have at least one applicable rule permitting, else Default Deny
    ...
    }
}
```

### Phase 6: Enforcement (PEP) and RFC 7807 Safe Rejections

When an access check fails, return a standard RFC 7807 response:

```go
type ProblemDetails struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail"`
    Instance string `json:"instance"`
}

func WriteForbiddenProblem(w http.ResponseWriter, r *http.Request, detail string) {
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(http.StatusForbidden)
    _ = json.NewEncoder(w).Encode(ProblemDetails{
        Type:     "https://example.com/errors/forbidden",
        Title:    "Forbidden",
        Status:   http.StatusForbidden,
        Detail:   detail,
        Instance: r.URL.Path,
    })
}
```

### Phase 7: Audit Logging and Observability

Record structured audit events without leaking sensitive credentials:

```go
logger.Warn("ABAC evaluation denied",
    slog.String("rule_id", failedRuleID),
    slog.String("subject_id", ctx.Subject.ID),
    slog.String("tenant_id", ctx.Subject.TenantID),
    slog.String("resource_type", ctx.Resource.Type),
    slog.String("resource_id", ctx.Resource.ID),
    slog.String("action", ctx.Action.Verb),
    slog.String("client_ip", ctx.Environment.ClientIP),
    slog.Duration("eval_latency", elapsed),
)
```

---

## Production Security Checklist

- [ ] **Default Deny / Fail-Closed**: Any unhandled request, empty policy list, or evaluation error results in `Deny`.
- [ ] **BOLA/IDOR Defense**: All tenant-scoped operations enforce `Subject.TenantID == Resource.TenantID`.
- [ ] **Thread-Safe Policy Updates**: Policy cache updates utilize `sync.RWMutex` or `atomic.Pointer`.
- [ ] **Race Condition Validation**: Every codebase running this skill must pass `go test -race ./...`.
- [ ] **No Context Key Collisions**: Context values are keyed by unexported struct types (`type contextKey struct{}`).
- [ ] **RFC 7807 Compliance**: Rejections return `403 Forbidden` with `application/problem+json`.
- [ ] **Information Disclosure Prevention**: Error details never reveal internal policy rules, schemas, or database fields to untrusted clients.
- [ ] **Audit Trail & SIEM Ingestion**: All denials and permissions produce structured logs via `log/slog`.
