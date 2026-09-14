# Audit Trail, Observability & Revocation Strategies

Access control is not a static check; it requires continuous accountability and lifecycle maintenance. This document details structured audit trails, observability metrics, and session revocation mechanisms in Go.

---

## 1. Tamper-Evident Audit Logging with `log/slog`

Every authorization decision must produce an immutable audit trail suitable for ingest into SIEM systems (e.g., Elastic, Splunk, CloudWatch).

### Audit Event Attributes

- `timestamp`: RFC 3339 nano format.
- `event_type`: `authz.decision`.
- `subject_id`: Unique identifier of the authenticated principal.
- `tenant_id`: Company or organization identifier.
- `roles`: Snapshot of assigned roles evaluated.
- `permission`: Target permission required.
- `decision`: `permit` or `deny`.
- `reason`: Machine-readable reason code (e.g., `role_unmatched`, `ssd_conflict`, `token_expired`).
- `client_ip`: Originating client IP.
- `latency_us`: Duration of the PDP evaluation in microseconds.

```go
func LogAudit(logger *slog.Logger, subject Subject, perm Permission, decision string, reason string, latency time.Duration, ip, path string) {
 level := slog.LevelInfo
 if decision == "deny" {
  level = slog.LevelWarn
 }

 logger.LogAttrs(context.Background(), level, "authz_decision",
  slog.String("event_type", "authz.decision"),
  slog.String("subject_id", subject.ID),
  slog.String("tenant_id", subject.TenantID),
  slog.Any("roles", subject.Roles),
  slog.String("required_permission", string(perm)),
  slog.String("decision", decision),
  slog.String("reason", reason),
  slog.String("client_ip", ip),
  slog.String("path", path),
  slog.Int64("latency_us", latency.Microseconds()),
 )
}
```

---

## 2. Real-Time Observability & OpenMetrics Counters

Instrumenting authorization helps detect brute-force attacks and privilege escalation attempts.

### Key Metrics

1. `authz_evaluations_total{decision="permit|deny", permission="...", tenant="..."}`: Counter tracking access decisions.
2. `authz_evaluation_duration_seconds`: Histogram tracking PDP decision latency (SLO: p99 < 500µs).
3. `authz_active_roles_gauge`: Total count of active roles registered in the in-memory engine.

---

## 3. Session & Role Revocation Strategies

A critical problem in distributed RBAC is **Privilege Latency**: when a user's role is revoked in the database, when does their active token stop working?

### Strategy 1: Short-Lived Access Tokens (Recommended)

- Issue stateless access JWTs with a short TTL (e.g., 5 to 15 minutes).
- When roles change, the user continues to hold permissions for at most 15 minutes.
- Refresh tokens (which are stateful) query the database for current roles on every token refresh.

### Strategy 2: Subject Token Versioning (Zero-Latency Revocation)

- Store a `token_version` (integer) on the user record in PostgreSQL.
- Embed `token_version: 3` in the JWT claims.
- Keep a high-speed Redis or in-memory map of `user_id -> current_token_version`.
- When an admin revokes a role, increment `token_version` in the database and flush the cache.
- The PEP compares the JWT's embedded version with the cache; if mismatched, it rejects with `401 Unauthorized`.

```go
type ClaimsWithVersion struct {
 UserID       string `json:"sub"`
 Roles        []Role `json:"roles"`
 TokenVersion int    `json:"v"`
}

func ValidateTokenVersion(cachedVersion int, claims ClaimsWithVersion) bool {
 return claims.TokenVersion == cachedVersion
}
```

---

## 4. Combating Privilege Creep (Periodic Access Reviews)

Over time, employees change departments and accumulate roles without old ones being revoked.

- Maintain an audit table: `role_assignments(user_id, role, granted_at, expires_at, granted_by)`.
- Enforce mandatory role expiration: time-bounded role grants (e.g., 90 days).
- Provide automated reports listing inactive accounts holding high-privilege roles.
