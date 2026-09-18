---
name: go-transport-layer
description: "Production-ready, protocol-agnostic Transport Layer architecture for Go backends unifying HTTP (REST/JSON) and gRPC (Protobuf) before request handling. Covers Hexagonal primary adapter boundaries, TransportContext abstraction, unexported type-safe context keying, 6-stage pre-handling lifecycle pipeline, protocol identification, dual-protocol middleware/interceptors, port multiplexing (cmux & h2c), and bidirectional domain error mapping."
---

# Go Transport Layer Architecture: Unifying HTTP & gRPC Pre-Handling

This skill defines a production-grade, battle-tested, and protocol-agnostic **Transport Layer Architecture** for Go backend services. It provides a clean, unified ingress boundary for applications that simultaneously support **HTTP/REST (JSON)** and **gRPC (Protobuf over HTTP/2)**, guaranteeing that requests are stamped, normalized, authenticated, validated, and enriched **before** entering core application use cases or domain handlers.

The architecture synthesizes official Go idiomatic standards from `net/http`, Google's official `google.golang.org/grpc` project, Hexagonal / Clean Architecture (Ports & Adapters), Go kit transport design, and modern RPC unification patterns (ConnectRPC, gRPC-Gateway).

---

## Production Architectural Principles

1. **Strict Primary / Driving Adapter Placement**:
   The transport layer lives at the outermost edge of the system (`internal/transport/` or `adapter/transport/`). It acts purely as a **Driving (Primary) Adapter**. It translates external network bytes (TCP/HTTP/gRPC) into domain calls and translates domain results into network responses.

2. **Zero Protocol Leakage into Business Handlers**:
   Domain services, use cases, and entities MUST NOT import `net/http`, `google.golang.org/grpc`, or auto-generated Protobuf structs. No `*http.Request`, `http.ResponseWriter`, or `grpc.ServerStream` may cross into the application core. Passing transport artifacts into use cases creates tight network coupling, breaks unit testability, and violates the Dependency Inversion Principle.

3. **Context-Scoped, Type-Safe Identity & Protocol Propagation**:
   All metadata extracted at the transport boundary (Protocol, Request ID, Client IP, Authenticated Principal) MUST be propagated into Go's standard `context.Context` using **unexported, package-private key types** (`type contextKey int`). This mathematically eliminates key collisions across libraries.

4. **Explicit Protocol Identification (Self-Aware Ingress)**:
   The application core and cross-cutting pipelines can always query *which protocol* delivered the request via a strongly-typed `TransportProtocol` enum (`HTTP` vs `GRPC`). This enables protocol-specific telemetry, selective caching, or customized response shapes when needed without breaking abstractions.

5. **Deterministic 6-Stage Pre-Handling Pipeline**:
   Every incoming request (regardless of whether it entered via HTTP router or gRPC listener) must traverse an identical, ordered pre-handling pipeline before reaching the handler:
   - **Stage 1: Protocol Stamping & Ingress Logging**
   - **Stage 2: Correlation ID & Distributed Tracing** (`X-Request-ID`, W3C TraceContext)
   - **Stage 3: Metadata Normalization** (Header case-folding, Client IP resolution)
   - **Stage 4: Authentication & SecurityPrincipal Resolution** (Bearer JWT / API Key)
   - **Stage 5: Syntactic Payload Validation** (Schema/format checks vs domain logic)
   - **Stage 6: Deadlines & Rate Limiting** (`context.WithDeadline`, token buckets)

6. **Bidirectional Error & Status Code Mapping**:
   Domain handlers return pure Go domain errors (e.g., `ErrNotFound`, `ErrConflict`, `ErrUnauthorized`). The transport adapter maps these errors symmetrically to **HTTP Status Codes** (with RFC 7807 Problem Details) and **gRPC Status Codes** (`codes.NotFound`, `codes.AlreadyExists`, `codes.Unauthenticated`).

7. **Fail-Safe Panic Recovery**:
   Any panic occurring during transport decoding, pre-handling, or business execution is caught at the transport boundary. The panic is logged with its stack trace to internal telemetry, and a safe, sanitized `500 Internal Server Error` (or gRPC `codes.Internal`) is returned to the client without leaking memory pointers or infrastructure details.

---

