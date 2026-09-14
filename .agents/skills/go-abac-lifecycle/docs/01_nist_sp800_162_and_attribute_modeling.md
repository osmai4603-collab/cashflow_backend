# NIST SP 800-162 and Attribute Modeling in Go

This document details the formal mathematical modeling of Attribute-Based Access Control (ABAC) in accordance with **NIST SP 800-162** (*Guide to Attribute Based Access Control Definition and Considerations*) and how it translates into idiomatic, high-performance Go structures.

---

## 1. The Formal ABAC Model (NIST SP 800-162)

NIST SP 800-162 defines ABAC as an access control methodology where authorization privileges are granted to users through the evaluation of attributes assigned to subjects, resources, actions, and the environment.

Formally, the evaluation space is defined as the Cartesian product of four attribute domains:

$$\mathcal{C} = \mathcal{S} \times \mathcal{R} \times \mathcal{A} \times \mathcal{E}$$

Where:

- $\mathcal{S}$ is the set of **Subject Attributes**.
- $\mathcal{R}$ is the set of **Resource Attributes**.
- $\mathcal{A}$ is the set of **Action Attributes**.
- $\mathcal{E}$ is the set of **Environment Attributes**.

A policy rule $R_i$ is a Boolean predicate function mapping an evaluation context $c \in \mathcal{C}$ to a decision:

$$R_i: \mathcal{C} \to \{\text{Permit}, \text{Deny}, \text{NotApplicable}\}$$

```text
       ┌────────────────────────┐
       │   Subject Attributes   │
       │ (ID, Roles, Tenant)    │
       └───────────┬────────────┘
                   │
                   ▼
       ┌────────────────────────┐         ┌────────────────────────┐
       │  Resource Attributes   │────────▶│   Evaluation Context   │
       │ (Type, Owner, Status)  │         │     (Quadruple)        │
       └────────────────────────┘         └───────────┬────────────┘
                   ▲                                  │
                   │                                  ▼
       ┌───────────┴────────────┐         ┌────────────────────────┐
       │   Action Attributes    │         │  Boolean Policy Rules  │
       │ (Verb: read/update)    │         │    R_i(S, R, A, E)     │
       └────────────────────────┘         └───────────┬────────────┘
                   ▲                                  │
                   │                                  ▼
       ┌───────────┴────────────┐         ┌────────────────────────┐
       │ Environment Attributes │         │    Final PDP Decision  │
       │ (Time, IP, Security)   │         │    Permit vs Deny      │
       └────────────────────────┘         └────────────────────────┘
```

---

## 2. Strong Typing vs. Dynamic Extensibility in Go

A common architectural trap in Go ABAC implementations is choosing between:

1. **Rigid Strong Typing (`struct`)**: Maximum compile-time type safety, zero allocations, fast property access, but hard to extend with dynamic attributes across varied microservices.
2. **Untyped Generic Maps (`map[string]any`)**: Infinite flexibility, but heavy heap allocations, loss of type safety, and runtime panic risks due to faulty type assertions.

The **idiomatic Go pattern** achieves the best of both worlds by combining:

- **Canonical Top-Level Fields**: Strongly typed fields for attributes universally required across all authorization checks (`ID`, `TenantID`, `Type`, `OwnerID`, `RequestTime`).
- **Secondary Typed Extension Map (`map[string]any`)**: An optional auxiliary map for domain-specific attributes that varies per entity or subsystem.

### Canonical Go Modeling

