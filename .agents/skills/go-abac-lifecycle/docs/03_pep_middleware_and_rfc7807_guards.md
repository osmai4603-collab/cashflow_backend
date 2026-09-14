# PEP Middleware and RFC 7807 Problem Details Guards in Go

This document details the design and implementation of the **Policy Enforcement Point (PEP)** in Go HTTP and gRPC services, focusing on collision-free context propagation, fail-closed enforcement guards, and standardized **RFC 7807 Problem Details** error formatting.

---

## 1. The Role of the Policy Enforcement Point (PEP)

The PEP protects application boundaries by:

1. **Intercepting** incoming requests before any domain or business handler executes.
2. **Extracting & Propagating** caller identity and environmental context.
3. **Coordinating** with the PDP (and PIP if dynamic attributes are required).
4. **Enforcing** the decision: allowing execution to proceed (`200 OK` path) or terminating execution immediately with a safe `403 Forbidden` response.

```text
Incoming HTTP/gRPC Request
           │
           ▼
┌──────────────────────────────────────────────────────────┐
│ PEP Authentication Middleware (AuthN)                    │
│ 1. Validate JWT / mTLS / Session                         │
│ 2. Construct Subject & Environment                       │
│ 3. Inject into request context (unexported contextKey)   │
└──────────────────────────┬───────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────┐
│ PEP Authorization Guard (AuthZ)                          │
│ 1. Extract Subject & Env from ctx                        │
│ 2. PIP Lookup: Resolve target Resource attributes        │
│ 3. PDP Evaluate: Run combining algorithm                 │
└──────────────────────────┬───────────────────────────────┘
                           │
             ┌─────────────┴─────────────┐
             │                           │
      [DecisionPermit]            [DecisionDeny]
             │                           │
             ▼                           ▼
┌─────────────────────────┐ ┌──────────────────────────────┐
│ Call next.ServeHTTP()   │ │ Return RFC 7807 Problem (403)│
│ (Execute Domain Action) │ │ (Terminate pipeline safely)  │
└─────────────────────────┘ └──────────────────────────────┘
```

---

## 2. Type-Safe, Collision-Free Context Propagation

In Go, `context.Context` accepts `any` as key and value. Using raw string keys (e.g., `"subject"`) invites key collisions between packages or third-party middleware.

### The Idiomatic Go Standard

Define an **unexported struct type** specifically for context keys:

```go
package abac

import "context"

// contextKey is unexported, making collisions outside this package impossible.
type contextKey struct{}

var (
    subjectKey     = contextKey{}
    environmentKey = contextKey{}
)

// WithSubject stores the authenticated Subject in the context.
func WithSubject(ctx context.Context, sub Subject) context.Context {
    return context.WithValue(ctx, subjectKey, sub)
}

// GetSubject extracts the Subject from the context.
func GetSubject(ctx context.Context) (Subject, bool) {
    sub, ok := ctx.Value(subjectKey).(Subject)
    return sub, ok
}

// WithEnvironment stores the Environment in the context.
func WithEnvironment(ctx context.Context, env Environment) context.Context {
    return context.WithValue(ctx, environmentKey, env)
}

// GetEnvironment extracts the Environment from the context.
func GetEnvironment(ctx context.Context) (Environment, bool) {
    env, ok := ctx.Value(environmentKey).(Environment)
    return env, ok
}
```

---

## 3. RFC 7807 Problem Details Rejection Guard

When an access check fails, the API must return a structured, standardized JSON response conforming to **RFC 7807** (*Problem Details for HTTP APIs*).

### 3.1 Problem Details Schema

```go
package abac

import (
    "encoding/json"
    "net/http"
)

// ProblemDetails models an RFC 7807 problem payload.
type ProblemDetails struct {
    Type     string `json:"type"`               // URI reference identifying the problem type
    Title    string `json:"title"`              // Short human-readable summary
    Status   int    `json:"status"`             // HTTP status code (403)
    Detail   string `json:"detail"`             // Human-readable explanation of this specific occurrence
    Instance string `json:"instance,omitempty"` // URI reference identifying the specific request path
}

// WriteForbiddenProblem sends a standard RFC 7807 response with 403 Forbidden.
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

### 3.2 Information Disclosure Prevention

> [!CAUTION]
> **Never leak internal policy details**:
> An attacker can map out the security system if error messages state:
> `"Denied: Subject clearance 2 is less than Resource sensitivity 5, and Department accounting does not match finance"`.
>
> In production:
>
> - **External Response (Public RFC 7807)**: `"Access denied: insufficient privileges to access this resource"`.
> - **Internal Audit Log (`log/slog`)**: Detailed reason including failed rule ID, clearance, department, and tenant match status.

---

## 4. Reusable HTTP PEP Middleware Implementation

Here is a generic, reusable HTTP middleware function enforcing ABAC:

```go
type ResourceExtractor func(r *http.Request) (Resource, error)

func RequireABAC(pdp *Engine, verb string, extractResource ResourceExtractor) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Get Subject from Context (injected by upstream AuthN middleware)
            subject, ok := GetSubject(r.Context())
            if !ok {
                WriteForbiddenProblem(w, r, "Access denied: unauthenticated subject context")
                return
            }

            // 2. Resolve target Resource
            resource, err := extractResource(r)
            if err != nil {
                WriteForbiddenProblem(w, r, "Access denied: unable to resolve target resource")
                return
            }

            // 3. Resolve Environment (from ctx or current request)
            env, ok := GetEnvironment(r.Context())
            if !ok {
                env = Environment{
                    RequestTime: time.Now(),
                    ClientIP:    r.RemoteAddr,
                }
            }

            // 4. Construct EvaluationContext
            evalCtx := EvaluationContext{
                Subject:     subject,
                Resource:    resource,
                Action:      Action{Verb: verb, Method: r.Method},
                Environment: env,
            }

            // 5. Evaluate PDP
            decision, _ := pdp.Evaluate(evalCtx)
            if decision != DecisionPermit {
                // Fail-Closed: return 403 Forbidden without leaking internal rule names
                WriteForbiddenProblem(w, r, "Access denied: you do not have permission to perform this action on this resource")
                return
            }

            // 6. Access permitted -> delegate to downstream handler
            next.ServeHTTP(w, r)
        })
    }
}
```
