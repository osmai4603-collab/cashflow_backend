# Official Standards & Architecture References for Transport Layer

This document catalogs the authoritative standards, official Go documentation, and reference implementations underpinning the unified HTTP/gRPC transport layer architecture.

---

## 1. Official Go Language Documentation (`go.dev`)

- [Go Standard Library `net/http` Package](https://pkg.go.dev/net/http): Definitive reference for `http.Handler`, `http.ResponseWriter`, `http.Request`, and canonical header formatting (`CanonicalHeaderKey`).
- [Go Standard Library `context` Package](https://pkg.go.dev/context): Best practices for request-scoped cancellation, deadlines, and collision-free unexported key typing.
- [Go Standard Library `log/slog` Package](https://pkg.go.dev/log/slog): Structured, high-performance logging for ingress boundaries.
- [Go Standard Library `net` Package](https://pkg.go.dev/net): IP address splitting (`net.SplitHostPort`) and listener multiplexing primitives.

---

## 2. Official gRPC & RPC Ecosystem Documentation

- [Official gRPC Go Documentation](https://grpc.io/docs/languages/go/): Comprehensive guides for unary and streaming server interceptors.
- [Go gRPC Package Reference (`google.golang.org/grpc`)](https://pkg.go.dev/google.golang.org/grpc): `UnaryServerInterceptor`, `StreamServerInterceptor`, and metadata lifecycle.
- [gRPC Metadata Reference (`google.golang.org/grpc/metadata`)](https://pkg.go.dev/google.golang.org/grpc/metadata): Key-value pair semantics for request and response headers (`metadata.MD`).
- [ConnectRPC Specification](https://connectrpc.com/): Official guidelines for unified RPC over HTTP/1.1, HTTP/2, and standard `http.Handler`.
- [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway): Reverse-proxy pattern translating REST/JSON into gRPC.
- [Go kit Transport Architecture](https://gokit.io/docs/architecture/): Foundational 3-layer service design (Transport -> Endpoint -> Service).

---

## 3. International Standards & RFCs

- **RFC 7807**: *Problem Details for HTTP APIs*. Standardized format for machine-readable HTTP error payloads (`application/problem+json`).
- **RFC 9110**: *HTTP Semantics*. Standards for HTTP status codes, method idempotency, and header parsing.
- **W3C Trace Context Specification**: Standardized `traceparent` and `tracestate` headers for distributed microservice tracing across HTTP and gRPC.
- **RFC 7519**: *JSON Web Token (JWT)*. Format for bearer tokens transported across `Authorization` headers.
- **RFC 7239**: *Forwarded HTTP Extension*. Canonical syntax for proxy forwarding headers (`Forwarded`, `X-Forwarded-For`).