```go
package abac

import (
    "fmt"
    "time"
)

// Subject encapsulates caller identity and security credentials.
type Subject struct {
    ID          string         `json:"id"`
    TenantID    string         `json:"tenant_id,omitempty"`
    Roles       []string       `json:"roles,omitempty"`
    Department  string         `json:"department,omitempty"`
    Clearance   int            `json:"clearance,omitempty"`
    Attributes  map[string]any `json:"attributes,omitempty"`
}

// Resource encapsulates the target entity being operated upon.
type Resource struct {
    ID          string         `json:"id"`
    Type        string         `json:"type"` // e.g. "document", "financial_record"
    OwnerID     string         `json:"owner_id,omitempty"`
    TenantID    string         `json:"tenant_id,omitempty"`
    Department  string         `json:"department,omitempty"`
    Sensitivity int            `json:"sensitivity,omitempty"`
    Status      string         `json:"status,omitempty"`
    Attributes  map[string]any `json:"attributes,omitempty"`
}

// Action encapsulates the intended verb and invocation semantics.
type Action struct {
    Verb   string `json:"verb"`             // e.g. "read", "create", "update", "delete", "approve"
    Method string `json:"method,omitempty"` // e.g. "POST", "GET", or gRPC method
}

// Environment encapsulates ambient and ephemeral context.
type Environment struct {
    RequestTime time.Time      `json:"request_time"`
    ClientIP    string         `json:"client_ip,omitempty"`
    NetworkZone string         `json:"network_zone,omitempty"`
    Attributes  map[string]any `json:"attributes,omitempty"`
}

// EvaluationContext binds the 4 dimensions together.
type EvaluationContext struct {
    Subject     Subject     `json:"subject"`
    Resource    Resource    `json:"resource"`
    Action      Action      `json:"action"`
    Environment Environment `json:"environment"`
}
```

---

## 3. Safe Attribute Accessor Helpers

To prevent runtime panics when reading from dynamic `Attributes` maps, safe accessor functions with fallback defaults must be used:

```go
// GetStringAttr safely extracts a string attribute from a generic attribute map.
func GetStringAttr(attrs map[string]any, key string, defaultVal string) string {
    if attrs == nil {
        return defaultVal
    }
    val, ok := attrs[key]
    if !ok {
        return defaultVal
    }
    strVal, ok := val.(string)
    if !ok {
        return defaultVal
    }
    return strVal
}

// GetFloatAttr safely extracts a float64 attribute.
func GetFloatAttr(attrs map[string]any, key string, defaultVal float64) float64 {
    if attrs == nil {
        return defaultVal
    }
    val, ok := attrs[key]
    if !ok {
        return defaultVal
    }
    switch v := val.(type) {
    case float64:
        return v
    case float32:
        return float64(v)
    case int:
        return float64(v)
    case int64:
        return float64(v)
    default:
        return defaultVal
    }
}

// GetBoolAttr safely extracts a boolean attribute.
func GetBoolAttr(attrs map[string]any, key string, defaultVal bool) bool {
    if attrs == nil {
        return defaultVal
    }
    val, ok := attrs[key]
    if !ok {
        return defaultVal
    }
    boolVal, ok := val.(bool)
    if !ok {
        return defaultVal
    }
    return boolVal
}
```

---

## 4. Multi-Tenant Isolation & BOLA/IDOR Invariants

In any multi-tenant system, access control must enforce strict cryptographic and logical boundaries between tenants. NIST SP 800-162 ABAC models handle multi-tenancy natively at the attribute layer:

```go
// InvariantTenantIsolation verifies that the subject and resource belong to the exact same tenant.
func InvariantTenantIsolation(ctx EvaluationContext) error {
    if ctx.Subject.TenantID == "" || ctx.Resource.TenantID == "" {
        return fmt.Errorf("tenant isolation violation: missing tenant ID")
    }
    if ctx.Subject.TenantID != ctx.Resource.TenantID {
        return fmt.Errorf("tenant isolation violation: subject tenant %q does not match resource tenant %q",
            ctx.Subject.TenantID, ctx.Resource.TenantID)
    }
    return nil
}
```

By verifying this invariant first in every policy rule or PDP engine evaluation, **Broken Object Level Authorization (BOLA)** and **Insecure Direct Object Reference (IDOR)** vulnerabilities (OWASP Top 10 API Security #1) are eliminated by design.
