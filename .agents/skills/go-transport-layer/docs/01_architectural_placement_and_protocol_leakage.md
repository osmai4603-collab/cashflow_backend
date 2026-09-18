# Architectural Placement and Protocol Leakage Prevention

This document details the architectural positioning of the Transport Layer within Hexagonal (Ports & Adapters) and Clean Architecture in Go, focusing on the root causes and mitigation of **Protocol Leakage**.

---

## 1. Architectural Positioning: The Driving Adapter Boundary

In modern software architecture, the transport layer is classified strictly as a **Driving (Primary) Adapter**.

```text
       ┌────────────────────────────────────────────────────────┐
       │                   Driving Adapters                     │
       │                                                        │
       │   HTTP Handlers (REST/JSON)   gRPC Services (Protobuf) │
       │   └── http.Handler            └── gRPC Server/Stream   │
       │              │                            │            │
       └──────────────┼────────────────────────────┼────────────┘
                      ▼                            ▼
       ┌────────────────────────────────────────────────────────┐
       │                 Application Ingress Ports              │
       │                                                        │
       │       type UseCase interface { ... }                   │
       │       Pure DTOs (Data Transfer Objects)                │
       └──────────────────────────┬─────────────────────────────┘
                                  ▼
       ┌────────────────────────────────────────────────────────┐
       │             Core Application & Domain Engine           │
       │                                                        │
       │       Entities, Aggregate Roots, Domain Services       │
       └────────────────────────────────────────────────────────┘
```

### The Inward Dependency Rule

Dependencies must point strictly **inward**:

1. External network protocols depend on Transport Adapters.
2. Transport Adapters depend on Application Ports (`UseCase` interfaces) and Application DTOs.
3. Use Cases depend on Domain Entities and Driven Ports (Repositories, External Gateways).
4. **The Application and Domain layers must NEVER depend on the Transport layer, `net/http`, or `google.golang.org/grpc`.**

---

## 2. What is Protocol Leakage?

Protocol Leakage occurs when concepts, abstractions, types, or paradigms specific to a transport protocol cross the boundary into application use cases, domain services, or entities.

### Common Manifestations of Leakage

#### Leak 1: Passing `*http.Request` or `http.ResponseWriter` into Business Services

```go
// ANTI-PATTERN: Leaks HTTP into the application layer!
func (s *OrderService) CreateOrder(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID")
    var req CreateOrderPayload
    json.NewDecoder(r.Body).Decode(&req)
    // ...
    w.WriteHeader(http.StatusCreated)
}
```

**Why this is catastrophic:**

- It is impossible to call `CreateOrder` from a gRPC service, CLI command, or background worker without constructing synthetic `http.Request` and dummy `httptest.ResponseRecorder` objects.
- Unit tests require mocking entire HTTP requests, setting headers, cookies, URL paths, and body streams.
- The service becomes responsible for status codes, content-types, and network serialization instead of business invariants.

#### Leak 2: Passing Generated Protobuf Structs into Core Domain Entities

```go
// ANTI-PATTERN: Leaks Protobuf generators into domain entities!
func (s *OrderService) ProcessOrder(ctx context.Context, pb *orderpb.OrderRequest) (*orderpb.OrderResponse, error) {
    // Directly manipulating protobuf internal mutexes, unknown fields, and getters
}
```

**Why this is catastrophic:**

- Protobuf generated structs contain code-generated state (`state protoimpl.MessageState`, `sizeCache`, `unknownFields`) and lack rich domain methods or encapsulation.
- Domain rules become entangled with Protobuf field tagging and default zero-value representations (e.g., Protobuf cannot distinguish between an unset string and an empty string `""` without optional wrappers).
- HTTP endpoints must convert JSON to Protobuf structures just to invoke the usecase, creating unnecessary serialization overhead.

---

## 3. Comparison of Architectural Patterns

| Architecture Pattern | How it Operates | Architectural Pros | Architectural Cons & Cost | Recommended Usage |
| :--- | :--- | :--- | :--- | :--- |
| **1. Go kit (Endpoint + Transport)** | Divides service into (Transport -> Endpoint -> Service). Each operation is `type Endpoint func(ctx, req any) (resp any, err error)`. | Strong separation; middleware is completely reusable across protocols. | High boilerplate overhead: requires custom Decoder and Encoder functions per operation. | High-scale, complex multi-protocol distributed systems. |
| **2. Dual Driving Adapters** | Separate `http.Handler` and `grpc.Server` in `internal/transport/` calling the same `UseCase` interface. | Simple, idiomatic Go; easy to understand and maintain without third-party frameworks. | Risk of duplicating metadata extraction, auth checks, and validation across adapters. | Standard production microservices and modular monoliths. |
| **3. Unified TransportContext & Pre-Handling Pipeline (This Architecture)** | Introduces a common `TransportContext` interface and a unified 6-stage pipeline prior to use-case invocation. | Single source of truth for auth, tracing, logging, and metadata. Fully protocol-agnostic use cases. | Requires designing a clean `TransportContext` interface. | **Enterprise systems requiring robust HTTP & gRPC dual ingress.** |
| **4. ConnectRPC** | Schema-first RPC framework that compiles Protobuf into standard `http.Handler` conforming endpoints (supporting gRPC, gRPC-Web, and JSON). | Exceptional performance; 100% compliant with standard library `net/http`; single server. | Vendor lock-in to ConnectRPC toolchain and generated code. | Greenfield projects wanting Protobuf-first contracts with standard HTTP. |
| **5. gRPC-Gateway** | Reverse-proxy server that translates incoming HTTP/REST JSON calls into internal gRPC calls. | Single source of truth (`.proto` file); automated OpenAPI documentation. | Extra network or in-memory RPC hop; difficult to customize complex HTTP-specific headers/caching. | Legacy gRPC services needing secondary REST endpoints. |

---

## 4. The Unified Transport Boundary: How Ingress Works

To eliminate protocol leakage while preventing code duplication across HTTP and gRPC, the Transport Layer defines:

1. **The Ingress Envelope (`TransportContext`)**: A lightweight abstraction implemented by both HTTP and gRPC wrappers that normalizes headers, client IP, user agent, and security identity.
2. **The Pre-Handling Pipeline**: A deterministic series of cross-cutting interceptors executing before any domain handler runs.
3. **Pure Use Cases**: Business operations that accept a standard `context.Context` and a plain Go DTO (Data Transfer Object), returning a domain result or a domain error.

```text
Incoming HTTP Request ──> HTTPTransportContext ──┐
                                                 ├──> PreHandlingPipeline ──> UseCase(ctx, DTO)
Incoming gRPC Call    ──> GRPCTransportContext ──┘
```
