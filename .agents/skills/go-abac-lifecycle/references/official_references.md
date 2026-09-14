# Official Standards & Architecture References for ABAC

This document catalogs the authoritative standards, official Go documentation, and reference implementations underpinning this ABAC lifecycle architecture.

---

## 1. Official Go Language Documentation (`go.dev`)

- [Go Security Architecture & Decisions](https://go.dev/doc/security/): Official overview of Go's design decisions regarding security, memory safety, minimal dependencies, and framework independence.
- [Security Best Practices for Go Developers](https://go.dev/security/best-practices): Official guidance covering automated vulnerability scanning (`govulncheck`), race condition detection (`-race`), and fuzz testing (`-fuzz`).
- [Go Vulnerability Database](https://go.dev/security/vuln/): Real-time tracking of security advisories in the Go ecosystem.
- [Go Standard Library `context` Package](https://pkg.go.dev/context): Best practices for value propagation, cancellation, and collision-free unexported key typing.
- [Go Standard Library `crypto/subtle` Package](https://pkg.go.dev/crypto/subtle): Constant-time byte comparisons to prevent timing side-channel attacks during token/attribute verification.
- [Go Standard Library `sync/atomic` Package](https://pkg.go.dev/sync/atomic): Atomic pointer techniques (`atomic.Pointer`) for zero-contention, lock-free policy hot-reloading.

---

## 2. Formal International Access Control Standards

- **NIST SP 800-162**: *Guide to Attribute Based Access Control (ABAC) Definition and Considerations*. The foundational federal standard defining ABAC architecture, attribute categorization (Subject, Resource, Action, Environment), and PDP/PEP/PIP/PAP functional components.
- **OASIS XACML 3.0**: *eXtensible Access Control Markup Language (XACML) Version 3.0*. Formal specification of combining algorithms (`deny-overrides`, `permit-overrides`, `first-applicable`) and policy evaluation semantics.
- **RFC 7807**: *Problem Details for HTTP APIs*. The standardized JSON structure for machine-readable HTTP error responses (`application/problem+json`) without leaking sensitive internal authorization rules.
- **RFC 7519**: *JSON Web Token (JWT)*. Industry standard for transmitting authenticated subject identity and claims securely.
- **OWASP API Security Top 10**: API1:2023 Broken Object Level Authorization (BOLA), and API5:2023 Broken Function Level Authorization (BFLA).

---

## 3. High-Scale Reference Implementations in Go

- [Kubernetes ABAC Authorizer (`k8s.io/apiserver`)](https://github.com/kubernetes/kubernetes/tree/master/pkg/apis/abac): The world's most widely deployed native Go ABAC implementation, demonstrating clean attribute interfaces (`authorizer.Attributes`) and policy matching.
- [Google CEL for Go (`google/cel-go`)](https://github.com/google/cel-go): Common Expression Language implemented in Go. A fast, memory-safe, non-Turing-complete language powering ABAC in Google Cloud IAM Conditions, Kubernetes ValidatingAdmissionPolicy, and Envoy/Istio.
- [Open Policy Agent (OPA)](https://github.com/open-policy-agent/opa): A CNCF graduated project written entirely in Go, enabling declarative policy-as-code evaluation for cloud-native microservices.
- [Casbin Authorization Library](https://github.com/casbin/casbin): Leading multi-paradigm authorization engine in Go supporting ABAC, RBAC, and custom policy matchers.