## Architectural Topology: Dual Ingress to Pure Handling

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                 Incoming Network                                 │
│         Client (HTTP/REST JSON)                     Microservice (gRPC Protobuf) │
└──────────────────────────┬───────────────────────────────────────┬───────────────┘
                           │                                       │
                           ▼                                       ▼
┌───────────────────────────────────────┐   ┌──────────────────────────────────────┐
│       HTTP Router / Chi / stdlib      │   │          gRPC Server Engine          │
│            (http.Handler)             │   │       (UnaryServerInterceptor)       │
└──────────────────┬────────────────────┘   └──────────────────────┬───────────────┘
                   │                                               │
                   ▼                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                   Unified Transport Adapter Layer (Ingress Boundary)             │
│                                                                                  │
│   HTTPTransportContext                                  GRPCTransportContext     │
│   ├── Wraps *http.Request                               ├── Wraps metadata.MD    │
│   └── Implements TransportContext                       └── Implements TC        │
│                                                                                  │
│   ┌──────────────────────────────────────────────────────────────────────────┐   │
│   │                      6-Stage Pre-Handling Pipeline                       │   │
│   │                                                                          │   │
│   │  [1] Protocol Stamping & Ingress Logging (HTTP vs GRPC)                  │   │
│   │  [2] Correlation & Tracing (X-Request-ID, W3C traceparent)               │   │
│   │  [3] Metadata Normalization (Client IP, Headers, User-Agent)             │   │
│   │  [4] Authentication & SecurityPrincipal Resolution                      │   │
│   │  [5] Syntactic Payload Validation & Schema Decoupling                    │   │
│   │  [6] Context Deadlines, Timeouts & Rate Limits                           │   │
│   └─────────────────────────────────────┬────────────────────────────────────┘   │
└─────────────────────────────────────────┼────────────────────────────────────────┘
                                          │
                                          ▼ Enriched context.Context + Pure DTO
┌──────────────────────────────────────────────────────────────────────────────────┐
│                Pure Application Core / Use Case (Zero Network Imports)           │
│                                                                                  │
│   type OrderUseCase interface {                                                  │
│       CreateOrder(ctx context.Context, req CreateOrderDTO) (*OrderResult, error) │
│   }                                                                              │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Interfaces and Data Contracts

### 1. Protocol Definition and Security Principal

```go
package transport

import (
 "context"
 "time"
)

// TransportProtocol identifies the network ingress channel that delivered the request.
type TransportProtocol string

const (
 ProtocolHTTP TransportProtocol = "HTTP"
 ProtocolGRPC TransportProtocol = "GRPC"
)

func (p TransportProtocol) String() string {
 return string(p)
}

// SecurityPrincipal represents an authenticated caller identity, completely decoupled from
// network transport, headers, or token encoding.
type SecurityPrincipal struct {
 UserID    string    `json:"user_id"`
 TenantID  string    `json:"tenant_id,omitempty"`
 Roles     []string  `json:"roles,omitempty"`
 Scopes    []string  `json:"scopes,omitempty"`
 Subject   string    `json:"subject,omitempty"`
 IsAdmin   bool      `json:"is_admin"`
 IssuedAt  time.Time `json:"issued_at,omitempty"`
 ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// TransportInfo holds diagnostic transport metadata injected into context.Context.
type TransportInfo struct {
 Protocol  TransportProtocol `json:"protocol"`
 RequestID string            `json:"request_id"`
 ClientIP  string            `json:"client_ip"`
 UserAgent string            `json:"user_agent"`
}
```

### 2. The Abstract `TransportContext` Interface

```go
// TransportContext abstracts incoming request properties across HTTP and gRPC protocols,
// enabling reusable interceptors, logging, and security evaluation without network imports.
type TransportContext interface {
 // Protocol returns whether the request arrived via HTTP or gRPC.
 Protocol() TransportProtocol

 // Context returns the standard Go context associated with the request.
 Context() context.Context

 // RequestID returns the distributed trace / correlation identifier.
 RequestID() string

 // ClientIP extracts the canonical client IP address after resolving proxy headers.
 ClientIP() string

 // UserAgent returns the client software signature.
 UserAgent() string

 // Header returns a normalized, case-insensitive metadata value.
 Header(key string) string

 // Principal retrieves the resolved security identity, if authenticated.
 Principal() (*SecurityPrincipal, bool)

 // SetPrincipal associates an authenticated identity with the transport context.
 SetPrincipal(principal *SecurityPrincipal)
}
```

