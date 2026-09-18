# The 6-Stage Pre-Handling Pipeline & Security Boundary

This document breaks down the ordered lifecycle every request undergoes before entering the application handler, covering panic recovery, correlation tracing, security principal resolution, and syntactic validation.

---

## 1. Lifecycle Sequence

```text
Request Ingress
      │
      ▼
[Stage 1] Protocol Stamping & Fail-Safe Panic Recovery
      │
      ▼
[Stage 2] Correlation ID & Distributed Tracing (X-Request-ID, W3C traceparent)
      │
      ▼
[Stage 3] Metadata Normalization & Context Injection (Client IP, Headers)
      │
      ▼
[Stage 4] Authentication & SecurityPrincipal Resolution (JWT / API Key)
      │
      ▼
[Stage 5] Syntactic Payload Validation & DTO Decoding
      │
      ▼
[Stage 6] Deadlines, Timeouts & Rate Limiting Enforcement
      │
      ▼
Hand-off to Pure Application Use Case (Execute)
```

---

## 2. Deep Dive: The Six Stages

### Stage 1: Protocol Stamping & Panic Recovery

1. **Protocol Stamping**: The transport adapter determines the ingress path (`ProtocolHTTP` vs `ProtocolGRPC`) and records it in `TransportInfo`.
2. **Panic Recovery**: The transport boundary defers a recovery interceptor.
   - If downstream code panics (e.g., nil pointer dereference, slice out of bounds), the interceptor catches the panic.
   - The panic value and stack trace are logged to structured telemetry (`slog.Error`).
   - A standardized failure is returned to the client (`500 Internal Server Error` in HTTP, `codes.Internal` in gRPC) without exposing stack traces to external callers.

### Stage 2: Correlation ID & Distributed Tracing

Distributed systems require a correlation identifier linking requests across multiple microservices:

1. **Extraction**:
   - HTTP: Read `X-Request-ID` or `X-Correlation-ID`.
   - gRPC: Read `x-request-id` from incoming `metadata.MD`.
2. **Generation**: If missing, generate a monotonically sortable or cryptographically random identifier (`UUIDv7` or 128-bit hex string).
3. **Propagation**:
   - HTTP: Write `X-Request-ID` to the outgoing `w.Header()`.
   - gRPC: Call `grpc.SetHeader(ctx, metadata.Pairs("x-request-id", reqID))`.
4. **W3C TraceContext**: If `traceparent` (e.g., `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`) is present, extract and bind it to an OpenTelemetry span.

### Stage 3: Metadata Normalization

HTTP and gRPC handle metadata differently:

- **HTTP**: Case-insensitive headers accessed via `http.Header.Get()`, which canonicalizes keys (e.g., `Authorization`).
- **gRPC**: Metadata keys are case-folded to lowercase strings in `metadata.MD` (e.g., `authorization`).
- **Resolution**: `TransportContext.Header(key)` normalizes lookups so security checks don't need protocol-specific branches.
- **Client IP**: Checks `X-Forwarded-For` (first IP), `X-Real-IP`, and TCP peer address (`net.SplitHostPort`).

### Stage 4: Authentication & SecurityPrincipal Resolution

Authentication verifies *who* is calling:

1. **Credential Extraction**:
   - Look for `Authorization: Bearer <token>`.
   - Look for API Keys in `X-API-Key` or metadata.
2. **Cryptographic Verification**:
   - Signature verification (HMAC-SHA256, RSA, or Ed25519).
   - Expiration validation (`exp > now`) and Not-Before validation (`nbf <= now`).
3. **Principal Creation**:
   - Map claims to a clean `SecurityPrincipal`: `UserID`, `TenantID`, `Roles`, `Scopes`.
   - Inject the principal into both `TransportContext.SetPrincipal()` and `context.Context` via `WithPrincipal(ctx, principal)`.

### Stage 5: Syntactic Payload Validation vs Semantic Domain Validation

A critical separation of concerns:

- **Syntactic Validation (Transport Boundary)**:
  - Is the JSON malformed?
  - Are mandatory fields present?
  - Are strings within length constraints?
  - Are email addresses or UUIDs formatted correctly?
  - **Action**: Performed in the transport adapter/decoder. If invalid, reject immediately with `400 Bad Request` or `codes.InvalidArgument` before touching domain services.
- **Semantic / Domain Validation (Application Core)**:
  - Does this account have sufficient funds?
  - Is this order state eligible for cancellation?
  - **Action**: Handled exclusively inside the Use Case / Domain Entity.

### Stage 6: Deadlines, Timeouts & Rate Limiting

1. **Deadlines**:
   - gRPC propagates deadlines automatically via the `grpc-timeout` header, converted by the gRPC runtime into a `context.WithDeadline`.
   - HTTP requests should have an explicit timeout enforced via `context.WithTimeout(ctx, defaultTimeout)` to prevent slow-loris or resource exhaustion attacks.
2. **Rate Limiting**:
   - Token bucket or sliding-window rate limiters evaluated against `ClientIP` (for public endpoints) or `Principal.TenantID` / `Principal.UserID` (for authenticated users).
   - If exceeded, return `429 Too Many Requests` or `codes.ResourceExhausted`.
