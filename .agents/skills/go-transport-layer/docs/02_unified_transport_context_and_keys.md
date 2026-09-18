# Unified TransportContext and Collision-Free Context Keying

This document outlines the engineering specification for the `TransportContext` interface, concrete implementations for HTTP and gRPC, and the mechanics of type-safe, unexported context propagation in Go.

---

## 1. The `TransportContext` Interface Design

The `TransportContext` interface provides a normalized, protocol-neutral view of an incoming network request. It enables security validators, telemetry injectors, and rate limiters to inspect incoming requests without importing `net/http` or `google.golang.org/grpc/metadata`.

```go
package transport

import "context"

type TransportContext interface {
 // Protocol identifies whether the request arrived via HTTP or gRPC.
 Protocol() TransportProtocol

 // Context returns the standard Go context carrying deadlines and cancellation signals.
 Context() context.Context

 // RequestID returns the distributed trace / correlation identifier.
 RequestID() string

 // ClientIP returns the canonical remote IP address after resolving proxy headers.
 ClientIP() string

 // UserAgent returns the client software signature.
 UserAgent() string

 // Header retrieves a normalized, case-insensitive metadata value.
 Header(key string) string

 // Principal returns the authenticated caller identity, if resolved.
 Principal() (*SecurityPrincipal, bool)

 // SetPrincipal stores the authenticated identity after successful verification.
 SetPrincipal(principal *SecurityPrincipal)
}
```

---

## 2. Concrete Ingress Implementations

### 2.1 HTTP Transport Context (`httpTransportContext`)

The HTTP implementation wraps standard `http.ResponseWriter` and `*http.Request`:

```go
package transport

import (
 "context"
 "net"
 "net/http"
 "strings"
)

type httpTransportContext struct {
 req       *http.Request
 requestID string
 principal *SecurityPrincipal
}

func NewHTTPTransportContext(req *http.Request, requestID string) TransportContext {
 return &httpTransportContext{
  req:       req,
  requestID: requestID,
 }
}

func (h *httpTransportContext) Protocol() TransportProtocol {
 return ProtocolHTTP
}

func (h *httpTransportContext) Context() context.Context {
 return h.req.Context()
}

func (h *httpTransportContext) RequestID() string {
 return h.requestID
}

func (h *httpTransportContext) ClientIP() string {
 // 1. Check standard proxy headers (Forwarded, X-Forwarded-For, X-Real-IP)
 if xff := h.req.Header.Get("X-Forwarded-For"); xff != "" {
  parts := strings.Split(xff, ",")
  return strings.TrimSpace(parts[0])
 }
 if xrip := h.req.Header.Get("X-Real-IP"); xrip != "" {
  return strings.TrimSpace(xrip)
 }

 // 2. Fall back to direct TCP RemoteAddr
 host, _, err := net.SplitHostPort(h.req.RemoteAddr)
 if err != nil {
  return h.req.RemoteAddr
 }
 return host
}

func (h *httpTransportContext) UserAgent() string {
 return h.req.UserAgent()
}

func (h *httpTransportContext) Header(key string) string {
 return h.req.Header.Get(key)
}

func (h *httpTransportContext) Principal() (*SecurityPrincipal, bool) {
 if h.principal != nil {
  return h.principal, true
 }
 return GetPrincipal(h.req.Context())
}

func (h *httpTransportContext) SetPrincipal(principal *SecurityPrincipal) {
 h.principal = principal
 *h.req = *h.req.WithContext(WithPrincipal(h.req.Context(), principal))
}
```

### 2.2 gRPC Transport Context (`grpcTransportContext`)

The gRPC implementation wraps `context.Context`, `metadata.MD`, and `peer.Peer`:

```go
package transport

import (
 "context"
 "net"
 "strings"

 "google.golang.org/grpc/metadata"
 "google.golang.org/grpc/peer"
)

type grpcTransportContext struct {
 ctx       context.Context
 md        metadata.MD
 requestID string
 principal *SecurityPrincipal
}

func NewGRPCTransportContext(ctx context.Context, requestID string) TransportContext {
 md, ok := metadata.FromIncomingContext(ctx)
 if !ok {
  md = metadata.New(nil)
 }
 return &grpcTransportContext{
  ctx:       ctx,
  md:        md,
  requestID: requestID,
 }
}

func (g *grpcTransportContext) Protocol() TransportProtocol {
 return ProtocolGRPC
}

func (g *grpcTransportContext) Context() context.Context {
 return g.ctx
}

func (g *grpcTransportContext) RequestID() string {
 return g.requestID
}

func (g *grpcTransportContext) ClientIP() string {
 // 1. Inspect metadata for proxy forwarding
 if vals := g.md.Get("x-forwarded-for"); len(vals) > 0 {
  parts := strings.Split(vals[0], ",")
  return strings.TrimSpace(parts[0])
 }
 if vals := g.md.Get("x-real-ip"); len(vals) > 0 {
  return strings.TrimSpace(vals[0])
 }

 // 2. Extract TCP peer from context
 if p, ok := peer.FromContext(g.ctx); ok && p.Addr != nil {
  host, _, err := net.SplitHostPort(p.Addr.String())
  if err == nil {
   return host
  }
  return p.Addr.String()
 }
 return "unknown"
}

func (g *grpcTransportContext) UserAgent() string {
 if vals := g.md.Get("user-agent"); len(vals) > 0 {
  return vals[0]
 }
 return "grpc-client"
}

func (g *grpcTransportContext) Header(key string) string {
 // gRPC metadata keys are strictly lowercase
 vals := g.md.Get(strings.ToLower(key))
 if len(vals) > 0 {
  return vals[0]
 }
 return ""
}

func (g *grpcTransportContext) Principal() (*SecurityPrincipal, bool) {
 if g.principal != nil {
  return g.principal, true
 }
 return GetPrincipal(g.ctx)
}

func (g *grpcTransportContext) SetPrincipal(principal *SecurityPrincipal) {
 g.principal = principal
 g.ctx = WithPrincipal(g.ctx, principal)
}
```

---

## 3. Collision-Free Context Keying Architecture

In Go, `context.Context` stores values as an association between `any` key and `any` value. If two packages use a string `"user_id"` or an exported integer as a key, one will overwrite the other, causing silent bugs or security vulnerabilities.

### The Unexported Type Idiom

To ensure mathematical impossibility of collisions:

1. Define an unexported custom type: `type contextKey int` or `type contextKey struct{}`.
2. Declare unexported constants or variables of that type.
3. Expose only typed getter and setter helper functions.

```go
package transport

import "context"

type contextKey int

const (
 transportInfoKey contextKey = iota
 principalKey
)

func WithTransportInfo(ctx context.Context, info TransportInfo) context.Context {
 return context.WithValue(ctx, transportInfoKey, info)
}

func GetTransportInfo(ctx context.Context) (TransportInfo, bool) {
 info, ok := ctx.Value(transportInfoKey).(TransportInfo)
 return info, ok
}

func WithPrincipal(ctx context.Context, p *SecurityPrincipal) context.Context {
 return context.WithValue(ctx, principalKey, p)
}

func GetPrincipal(ctx context.Context) (*SecurityPrincipal, bool) {
 p, ok := ctx.Value(principalKey).(*SecurityPrincipal)
 return p, ok && p != nil
}
```

### Memory Allocation Considerations

- `TransportInfo` is a compact 4-field struct (~64 bytes). It is copied directly into `context.WithValue` with negligible heap allocation.
- `SecurityPrincipal` is stored as a pointer `*SecurityPrincipal`. This ensures zero copy overhead across middleware chains and prevents stale identity states.