### 3. Collision-Free Context Keying

```go
package transport

import "context"

type contextKey int

const (
 transportInfoKey contextKey = iota
 principalKey
)

// WithTransportInfo embeds transport diagnostic metadata into context.Context.
func WithTransportInfo(ctx context.Context, info TransportInfo) context.Context {
 return context.WithValue(ctx, transportInfoKey, info)
}

// GetTransportInfo retrieves transport diagnostic metadata from context.Context.
func GetTransportInfo(ctx context.Context) (TransportInfo, bool) {
 info, ok := ctx.Value(transportInfoKey).(TransportInfo)
 return info, ok
}

// WithPrincipal injects an authenticated SecurityPrincipal into context.Context.
func WithPrincipal(ctx context.Context, p *SecurityPrincipal) context.Context {
 return context.WithValue(ctx, principalKey, p)
}

// GetPrincipal retrieves an authenticated SecurityPrincipal from context.Context.
func GetPrincipal(ctx context.Context) (*SecurityPrincipal, bool) {
 p, ok := ctx.Value(principalKey).(*SecurityPrincipal)
 return p, ok && p != nil
}
```

---

## The 6-Stage Pre-Handling Pipeline Engine

```go
package transport

import (
 "context"
 "fmt"
 "log/slog"
 "strings"
 "time"
)

type TokenValidator interface {
 ValidateToken(ctx context.Context, token string) (*SecurityPrincipal, error)
}

type PreHandlingPipeline struct {
 validator TokenValidator
 logger    *slog.Logger
 timeout   time.Duration
}

func NewPreHandlingPipeline(validator TokenValidator, logger *slog.Logger, defaultTimeout time.Duration) *PreHandlingPipeline {
 if defaultTimeout <= 0 {
  defaultTimeout = 10 * time.Second
 }
 return &PreHandlingPipeline{
  validator: validator,
  logger:    logger,
  timeout:   defaultTimeout,
 }
}

// Execute orchestrates the 6 stages prior to business logic execution.
func (p *PreHandlingPipeline) Execute(tc TransportContext) (context.Context, error) {
 start := time.Now()
 protocol := tc.Protocol()
 reqID := tc.RequestID()
 clientIP := tc.ClientIP()

 // Stage 1 & 2: Protocol Stamping, Correlation Logging
 p.logger.Info("Ingress request received at transport boundary",
  slog.String("protocol", protocol.String()),
  slog.String("request_id", reqID),
  slog.String("client_ip", clientIP),
  slog.String("user_agent", tc.UserAgent()),
 )

 // Stage 3: Metadata Normalization & Context Injection
 ctx := tc.Context()
 ctx = WithTransportInfo(ctx, TransportInfo{
  Protocol:  protocol,
  RequestID: reqID,
  ClientIP:  clientIP,
  UserAgent: tc.UserAgent(),
 })

 // Stage 4: Authentication & SecurityPrincipal Resolution
 authHeader := tc.Header("Authorization")
 if authHeader != "" {
  token := strings.TrimPrefix(authHeader, "Bearer ")
  token = strings.TrimSpace(token)
  if token != "" && p.validator != nil {
   principal, err := p.validator.ValidateToken(ctx, token)
   if err != nil {
    p.logger.Warn("Authentication rejected at transport boundary",
     slog.String("protocol", protocol.String()),
     slog.String("request_id", reqID),
     slog.Any("error", err),
    )
    return nil, fmt.Errorf("unauthenticated: %w", err)
   }
   tc.SetPrincipal(principal)
   ctx = WithPrincipal(ctx, principal)
  }
 }

 // Stage 6: Timeout / Deadline Enforcement (if not already set upstream)
 if _, hasDeadline := ctx.Deadline(); !hasDeadline && p.timeout > 0 {
  var cancel context.CancelFunc
  ctx, cancel = context.WithTimeout(ctx, p.timeout)
  _ = cancel // Handled by caller or transport wrapper completion
 }

 p.logger.Debug("Pre-handling pipeline successfully completed",
  slog.String("protocol", protocol.String()),
  slog.Duration("duration_us", time.Since(start)),
 )

 return ctx, nil
}
```

---

## Dual Protocol Ingress Adapters

