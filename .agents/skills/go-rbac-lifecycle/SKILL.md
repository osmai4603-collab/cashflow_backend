---
name: go-rbac-lifecycle
description: "Production-ready, framework-agnostic Role-Based Access Control (RBAC) lifecycle architecture for Go applications. Covers NIST/ANSI RBAC standards (Core, Hierarchical, Constrained SSD/DSD, Symmetric), context-scoped identity propagation, lock-free/RWMutex policy caching, hot-reloading, default-deny PEP guards, RFC 7807 problem details, and tamper-evident audit logging."
---

# Go RBAC Lifecycle Architecture Skill

This skill defines a comprehensive, production-ready, and completely framework-agnostic **Role-Based Access Control (RBAC) Security Lifecycle** for Go services. It is strictly decoupled from any specific business domain (e.g., e-commerce, banking, ERP, SaaS), making it directly applicable to any Go microservice, monolith, REST API, gRPC backend, or distributed cloud-native daemon.

The architecture synthesizes:

- Official Go language security standards and idioms from [go.dev](https://go.dev/doc/security) and [go.dev/security/best-practices](https://go.dev/security/best-practices).
- Formal international access control standards from **NIST SP 800-21d** and **ANSI/INCITS 359-2012** (Core, Hierarchical, Constrained SSD/DSD, Symmetric RBAC).
- The battle-tested authorization decoupling patterns pioneered by **Kubernetes** (`k8s.io/apiserver`).

---

## Production Security Principles

1. **Strict Default Deny (Fail-Closed Architecture)**:
   Access is denied by default. An action is permitted if and only if an explicit matching permission grant is discovered through active or inherited roles. If a role lookup fails, a token is malformed, or an internal error occurs, the decision MUST immediately fall back to `Deny`.

2. **Domain-Agnostic Separation of Concerns**:
   - **Identity & Authentication (AuthN)** verifies *who* the caller is (`Subject`).
   - **Authorization (AuthZ / PDP)** evaluates *what* the subject is allowed to perform based on assigned roles and calculated effective permissions.
   - **Enforcement (PEP)** blocks or forwards execution at the protocol boundary (HTTP, gRPC, CLI).
   - **Domain / Repository Layer** applies fine-grained object scoping (e.g., tenant or entity ownership).

3. **Type-Safe, Unexported Context Keying**:
   Caller identity and active roles MUST be propagated through `context.Context` using unexported private key types (`type contextKey struct{}`) to make collision between packages mathematically impossible.

4. **Thread-Safe, High-Throughput In-Memory Caching**:
   Permission evaluations occur on every incoming request and must execute within sub-microsecond latency. The policy store and role inheritance tree must live in memory, protected by `sync.RWMutex` for concurrent read scalability or an `atomic.Pointer` for zero-downtime hot-reloading.

5. **Constrained Separation of Duties (SSD & DSD)**:
   - **Static Separation of Duties (SSD)**: Enforces mutual exclusion at role assignment time (e.g., a subject cannot possess both `billing_creator` and `billing_approver`).
   - **Dynamic Separation of Duties (DSD)**: Enforces mutual exclusion at session/request activation time (e.g., a subject who holds both roles cannot activate both within the same operational transaction).

6. **Information Disclosure Prevention on Rejection**:
   Forbidden responses MUST follow **RFC 7807 (Problem Details for HTTP APIs)** with HTTP status code `403 Forbidden`. The public error message MUST NOT reveal internal role structures, missing permission strings, or database schemas.

7. **Tamper-Evident Audit Trails & Observability**:
   Every authorization decision (especially denials) must emit structured logs (`log/slog`) and export real-time counters (e.g., Prometheus `authz_evaluations_total{decision="denied"}`).

---

## Architectural Topology: The 7-Phase RBAC Lifecycle

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 1: Modeling          Define Permissions (resource:action), Roles, and Inheritance DAG      │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 2: Provisioning      Store in persistence (SQL/KV) + Load into in-memory cache (RWMutex)   │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 3: Assignment        Bind Subject -> Roles with Tenant Scoping & SSD Mutual Exclusion      │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 4: Ingress & Context Intercept incoming request, verify Token, inject Subject into context  │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 5: Evaluation (PDP)  Traverse Hierarchical DAG, resolve Effective Permissions, verify DSD  │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 6: Enforcement (PEP) Allowed: Forward to Handler (200) | Denied: Return RFC 7807 (403)     │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Phase 7: Audit & Revoke    Emit slog JSON + OpenMetrics, manage token revocation & access review │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase-by-Phase Engineering Guide

### Phase 1: Domain-Agnostic Modeling

Represent permissions using structured constants following the standard `resource:action` convention:

```go
package rbac

type Permission string

type Role string

type Subject struct {
 ID       string            `json:"id"`
 TenantID string            `json:"tenant_id,omitempty"` // For multi-tenant systems
 Roles    []Role            `json:"roles"`
 Metadata map[string]string `json:"metadata,omitempty"`
}
```

### Phase 2: In-Memory Engine with Hierarchical DAG & Hot-Reload

A production RBAC engine resolves effective permissions by traversing a Directed Acyclic Graph (DAG) representing role inheritance ($r_{senior} \succeq r_{junior}$):

```go
package rbac

import (
 "fmt"
 "sync"
)

type Engine struct {
 mu           sync.RWMutex
 rolePerms    map[Role]map[Permission]bool
 inheritance  map[Role][]Role
 ssdConflicts map[Role][]Role
}

func NewEngine() *Engine {
 return &Engine{
  rolePerms:    make(map[Role]map[Permission]bool),
  inheritance:  make(map[Role][]Role),
  ssdConflicts: make(map[Role][]Role),
 }
}

// DefineRole associates direct permissions with a role.
func (e *Engine) DefineRole(role Role, perms ...Permission) {
 e.mu.Lock()
 defer e.mu.Unlock()

 if e.rolePerms[role] == nil {
  e.rolePerms[role] = make(map[Permission]bool)
 }
 for _, p := range perms {
  e.rolePerms[role][p] = true
 }
}

// AddInheritance defines that seniorRole inherits all permissions from juniorRole.
func (e *Engine) AddInheritance(senior Role, junior Role) {
 e.mu.Lock()
 defer e.mu.Unlock()
 e.inheritance[senior] = append(e.inheritance[senior], junior)
}

// ResolveEffectivePermissions traverses the inheritance DAG to compute all granted permissions.
func (e *Engine) ResolveEffectivePermissions(roles []Role) map[Permission]bool {
 e.mu.RLock()
 defer e.mu.RUnlock()

 effective := make(map[Permission]bool)
 visited := make(map[Role]bool)

 var walk func(r Role)
 walk = func(r Role) {
  if visited[r] {
   return
  }
  visited[r] = true

  for p := range e.rolePerms[r] {
   effective[p] = true
  }
  for _, parent := range e.inheritance[r] {
   walk(parent)
  }
 }

 for _, r := range roles {
  walk(r)
 }
 return effective
}

// HasPermission checks if the subject roles satisfy the required permission (Default Deny).
func (e *Engine) HasPermission(roles []Role, required Permission) bool {
 perms := e.ResolveEffectivePermissions(roles)
 return perms[required]
}
```

### Phase 3: Separation of Duties (SSD) Enforcement

Validate role assignments against mutually exclusive role definitions to prevent toxic combinations:

```go
func (e *Engine) RegisterSSDConflict(roleA, roleB Role) {
 e.mu.Lock()
 defer e.mu.Unlock()
 e.ssdConflicts[roleA] = append(e.ssdConflicts[roleA], roleB)
 e.ssdConflicts[roleB] = append(e.ssdConflicts[roleB], roleA)
}

func (e *Engine) ValidateAssignment(roles []Role) error {
 e.mu.RLock()
 defer e.mu.RUnlock()

 assigned := make(map[Role]bool, len(roles))
 for _, r := range roles {
  assigned[r] = true
 }

 for _, r := range roles {
  for _, conflict := range e.ssdConflicts[r] {
   if assigned[conflict] {
    return fmt.Errorf("SSD conflict violation: cannot assign both %q and %q", r, conflict)
   }
  }
 }
 return nil
}
```

### Phase 4: Context Propagation Pattern (Type-Safe)

Always encapsulate context keys within the package:

```go
package rbac

import "context"

type contextKey struct{}

var subjectKey = contextKey{}

func WithSubject(ctx context.Context, sub Subject) context.Context {
 return context.WithValue(ctx, subjectKey, sub)
}

func SubjectFromContext(ctx context.Context) (Subject, bool) {
 sub, ok := ctx.Value(subjectKey).(Subject)
 return sub, ok
}
```

### Phases 5 & 6: PEP Guard Middleware & RFC 7807 Response

Enforce access control at the route level:

```go
package rbac

import (
 "encoding/json"
 "log/slog"
 "net/http"
 "time"
)

type ProblemDetails struct {
 Type     string `json:"type"`
 Title    string `json:"title"`
 Status   int    `json:"status"`
 Detail   string `json:"detail"`
 Instance string `json:"instance"`
}

func writeRFC7807(w http.ResponseWriter, status int, title, detail, instance string) {
 w.Header().Set("Content-Type", "application/problem+json")
 w.WriteHeader(status)
 _ = json.NewEncoder(w).Encode(ProblemDetails{
  Type:     "https://golang.org/errors/forbidden",
  Title:    title,
  Status:   status,
  Detail:   detail,
  Instance: instance,
 })
}

// RequirePermission constructs an HTTP middleware PEP guard.
func RequirePermission(engine *Engine, logger *slog.Logger, required Permission) func(http.Handler) http.Handler {
 return func(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   start := time.Now()
   subject, ok := SubjectFromContext(r.Context())
   if !ok {
    logger.Warn("Unauthenticated request blocked at PEP", "path", r.URL.Path)
    writeRFC7807(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
    return
   }

   allowed := engine.HasPermission(subject.Roles, required)
   duration := time.Since(start)

   if !allowed {
    logger.Warn("RBAC access denied",
     "subject_id", subject.ID,
     "tenant_id", subject.TenantID,
     "roles", subject.Roles,
     "required_permission", required,
     "path", r.URL.Path,
     "duration_us", duration.Microseconds(),
    )
    writeRFC7807(w, http.StatusForbidden, "Forbidden", "Insufficient privileges", r.URL.Path)
    return
   }

   logger.Info("RBAC access permitted",
    "subject_id", subject.ID,
    "permission", required,
    "duration_us", duration.Microseconds(),
   )

   next.ServeHTTP(w, r)
  })
 }
}
```

### Phase 7: Audit Trail & Revocation Strategy

- **Audit Slog Payload**: Always log `subject_id`, `tenant_id`, `roles`, `required_permission`, `client_ip`, and `decision`.
- **Token Invalidation on Role Change**:
  When a user's role assignment is modified, access tokens must not grant stale privileges. Implement either:
  1. **Short-lived access tokens** ($\le 15\text{ minutes}$) paired with secure refresh tokens.
  2. **Token Versioning / Serial Check**: Embed a `token_version` in the JWT claims and check against a cache on mutating requests.
  3. **Event-driven In-Memory Cache Invalidation**: Emit a message on an in-memory or distributed event bus to flush cached subject permissions.

---

## Anti-Patterns to Avoid

| Anti-Pattern | Security Risk | Production Remediation |
| :--- | :--- | :--- |
| **String Literal Context Keys** (`ctx.Value("user")`) | Third-party packages can overwrite or collide with security context | Use unexported struct keys (`type contextKey struct{}`) |
| **Checking Roles in Handlers** (`if role == "Admin"`) | Fragile role coupling, prevents granular privilege delegation | Check fine-grained permissions (`HasPermission(perms, "invoices:approve")`) |
| **Direct DB Lookup per Request** | Severe database saturation and latency spikes under load | Cache policies in memory with `sync.RWMutex` or `atomic.Pointer` |
| **Leaking Permission Details in 403** | Attackers enumerate system capabilities and attack vectors | Return generic detail (`"Insufficient privileges"`) via RFC 7807 |
| **Ignoring Race Conditions** | Data corruption or authorization bypass during concurrent updates | Always run tests with `go test -race` |
| **Permissive Default Fallback** | New or unmapped routes inadvertently become public | Implement strict `Default Deny` (Fail-Closed) |

---

## Production Security Verification Checklist

- [ ] **Default Deny**: Unmapped permissions or unauthenticated contexts reject immediately with 401/403.
- [ ] **Thread-Safe DAG**: Role inheritance traversal is read-locked and protected against cycles.
- [ ] **Unexported Context Keys**: All context values use unexported private key types.
- [ ] **SSD Mutual Exclusion**: Conflicting roles are rejected at assignment time.
- [ ] **RFC 7807 Compliance**: Forbidden responses use `application/problem+json` format.
- [ ] **Zero Sensitive Information Leakage**: Error responses do not disclose required permission keys.
- [ ] **Race Detector Clean**: Passes `go test -race ./...` without data races.
- [ ] **Fuzz Tested**: Role and permission parsers pass `go test -fuzz`.
- [ ] **Vulnerability Verified**: Clean report from `govulncheck ./...`.
- [ ] **Revocation Strategy**: Mechanism exists to invalidate active sessions when roles change.
