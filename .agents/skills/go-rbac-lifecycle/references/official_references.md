# Official Standards & Architecture References

This document catalogs the authoritative standards, official Go documentation, and reference implementations underpinning this RBAC lifecycle architecture.

---

## 1. Official Go Language Documentation (`go.dev`)

- [Go Security Architecture & Philosophy](https://go.dev/doc/security/): Official overview of Go's design decisions regarding security, memory safety, and framework independence.
- [Security Best Practices for Go Developers](https://go.dev/security/best-practices): Official guidance covering automated vulnerability scanning (`govulncheck`), race condition detection (`-race`), and fuzzing (`-fuzz`).
- [Go Vulnerability Database](https://go.dev/security/vuln/): Real-time tracking of security advisories in the Go ecosystem.
- [Go Standard Library `context` Package](https://pkg.go.dev/context): Best practices for value propagation, cancellation, and unexported key typing.
- [Go Standard Library `crypto/subtle` Package](https://pkg.go.dev/crypto/subtle): Constant-time byte comparisons to prevent timing side-channel attacks.
- [Effective Go](https://go.dev/doc/effective_go): Idiomatic Go design guidelines, package structure, and concurrency idioms.

---

## 2. Formal International Access Control Standards

- **NIST SP 800-21d**: *Proposed NIST Standard for Role-Based Access Control*. Defines the formal mathematical foundation of Core RBAC, Hierarchical RBAC, Constrained RBAC, and Symmetric RBAC.
- **ANSI/INCITS 359-2012**: *American National Standard for Information Technology - Role Based Access Control*. The international consensus standard for RBAC systems.
- **NIST SP 800-162**: *Guide to Attribute Based Access Control (ABAC) Definition and Considerations*. Reference for hybrid RBAC-ABAC models.
- **RFC 7807**: *Problem Details for HTTP APIs*. The standardized JSON structure for machine-readable HTTP error handling (`application/problem+json`).
- **RFC 7519**: *JSON Web Token (JWT)*. Industry standard for transmitting claims securely between parties.

---

## 3. High-Scale Reference Implementations in Go

- [Kubernetes Authorizer Engine (`k8s.io/apiserver`)](https://github.com/kubernetes/kubernetes/tree/master/pkg/apis/rbac): The gold standard for decoupled RBAC in Go, separating Role definitions from Subject Bindings.
- [Kubernetes RBAC Reference Guide](https://kubernetes.io/docs/reference/access-authn-authz/rbac/): Production deployment model for roles, cluster roles, subjects, and verbs.
- [Casbin Authorization Framework](https://github.com/casbin/casbin): Leading open-source Go authorization engine based on the PERM (Policy, Effect, Request, Matcher) paradigm.
