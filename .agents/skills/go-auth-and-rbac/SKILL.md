---
name: go-auth-and-rbac
description: "Production-ready authentication, role-based access control (RBAC), multi-tenant company isolation, and security middleware standards for Go HTTP backends. Covers JWT validation, context-scoped tenant propagation, BOLA/IDOR prevention, and endpoint rate limiting."
---

# Go Authentication, RBAC & Multi-Tenancy Skill

This skill defines production standards for identity verification, permission enforcement, and
multi-tenant data isolation in Go services. It guarantees that user sessions are cryptographically
validated, tenant boundaries (`company_id`) are impenetrable, and sensitive endpoints are protected
against authorization bypass (BOLA/IDOR) and brute force attacks.

---

## Production Security Principles

1. **Context-Scoped Multi-Tenant Isolation**:
   In multi-company architectures, every authenticated request must resolve the tenant (`company_id`) from the verified token/session and store it in the Go `context.Context`. Repositories must unconditionally scope every database query by `company_id` to prevent Broken Object-Level Authorization (BOLA/IDOR).
2. **Cryptographically Sound JWT Handling**:
   Validate JWT signatures, expiration (`exp`), issuer (`iss`), and audience (`aud`). Never accept `alg: none`. Enforce minimum signing secret lengths ($\ge 32$ bytes).
3. **Decoupled RBAC Guards**:
   Authorization is enforced in two places:
   - **Route Middleware**: High-level gating (e.g. requiring `role:manager` or `perm:invoices:write`).
   - **Domain / Use Case**: Low-level policy checks (e.g. "a manager can only approve invoices within their assigned company").
4. **Rate Limiting on Authentication Endpoints**:
   Login, password reset, and token refresh endpoints must enforce token-bucket or sliding-window rate limiting by client IP and account username to prevent brute-force attacks.
5. **Secure Headers & Strict CORS**:
   All HTTP responses must set modern security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Content-Security-Policy`, `Strict-Transport-Security`).

---

## Architecture: The Security & Tenant Chain

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                         Incoming HTTP Request                            │
│           Authorization: Bearer <token> ─── Origin Header                │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                       Security Middlewares Chain                         │
│  Security Headers ─── CORS ─── Rate Limiter (IP/User) ─── JWT Validator  │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                        Context Tenant Injection                          │
│     ctx = WithTenant(ctx, claims.CompanyID, claims.UserID, claims.Roles) │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           RBAC Permission Guard                          │
│     RequiresPermission(ctx, "finance:invoices:create")                   │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                  Repository Layer (Automatic Scoping)                    │
│     SELECT * FROM invoices WHERE id = $1 AND company_id = $2             │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| `company_id` from Query String | Attacker can manipulate the parameter to access other companies' data | Extract `company_id` strictly from authenticated JWT claims |
| `SELECT * WHERE id = $1` without tenant | BOLA/IDOR vulnerability: users can read competitors' invoices | Always append `AND company_id = $tenantID` |
| Long-lived access tokens | Stolen token grants indefinite access | Use short-lived access tokens (15m) + secure refresh tokens |
| Ignoring `err` on JWT verify | Forged tokens pass as valid | Strict verification of algorithm, signature, and expiration |
| Hardcoded JWT secrets in code | Secrets get committed to version control | Load from environment variables and validate secret length |

---

## Auth & Security Verification Checklist

```text
[ ] All tenant-scoped database queries explicitly filter by company_id
[ ] JWT tokens enforce HMAC-SHA256 or RSA with minimum 32-byte secret
[ ] Expired tokens return 401 Unauthorized immediately
[ ] Role and permission checks occur before executing business logic
[ ] Sensitive routes (login, token) have IP-based rate limiting
[ ] Security headers (X-Frame-Options, HSTS, X-Content-Type-Options) are set
```
