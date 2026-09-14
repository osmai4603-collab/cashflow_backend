---
name: go-platform-and-infrastructure
description: "Production-ready architecture standards for Platform primitives and Infrastructure adapters in Go HTTP backends. Covers external service gateways, circuit breaking, vendor SDK isolation, audit event logging, in-process notification bus, sequence generation, currency math, and explicit composition root bootstrapping."
---

# Go Platform Primitives & Infrastructure Adapters Skill

This skill defines production standards for delineating, designing, and maintaining the two foundational layers beneath business use cases in enterprise Go architectures: **Infrastructure** (external outbound services and vendor adapters) and **Platform** (shared cross-cutting technical primitives and application chassis).

---

## Architectural Separation: Platform vs Infrastructure

| Dimension | Platform Layer (`internal/platform`) | Infrastructure Layer (`internal/infrastructure`) |
| :--- | :--- | :--- |
| **Domain Scope** | Cross-cutting technical capabilities & chassis | External networks, third-party APIs, external protocols |
| **Dependency Vector** | Internal foundational primitives, zero vendor SDKs | Outbound adapters implementing domain ports |
| **Examples** | `audit`, `notificationbus`, `currency`, `sequence`, `i18n`, `config`, `app` | `payment` (Stripe/PayPal), `edi` (ZATCA/XML), `email`, `sms`, `odoo` |
| **Failure Mode** | In-memory invariants, deterministic logic | Network latency, remote downtime, vendor API rate limits |
| **Resilience Mechanism** | Memory-safe concurrency, atomic counters | Timeouts, circuit breakers, exponential backoff retries |

---

## Core Principles

### 1. Zero Vendor SDK Leakage (Infrastructure)

External vendor SDK types (e.g. `stripe.PaymentIntent`, `twilio.Message`, `aws.S3Client`) must **never** appear in `domain` or `usecase` packages. The domain defines a pure Port interface (`PaymentGateway`, `Mailer`), and the infrastructure adapter translates to and from vendor payloads.

### 2. Provider Registry Pattern (Infrastructure)

When supporting multiple external providers for a capability (e.g. manual payment, Stripe, PayPal), manage them via a thread-safe `ProviderRegistry`. Use cases request providers by code or capability, decoupled from concrete implementation details.

### 3. Non-Blocking Event & Notification Distribution (Platform)

In-process notification and event buses (`NotificationBus`) must decouple event producers from consumers. Slow subscribers must not block write transactions or API response latency. Use buffered channels with `select { case ch <- event: default: }` drops or background dispatch.

### 4. Deterministic Financial Precision (Platform)

Never use floating-point types (`float64`) for financial storage, balances, or arithmetic. Financial calculations must use integer minor units (cents) or fixed-point structures to eliminate IEEE-754 precision drift.

### 5. Atomic Sequence Generation (Platform)

Document numbers (e.g. `INV/2026/0001`) must combine atomic database locking (`SELECT ... FOR UPDATE`) with date token expansion (`%(year)s%(month)s`). Sequences must never produce duplicate numbers under concurrent traffic.

### 6. Explicit Composition Root (`app.go`)

Avoid magic dependency injection frameworks that rely on reflection or global registries. Construct the dependency graph explicitly in the composition root:
$$\text{Config} \rightarrow \text{Database} \rightarrow \text{Platform} \rightarrow \text{Infrastructure} \rightarrow \text{Repositories} \rightarrow \text{Use Cases} \rightarrow \text{Handlers} \rightarrow \text{Server}$$

---

## Architecture Flow Diagram

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                           HTTP Presentation                             │
│                  Handlers ─── Middlewares ─── Router                    │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Application Use Cases Layer                        │
│            Execute business actions using Domain Port Interfaces        │
└──────────────┬───────────────────────────────────────────┬──────────────┘
               │                                           │
               │ calls Outbound Ports                      │ utilizes shared tools
               ▼                                           ▼
┌────────────────────────────────────────┐ ┌──────────────────────────────┐
│       Infrastructure Adapters          │ │      Platform Primitives     │
│  - Payment Gateway (Stripe/PayPal)     │ │  - Audit Context & Tracker   │
│  - Regulatory EDI (Signing & Hash)     │ │  - Notification Event Bus    │
│  - External Mail & SMS Gateways        │ │  - Sequence Formatter        │
│  - Circuit Breakers & Timeout Handlers │ │  - Currency Math Engine      │
└──────────────────┬─────────────────────┘ └───────────────┬──────────────┘
                   │                                       │
                   ▼                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                   Composition Root (internal/platform/app)              │
│       Explicitly instantiates and wires all components at startup       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| Leaking Stripe/AWS types to Usecases | Couples core domain to external vendor APIs; breaks when switching providers | Implement pure domain Ports in Infrastructure adapters |
| Unbounded external HTTP calls | Third-party latency blocks server threads, exhausting database pool | Strict `context.WithTimeout` and circuit breakers on all external calls |
| `float64` for money calculations | Floating point drift creates rounding discrepancies in financial balances | Use integer minor units or fixed-point decimal structures |
| Slow consumers blocking event bus | A slow WebSocket client freezes invoice generation for all users | Buffered subscriber channels with non-blocking drops or worker pool |
| Global singletons for services | Causes race conditions and prevents parallel test execution | Explicit dependency injection via constructors |

---

## Platform & Infrastructure Checklist

```text
[ ] External vendor SDKs are isolated within internal/infrastructure packages
[ ] All external HTTP clients specify strict timeouts (Connect, TLS, Response)
[ ] Circuit breakers or retries with jitter are configured for external gateways
[ ] In-process event bus does not block producers when subscriber buffers fill
[ ] Audit fields (created_at, updated_at, created_by) propagate from context
[ ] Sequence numbering uses database row-level locking to prevent duplicates
[ ] Financial math avoids raw float64 arithmetic
[ ] Dependencies are wired explicitly in the composition root (app.go)
```