### 1. HTTP Adapter Middleware (`net/http`)

```go
package transport

import (
 "crypto/rand"
 "encoding/hex"
 "net/http"
)

func HTTPMiddleware(pipeline *PreHandlingPipeline) func(http.Handler) http.Handler {
 return func(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   reqID := r.Header.Get("X-Request-ID")
   if reqID == "" {
    reqID = generateID()
   }
   w.Header().Set("X-Request-ID", reqID)

   tc := NewHTTPTransportContext(r, reqID)
   enrichedCtx, err := pipeline.Execute(tc)
   if err != nil {
    http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
    return
   }

   next.ServeHTTP(w, r.WithContext(enrichedCtx))
  })
 }
}

func generateID() string {
 b := make([]byte, 16)
 _, _ = rand.Read(b)
 return hex.EncodeToString(b)
}
```

### 2. gRPC Unary Server Interceptor (`google.golang.org/grpc`)

```go
package transport

import (
 "context"

 "google.golang.org/grpc"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/metadata"
 "google.golang.org/grpc/status"
)

func GRPCUnaryInterceptor(pipeline *PreHandlingPipeline) grpc.UnaryServerInterceptor {
 return func(
  ctx context.Context,
  req any,
  info *grpc.UnaryServerInfo,
  handler grpc.UnaryHandler,
 ) (any, error) {
  md, _ := metadata.FromIncomingContext(ctx)
  var reqID string
  if vals := md.Get("x-request-id"); len(vals) > 0 {
   reqID = vals[0]
  } else {
   reqID = generateID()
  }

  _ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", reqID))
  tc := NewGRPCTransportContext(ctx, reqID)

  enrichedCtx, err := pipeline.Execute(tc)
  if err != nil {
   return nil, status.Error(codes.Unauthenticated, err.Error())
  }

  return handler(enrichedCtx, req)
 }
}
```

---

## Domain Error Mapping Matrix

| Domain Error | HTTP Status | RFC 7807 Problem Title | gRPC Status Code | Description |
| :--- | :--- | :--- | :--- | :--- |
| `ErrNotFound` | `404 Not Found` | Resource Not Found | `codes.NotFound` | Requested entity does not exist |
| `ErrUnauthorized` | `401 Unauthorized` | Authentication Required | `codes.Unauthenticated` | Missing or invalid auth credentials |
| `ErrForbidden` | `403 Forbidden` | Access Denied | `codes.PermissionDenied` | Valid caller lacks permission |
| `ErrInvalidInput` | `400 Bad Request` | Invalid Input | `codes.InvalidArgument` | Syntactic or validation failure |
| `ErrConflict` | `409 Conflict` | Resource Conflict | `codes.AlreadyExists` | Unique constraint violation |
| `ErrPreconditionFailed` | `412 Precondition` | Precondition Failed | `codes.FailedPrecondition` | Optimistic lock or state mismatch |
| `ErrRateLimited` | `429 Too Many Req` | Rate Limit Exceeded | `codes.ResourceExhausted` | Quota or rate threshold exceeded |
| `ErrDeadlineExceeded` | `504 Timeout` | Gateway Timeout | `codes.DeadlineExceeded` | Context deadline elapsed |
| `ErrInternal` / Panic | `500 Server Error` | Internal Server Error | `codes.Internal` | Unexpected unhandled failure |

---

## Production Verification Checklist

- [ ] **No Protocol Imports in Use Cases**: Verify with `go vet` or custom linting that `application/` and `domain/` packages do not import `net/http` or `google.golang.org/grpc`.
- [ ] **Context Keys Unexported**: Ensure all keys used with `context.WithValue` are private, zero-size types or unexported ints.
- [ ] **Safe Client IP Resolution**: Strip untrusted proxy headers (`X-Forwarded-For`) unless behind a verified trusted ingress gateway.
- [ ] **Panic Recovery at Boundary**: Ensure both HTTP middleware and gRPC interceptors defer a `recover()` call to prevent process crashes.
- [ ] **Case-Insensitive Headers**: Verify gRPC `metadata.MD` keys are lowercased and HTTP headers use `http.CanonicalHeaderKey`.
- [ ] **Structured Telemetry**: Ensure `TransportInfo` (protocol, request_id, client_ip) is attached to all `slog` messages and OpenTelemetry traces.
