# Port Multiplexing and Bidirectional Error Mapping

This document covers two critical transport concerns in Go: serving both HTTP and gRPC traffic on a single TCP port (Port Multiplexing) and translating internal domain errors into protocol-specific status codes (HTTP Status Codes & RFC 7807 vs gRPC Status Codes).

---

## 1. Port Multiplexing Strategies

In containerized environments (Kubernetes, AWS ECS, Cloud Run) and microservice fabrics, listening on a single network port simplifies ingress routing, DNS, and load balancer configuration.

### Strategy A: Connection Multiplexing with `cmux`

`cmux` inspects the initial bytes of an incoming TCP connection to determine the protocol before dispatching the socket to either the gRPC or HTTP server.

```go
package main

import (
 "log"
 "net"
 "net/http"

 "github.com/soheilhy/cmux"
 "google.golang.org/grpc"
)

func RunMultiplexedServer(addr string, grpcSrv *grpc.Server, httpHandler http.Handler) error {
 listener, err := net.Listen("tcp", addr)
 if err != nil {
  return err
 }

 m := cmux.New(listener)

 // 1. gRPC uses HTTP/2 with Content-Type: application/grpc
 grpcL := m.MatchWithWriters(
  cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
 )

 // 2. HTTP/1.1 traffic matches standard HTTP methods
 httpL := m.Match(cmux.HTTP1Fast())

 go func() {
  if err := grpcSrv.Serve(grpcL); err != nil {
   log.Printf("gRPC server error: %v", err)
  }
 }()

 httpSrv := &http.Server{Handler: httpHandler}
 go func() {
  if err := httpSrv.Serve(httpL); err != nil && err != http.ErrServerClosed {
   log.Printf("HTTP server error: %v", err)
  }
 }()

 log.Printf("Multiplexed server active on %s", addr)
 return m.Serve()
}
```

### Strategy B: Native HTTP/2 Multiplexing via `net/http`

When running over TLS or HTTP/2 cleartext (h2c), `grpc.Server.ServeHTTP` can be invoked directly from a top-level `http.Handler`:

```go
func UnifiedHandler(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
 return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
   grpcServer.ServeHTTP(w, r)
  } else {
   httpHandler.ServeHTTP(w, r)
  }
 })
}
```

> [!TIP]
> **Production Recommendation**:
> While port multiplexing is valuable for cloud environments with single-port constraints, deploying dual independent listeners (e.g., `:8080` for HTTP/REST and `:9090` for gRPC) is preferred for high-throughput mission-critical services. It prevents HTTP/1.1 keep-alive starvation from affecting gRPC HTTP/2 stream multiplexing.

---

## 2. Bidirectional Error Mapping

The Application layer defines pure domain errors. The Transport layer is responsible for translating domain errors into the respective protocol semantics.

### Standard Domain Errors

```go
package domain

import "errors"

var (
 ErrNotFound           = errors.New("resource not found")
 ErrUnauthorized       = errors.New("unauthorized: missing or invalid credentials")
 ErrForbidden          = errors.New("forbidden: insufficient privileges")
 ErrInvalidInput       = errors.New("invalid input data")
 ErrConflict           = errors.New("resource conflict")
 ErrPreconditionFailed = errors.New("precondition failed")
 ErrRateLimited        = errors.New("rate limit exceeded")
 ErrDeadlineExceeded   = errors.New("operation deadline exceeded")
 ErrInternal           = errors.New("internal server error")
)
```

### Mapping Matrix

| Domain Error | HTTP Status | RFC 7807 Title | gRPC Code (`codes.Code`) | Explanation |
| :--- | :--- | :--- | :--- | :--- |
| `ErrNotFound` | 404 Not Found | Not Found | `codes.NotFound` | Target entity does not exist |
| `ErrUnauthorized` | 401 Unauthorized | Unauthorized | `codes.Unauthenticated` | Caller identity unverified |
| `ErrForbidden` | 403 Forbidden | Forbidden | `codes.PermissionDenied` | Caller lacks permissions |
| `ErrInvalidInput` | 400 Bad Request | Bad Request | `codes.InvalidArgument` | Malformed payload or validation error |
| `ErrConflict` | 409 Conflict | Conflict | `codes.AlreadyExists` | Duplicate unique key or concurrent update |
| `ErrPreconditionFailed` | 412 Precondition Failed | Precondition Failed | `codes.FailedPrecondition` | Business state requirement unmet |
| `ErrRateLimited` | 429 Too Many Requests | Rate Limit Exceeded | `codes.ResourceExhausted` | Quota or throughput exceeded |
| `ErrDeadlineExceeded` | 504 Gateway Timeout | Gateway Timeout | `codes.DeadlineExceeded` | Context canceled / deadline elapsed |
| `ErrInternal` / panic | 500 Internal Server Error | Internal Server Error | `codes.Internal` | Unexpected unhandled failure |

---

## 3. Go Implementation of Error Translation

```go
package transport

import (
 "encoding/json"
 "errors"
 "net/http"

 "cashflow_backend/internal/domain"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/status"
)

// RFC 7807 Problem Details representation
type ProblemDetails struct {
 Type     string `json:"type,omitempty"`
 Title    string `json:"title"`
 Status   int    `json:"status"`
 Detail   string `json:"detail"`
 Instance string `json:"instance,omitempty"`
}

func WriteHTTPError(w http.ResponseWriter, reqID string, err error) {
 status := MapToHTTPStatus(err)
 w.Header().Set("Content-Type", "application/problem+json")
 w.WriteHeader(status)

 problem := ProblemDetails{
  Title:    http.StatusText(status),
  Status:   status,
  Detail:   err.Error(),
  Instance: reqID,
 }
 _ = json.NewEncoder(w).Encode(problem)
}

func MapToHTTPStatus(err error) int {
 switch {
 case errors.Is(err, domain.ErrNotFound):
  return http.StatusNotFound
 case errors.Is(err, domain.ErrUnauthorized):
  return http.StatusUnauthorized
 case errors.Is(err, domain.ErrForbidden):
  return http.StatusForbidden
 case errors.Is(err, domain.ErrInvalidInput):
  return http.StatusBadRequest
 case errors.Is(err, domain.ErrConflict):
  return http.StatusConflict
 case errors.Is(err, domain.ErrPreconditionFailed):
  return http.StatusPreconditionFailed
 case errors.Is(err, domain.ErrRateLimited):
  return http.StatusTooManyRequests
 case errors.Is(err, domain.ErrDeadlineExceeded):
  return http.StatusGatewayTimeout
 default:
  return http.StatusInternalServerError
 }
}

func MapToGRPCError(err error) error {
 code := codes.Internal
 switch {
 case errors.Is(err, domain.ErrNotFound):
  code = codes.NotFound
 case errors.Is(err, domain.ErrUnauthorized):
  code = codes.Unauthenticated
 case errors.Is(err, domain.ErrForbidden):
  code = codes.PermissionDenied
 case errors.Is(err, domain.ErrInvalidInput):
  code = codes.InvalidArgument
 case errors.Is(err, domain.ErrConflict):
  code = codes.AlreadyExists
 case errors.Is(err, domain.ErrPreconditionFailed):
  code = codes.FailedPrecondition
 case errors.Is(err, domain.ErrRateLimited):
  code = codes.ResourceExhausted
 case errors.Is(err, domain.ErrDeadlineExceeded):
  code = codes.DeadlineExceeded
 }
 return status.Error(code, err.Error())
}
```
