# Delineation of Responsibilities: Platform vs Infrastructure

In Clean and Hexagonal architectures, confusion often arises when deciding whether a new component belongs to the **Platform** layer or the **Infrastructure** layer. Establishing strict organizational criteria prevents architectural erosion and keeps dependencies unidirectional.

---

## 1. Defining the Platform Layer (`internal/platform`)

The Platform layer represents the **internal technical foundation** of the application. It provides cross-cutting capabilities that multiple domains and layers require to function, yet it contains no business rules specific to any one business entity.

### Characteristics of Platform Components

1. **Zero External Vendor Coupling**: Platform packages do not import third-party SaaS SDKs (e.g. Stripe, AWS, Twilio).
2. **Foundational & Shared**: Used across multiple domains and adapters (e.g., config parsing, response envelopes, error structures).
3. **Deterministic Execution**: Typically in-process memory operations, synchronization primitives, or standard library extensions.
4. **Application Chassis**: Houses the composition root (`app.go`) which boots the server and wires the entire dependency graph.

### Core Platform Packages

- `platform/audit`: Audit logging structures, context modifiers, and audit event sinks.
- `platform/notificationbus`: In-process publish/subscribe message bus.
- `platform/currency`: Precision currency conversion and exchange rate calculations.
- `platform/sequence`: Sequential document number generation and date formatting.
- `platform/config`: Layered configuration resolution and validation.
- `platform/errors`: Standardized RFC 7807 problem details and application error taxonomies.
- `platform/i18n`: Internationalization catalogs and localized string lookups.
- `platform/app`: The application composition root.

---

## 2. Defining the Infrastructure Layer (`internal/infrastructure`)

The Infrastructure layer contains the **concrete adapters that bridge the application to external systems, third-party APIs, and hardware protocols**. It encapsulates the messy details of networking, serialization, vendor authentication, and external protocol compliance.

### Characteristics of Infrastructure Components

1. **Outbound Port Implementations**: Implements interfaces defined by the domain or use case layers.
2. **Vendor SDK Encapsulation**: This is the **only** layer permitted to import external vendor libraries (e.g. AWS SDK, Stripe SDK, XML-RPC drivers).
3. **Non-Deterministic / Network Bound**: Inherently subject to network latency, partial failures, timeouts, and upstream rate limits.
4. **Resilience Mechanisms**: Must implement circuit breakers, retries with exponential backoff, and strict request timeouts.

### Core Infrastructure Packages

- `infrastructure/payment`: Payment gateway adapters (Stripe, PayPal, local payment switches).
- `infrastructure/edi`: Electronic Data Interchange, XML schema validation, cryptographic signing, QR codes.
- `infrastructure/email` & `sms`: Remote delivery gateways.
- `infrastructure/odoo`: Protocol adapters for external ERP synchronizations (XML-RPC/JSON-RPC).
- `infrastructure/runtime`: HTTP server lifecycle and worker supervisors.

---

## 3. Decision Matrix: Where Does a New Package Belong?

| Question | If YES $\rightarrow$ | If NO $\rightarrow$ |
| :--- | :--- | :--- |
| Does it communicate with an external SaaS, third-party network, or vendor API? | **Infrastructure** | Continue |
| Does it implement a domain Port for external messaging or payment? | **Infrastructure** | Continue |
| Is it a shared in-process utility needed across several domain modules? | **Platform** | Continue |
| Is it responsible for assembling dependencies and starting the application? | **Platform (`app`)** | Continue |
| Does it define enterprise business rules or data entities? | **Domain** | Use Case or Adapter |
