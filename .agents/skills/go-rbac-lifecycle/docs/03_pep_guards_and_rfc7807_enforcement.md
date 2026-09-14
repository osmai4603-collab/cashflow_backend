# PEP Guards and RFC 7807 Enforcement Architecture

This document describes how to construct resilient Policy Enforcement Points (PEPs) in Go, covering HTTP middlewares, gRPC interceptors, and secure RFC 7807 Problem Details error formatting.

---

## 1. The Dual-Tier Enforcement Strategy

In clean Go architecture, access control is enforced at two distinct tiers:

```text
Incoming Request
       │
       ▼
┌────────────────────────────────────────────────────────┐
│ Tier 1: Protocol PEP Guard (HTTP Middleware / gRPC)    │
│ Scope: Coarse-Grained Route Gating                     │
│ Check: Does the subject hold "invoices:create"?        │
└──────────────────────────┬─────────────────────────────┘
                           │ (Allowed)
                           ▼
┌────────────────────────────────────────────────────────┐
│ Tier 2: Domain / Use Case Enforcement                  │
│ Scope: Fine-Grained Object & Tenant Boundary Check     │
│ Check: Does subject's tenant match invoice.company_id? │
└────────────────────────────────────────────────────────┘
```

---

## 2. HTTP PEP Middleware Implementation

A production-ready HTTP middleware must:

1. Extract and validate identity from `r.Context()`.
2. Evaluate permissions via the in-memory engine.
3. Handle failure securely without information disclosure.

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

func WriteProblemDetails(w http.ResponseWriter, status int, title, detail, instance string) {
 w.Header().Set("Content-Type", "application/problem+json")
 w.WriteHeader(status)
 _ = json.NewEncoder(w).Encode(ProblemDetails{
  Type:     "https://golang.org/errors/access-denied",
  Title:    title,
  Status:   status,
  Detail:   detail,
  Instance: instance,
 })
}

func Guard(engine *Engine, logger *slog.Logger, required Permission) func(http.Handler) http.Handler {
 return func(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   start := time.Now()
   subject, ok := SubjectFromContext(r.Context())
   if !ok {
    WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
    return
   }

   if !engine.HasPermission(subject.Roles, required) {
    logger.Warn("RBAC access denied",
     "subject_id", subject.ID,
     "roles", subject.Roles,
     "required_permission", required,
     "path", r.URL.Path,
     "latency_us", time.Since(start).Microseconds(),
    )
    // Anti-leakage: Do NOT state "Missing permission: invoices:delete"
    WriteProblemDetails(w, http.StatusForbidden, "Forbidden", "You do not have permission to access this resource", r.URL.Path)
    return
   }

   next.ServeHTTP(w, r)
  })
 }
}
```

---

## 3. gRPC Unary Interceptor PEP Guard

For gRPC microservices, implement a Unary Server Interceptor:

```go
package rbac

import (
 "context"
 "google.golang.org/grpc"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/status"
)

type MethodPermissionMap map[string]Permission

func UnaryServerInterceptor(engine *Engine, methodPerms MethodPermissionMap) grpc.UnaryServerInterceptor {
 return func(
  ctx context.Context,
  req any,
  info *grpc.UnaryServerInfo,
  handler grpc.UnaryHandler,
 ) (any, error) {
  required, exists := methodPerms[info.FullMethod]
  if !exists {
   // Fail-Closed: If method is not registered, deny by default
   return nil, status.Errorf(codes.PermissionDenied, "access denied by default policy")
  }

  subject, ok := SubjectFromContext(ctx)
  if !ok {
   return nil, status.Errorf(codes.Unauthenticated, "unauthenticated caller")
  }

  if !engine.HasPermission(subject.Roles, required) {
   return nil, status.Errorf(codes.PermissionDenied, "insufficient privileges")
  }

  return handler(ctx, req)
 }
}
```
