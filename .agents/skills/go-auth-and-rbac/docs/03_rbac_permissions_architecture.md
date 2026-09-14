# Role-Based Access Control (RBAC) & Fine-Grained Permissions

Role-Based Access Control (RBAC) maps identities to roles, and roles to specific functional permissions. In a clean Go architecture, authorization must be decoupled from business logic while remaining auditable.

---

## 1. Roles vs Permissions

A common mistake is checking roles directly in business code (e.g. `if user.Role == "admin"`). This leads to brittle code when new roles (e.g. `auditor`, `finance_manager`) are introduced.

- **Permissions (Actions)**: Atomic capabilities, e.g., `invoices:create`, `invoices:approve`, `reports:export`.
- **Roles (Bundles)**: Logical collections of permissions assigned to users, e.g.:
  - `Admin`: `["*"]`
  - `Accountant`: `["invoices:create", "invoices:read", "invoices:update"]`
  - `Auditor`: `["invoices:read", "reports:read"]`

---

## 2. Two-Tier Authorization Architecture

Production Go applications enforce authorization at two distinct checkpoints:

```text
┌────────────────────────────────────────────────────────┐
│ 1. Endpoint / Middleware Tier (Coarse-Grained Guard)   │
│    "Does the caller have permission to call POST /api?"│
│    RequiresPermission("invoices:create")               │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ 2. Domain / Usecase Tier (Fine-Grained Policy Guard)   │
│    "Can this user approve an invoice exceeding $50k?"  │
│    policy.CanApproveAmount(user, invoice.Amount)       │
└────────────────────────────────────────────────────────┘
```

---

## 3. Route Guard Middleware Pattern

Route middleware should be concise and easily composable with the standard `net/http` handler signature.

```go
// RequirePermission creates an HTTP middleware that checks if the caller possesses the required permission.
func RequirePermission(authorizer Authorizer, requiredPerm string) func(http.Handler) http.Handler {
 return func(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   tenant, err := auth.FromContext(r.Context())
   if err != nil {
    http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
    return
   }

   if !authorizer.HasPermission(tenant.Roles, requiredPerm) {
    http.Error(w, `{"error":"forbidden","message":"insufficient permissions"}`, http.StatusForbidden)
    return
   }

   next.ServeHTTP(w, r)
  })
 }
}
```

---

## 4. Wildcard Matching & Hierarchies

Support hierarchical permission matching using colons (`:`), such as:

- `*` matches everything.
- `invoices:*` matches `invoices:read`, `invoices:create`, `invoices:approve`.
- `invoices:read` matches only `invoices:read`.

```go
func MatchPermission(grantedPattern, required string) bool {
 if grantedPattern == "*" || grantedPattern == required {
  return true
 }
 if strings.HasSuffix(grantedPattern, ":*") {
  prefix := strings.TrimSuffix(grantedPattern, ":*")
  return strings.HasPrefix(required, prefix+":")
 }
 return false
}
```
