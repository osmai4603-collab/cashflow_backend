# Multi-Tenant Data Isolation & Context Scoping

Multi-tenancy in enterprise Go services requires absolute data segregation between organizations (`company_id` / `tenant_id`). A failure in multi-tenant isolation leads to Broken Object-Level Authorization (BOLA / IDOR), allowing unauthorized actors to read or modify another tenant's confidential records.

---

## 1. The Principle of Inviolable Tenant Isolation

1. **Source of Truth**: The tenant identifier must originate **exclusively** from a cryptographically verified credential (e.g. JWT claims or a secure server-side session). It must **never** be accepted from untrusted client inputs such as query parameters (`?company_id=123`), request body parameters, or arbitrary headers unless explicitly signed and verified.
2. **Context Propagation**: Once verified, the tenant identity is attached to `context.Context` via strongly-typed, unexported context keys.
3. **Mandatory Query Filtering**: Every database query, update, and delete targeting tenant-owned tables must include the tenant condition in its `WHERE` clause:
   ```sql
   SELECT id, title, amount FROM invoices WHERE id = $1 AND company_id = $2;
   UPDATE invoices SET status = $1 WHERE id = $2 AND company_id = $3;
   DELETE FROM invoices WHERE id = $1 AND company_id = $2;
   ```
4. **Foreign Key Integrity**: In multi-tenant schemas, composite unique keys and foreign keys should reference the tenant ID (e.g., `FOREIGN KEY (company_id, customer_id) REFERENCES customers(company_id, id)`).

---

## 2. Context Propagation Pattern in Go

Using `context.Context` for request-scoped tenant metadata requires unexported types to prevent context key collision across packages.

```go
package auth

import (
	"context"
	"errors"
)

type contextKey struct{ name string }

var tenantContextKey = &contextKey{name: "tenant_context"}

var ErrMissingTenant = errors.New("tenant context missing from request")

// TenantContext holds immutable verified tenant metadata.
type TenantContext struct {
	CompanyID string
	UserID    string
	Roles     []string
}

// WithTenant returns a new context containing the verified TenantContext.
func WithTenant(ctx context.Context, tenant TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant)
}

// FromContext extracts the TenantContext from the context.
func FromContext(ctx context.Context) (TenantContext, error) {
	val := ctx.Value(tenantContextKey)
	if val == nil {
		return TenantContext{}, ErrMissingTenant
	}
	tc, ok := val.(TenantContext)
	if !ok {
		return TenantContext{}, errors.New("invalid tenant context type in context")
	}
	return tc, nil
}
```

---

## 3. Defense-in-Depth in the Repository Layer

Never rely solely on routing or controllers to enforce isolation. Repositories must enforce tenant boundaries directly:

### Good: Tenant Passed from Context to Query
```go
func (r *InvoiceRepository) FindByID(ctx context.Context, id string) (*Invoice, error) {
	tenant, err := auth.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository requires tenant context: %w", err)
	}

	query := `
		SELECT id, company_id, title, amount, status, created_at 
		FROM invoices 
		WHERE id = $1 AND company_id = $2`

	var inv Invoice
	err = r.db.QueryRowContext(ctx, query, id, tenant.CompanyID).Scan(
		&inv.ID, &inv.CompanyID, &inv.Title, &inv.Amount, &inv.Status, &inv.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &inv, err
}
```

### Bad: Accepting Tenant ID as an Unverified Parameter
```go
// INSECURE: If caller passes companyID from an unvalidated JSON body, IDOR occurs!
func (r *InvoiceRepository) FindByIDInsecure(ctx context.Context, id, companyID string) (*Invoice, error) {
    // ...
}
```

---

## 4. Tenant Row-Level Security (PostgreSQL RLS)

For defense-in-depth, PostgreSQL Row-Level Security (RLS) can be combined with application-level checks. When an application obtains a database connection from the pool, it can set a local session variable:

```sql
SET LOCAL app.current_company_id = 'tenant_123';
```

And in the database migration:
```sql
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON invoices
    FOR ALL
    TO application_role
    USING (company_id = current_setting('app.current_company_id', true));
```

This guarantees that even if a developer forgets `AND company_id = $2` in a query, PostgreSQL automatically excludes rows belonging to other tenants.
