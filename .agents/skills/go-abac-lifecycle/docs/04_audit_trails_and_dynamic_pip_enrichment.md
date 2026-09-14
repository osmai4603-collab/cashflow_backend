# Audit Trails and Dynamic PIP Enrichment in Go

This document details the architecture and implementation of the **Policy Information Point (PIP)** and **Tamper-Evident Audit Trails** in Go ABAC systems.

---

## 1. Dynamic Attribute Enrichment (PIP)

In a pure Attribute-Based Access Control system, client tokens (e.g., JWT) only carry **Subject** attributes (`user_id`, `roles`, `tenant_id`, `clearance`). They do not contain current **Resource** attributes (`owner_id`, `status`, `sensitivity_level`, `amount`) because:

1. Resources are mutable; embedding their state in client tokens leads to stale authorization decisions.
2. Embedding fine-grained resource data in JWTs inflates token size and creates security exposure.

Therefore, the **Policy Information Point (PIP)** dynamically enriches the evaluation context by fetching authoritative entity attributes from databases or caches.

```text
HTTP Request (Subject & Resource ID)
                 │
                 ▼
┌─────────────────────────────────┐
│ PEP Enforcement Middleware      │
└────────────────┬────────────────┘
                 │ Fetch target Resource Attributes
                 ▼
┌─────────────────────────────────┐
│ Policy Information Point (PIP)  │
│ (PostgreSQL / Redis / Memory)   │
└────────────────┬────────────────┘
                 │ Returns populated Resource struct
                 ▼
┌─────────────────────────────────┐
│ Policy Decision Point (PDP)     │
│ (Evaluates Quadruple)           │
└────────────────┬────────────────┘
                 │ Emits Audit Event
                 ▼
┌─────────────────────────────────┐
│ Structured Audit Logger (slog)  │
│ + Prometheus OpenMetrics        │
└─────────────────────────────────┘
```

---

## 2. Pluggable PIP Resolver Interface in Go

A clean architecture defines a port interface for PIP resolution:

```go
package abac

import "context"

// PIPResolver defines the contract for dynamically fetching resource attributes.
type PIPResolver interface {
    ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error)
}
```

### In-Memory / Database Implementation Example

```go
type DatabasePIPResolver struct {
    db DatabaseClient // Replace with your repository or pgxpool
}

func (r *DatabasePIPResolver) ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error) {
    // 1. Fetch raw entity from database
    row, err := r.db.FindEntity(ctx, resourceType, resourceID)
    if err != nil {
        return Resource{}, fmt.Errorf("failed to retrieve resource %s:%s: %w", resourceType, resourceID, err)
    }

    // 2. Map domain entity to generic Resource struct
    return Resource{
        ID:          row.ID,
        Type:        resourceType,
        OwnerID:     row.OwnerID,
        TenantID:    row.TenantID,
        Department:  row.Department,
        Sensitivity: row.SensitivityLevel,
        Status:      row.Status,
        Attributes: map[string]any{
            "amount": row.Amount,
        },
    }, nil
}
```

---

## 3. Tamper-Evident Structured Audit Logging (`log/slog`)

Every authorization decision must be audited to fulfill regulatory compliance (SOC2, ISO 27001, HIPAA, PCI-DSS).

### Audit Log Schema

```go
package abac

import (
    "log/slog"
    "time"
)

type AuditEvent struct {
    Timestamp   time.Time
    SubjectID   string
    TenantID    string
    ResourceType string
    ResourceID  string
    Action      string
    Decision    Decision
    RuleID      string
    Reason      string
    ClientIP    string
    Latency     time.Duration
}

func EmitAuditEvent(logger *slog.Logger, ev AuditEvent) {
    level := slog.LevelInfo
    if ev.Decision == DecisionDeny {
        level = slog.LevelWarn
    }

    decisionStr := "permit"
    if ev.Decision == DecisionDeny {
        decisionStr = "deny"
    }

    logger.Log(nil, level, "ABAC_AUDIT_DECISION",
        slog.String("event_type", "authorization_audit"),
        slog.String("decision", decisionStr),
        slog.String("rule_id", ev.RuleID),
        slog.String("reason", ev.Reason),
        slog.String("subject_id", ev.SubjectID),
        slog.String("tenant_id", ev.TenantID),
        slog.String("resource_type", ev.ResourceType),
        slog.String("resource_id", ev.ResourceID),
        slog.String("action", ev.Action),
        slog.String("client_ip", ev.ClientIP),
        slog.Duration("latency_ns", ev.Latency),
    )
}
```

---

## 4. Prometheus OpenMetrics Telemetry

Monitoring authorization latency and denial spikes in real time detects active brute-force or privilege escalation attacks:

```go
package abac

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    AuthzEvaluationsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "authz_evaluations_total",
            Help: "Total count of ABAC authorization evaluations partitioned by decision and resource type.",
        },
        []string{"decision", "resource_type", "action"},
    )

    AuthzEvaluationDurationSeconds = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "authz_evaluation_duration_seconds",
            Help:    "Latency histogram of ABAC policy evaluations in seconds.",
            Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005}, // 10µs to 5ms
        },
        []string{"resource_type"},
    )
)
```

By tracking `authz_evaluations_total{decision="deny"}`, your alerting infrastructure can immediately notify the security operations team (SOC) if an abnormal surge in authorization denials occurs for a given tenant or IP address.
