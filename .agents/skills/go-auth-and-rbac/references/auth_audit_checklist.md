# Authentication, RBAC & Multi-Tenancy Audit Checklist

Use this checklist during architecture reviews and code audits to ensure compliance with security and multi-tenancy standards.

---

## 1. Multi-Tenant Isolation (BOLA / IDOR Defense)

- [ ] Is `company_id` / `tenant_id` extracted exclusively from validated token/session claims?
- [ ] Is `company_id` injected into `context.Context` via an unexported key?
- [ ] Does every SQL query modifying or reading tenant resources include `WHERE company_id = $tenant`?
- [ ] Are route identifiers (e.g. `/invoices/{id}`) checked against the caller's tenant in the DB query?
- [ ] Are cross-company foreign keys protected by composite foreign key constraints?
- [ ] Is database Row-Level Security (RLS) considered or enabled for critical tables?

---

## 2. JWT & Token Lifecycle

- [ ] Is the signing algorithm explicitly verified (e.g. whitelist `HS256` or `RS256`)?
- [ ] Is `alg: "none"` explicitly rejected?
- [ ] Are HMAC keys at least 32 bytes (256 bits) of cryptographically secure random entropy?
- [ ] Is token expiration (`exp`) strictly enforced with no leeway beyond clock skew ($\le 60$s)?
- [ ] Are access tokens short-lived ($\le 15$ minutes)?
- [ ] Are refresh tokens rotated upon use (Refresh Token Rotation)?
- [ ] Does the system detect refresh token reuse and immediately invalidate all user sessions?

---

## 3. RBAC & Authorization

- [ ] Are route endpoints protected by permission-checking middleware (`RequirePermission`)?
- [ ] Are domain/use-case operations verifying ownership and contextual business rules?
- [ ] Is permission matching hierarchical or wildcard-enabled (`invoices:*`)?
- [ ] Do unauthorized requests return `401 Unauthorized` (when unauthenticated) vs `403 Forbidden` (when authenticated but lacking permissions)?
- [ ] Are permission changes or role revocations reflected immediately or upon token refresh?

---

## 4. Rate Limiting & Brute Force Defense

- [ ] Are authentication endpoints (`/login`, `/refresh`, `/reset-password`) protected by rate limiters?
- [ ] Is rate limiting enforced by both IP address and user identifier (email/username)?
- [ ] Are 429 responses returning standard `Retry-After` headers?
- [ ] Does password verification use constant-time comparisons (`bcrypt.CompareHashAndPassword`) to prevent timing attacks?
- [ ] Does failed login attempt accounting trigger graduated backoff or account lockout after threshold?

---

## 5. Network & HTTP Security

- [ ] Are HTTPS and HSTS (`Strict-Transport-Security`) enforced?
- [ ] Are standard security headers (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`) present on all responses?
- [ ] Is CORS strictly configured without wildcard (`*`) when credentials are supported?
- [ ] Are sensitive cookies marked `Secure`, `HttpOnly`, and `SameSite=Lax` or `Strict`?
