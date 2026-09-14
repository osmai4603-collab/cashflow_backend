---
name: go-database-migrations
description: "Production-ready database migration engineering, transactional unit of work, row-level locking, and connection pool management for Go services using PostgreSQL (pgxpool). Covers zero-downtime expand-and-contract schema evolution, reversible DDL, transaction safety, and query leak prevention."
---

# Go Database Migrations & Storage Engineering Skill

This skill defines production standards for schema migrations, transaction management, connection pool
governance, and data consistency in Go services interacting with relational databases (PostgreSQL/pgxpool).
It guarantees that schema updates execute without downtime, transactions protect financial invariants,
and queries never leak database pool connections.

---

## Production Database Principles

1. **Zero-Downtime Schema Evolution (Expand & Contract)**:
   Never rename or delete a column in a single migration while older application instances are actively serving traffic. Add the new column first (Expand), deploy code that writes to both, backfill data, switch reads, and only then drop the old column (Contract).
2. **Deterministic & Reversible Versioning**:
   Every schema change must be paired as forward (`up.sql`) and backward (`down.sql`) migrations. Migrations must be strictly ordered, tracked in a dedicated `schema_migrations` table, and executed inside a database transaction whenever supported by the DDL.
3. **Financial Consistency via Row-Level Locking (`FOR UPDATE`)**:
   When calculating balances, processing debits/credits, or updating inventory counts, avoid non-atomic read-modify-write races. Always lock target rows explicitly with `SELECT ... FOR UPDATE` within an active transaction.
4. **Leak-Proof Connection Governance**:
   Every acquired connection must be returned to the pool. When using `Query()`, rows must always be closed (`defer rows.Close()`), and `rows.Err()` must be checked after iteration to catch hidden network interruptions.
5. **Safe Foreign Key Indexing**:
   Every foreign key constraint in PostgreSQL must be accompanied by a dedicated B-Tree index to prevent full-table sequential scans during joins and cascade validations.

---

## Architecture: Migration & Storage Layer

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                             Application Boot                             │
│                         cmd/migrate or Startup                           │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                   Zero-Downtime Migration Engine                         │
│   schema_migrations Table ── Strict Version Sort ── Transactional DDL    │
├────────────────────────────────────┬─────────────────────────────────────┤
│   up.sql (Forward Migration)       │   down.sql (Rollback Script)        │
│   - Idempotent (IF NOT EXISTS)     │   - Safe tear-down                  │
│   - Concurrent Index Creation      │   - Data preservation check         │
└────────────────────────────────────┴─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      Repository & Transaction Plane                      │
│   Unit of Work Pattern ── Row-Level Locking (FOR UPDATE) ── pgxpool      │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| Single-step Column Rename | Breaks running instances with missing column errors | Multi-phase Expand & Contract: add new column, sync, drop old |
| Missing `defer rows.Close()` | Exhausts pool connections; service hangs indefinitely | Always invoke `defer rows.Close()` immediately after `Query()` |
| Omitting `FOR UPDATE` on balance math | Race conditions cause double-spend / balance discrepancies | Lock the account row with `SELECT ... FOR UPDATE` inside `tx` |
| Long transactions during migrations | Table locks block application queries, causing 504 timeouts | Keep migrations fast; create large indexes with `CONCURRENTLY` |
| Unindexed Foreign Keys | Causes sequential table scans during delete/update cascades | Always create an index on the referencing foreign key column |

---

## Database Verification Checklist

```text
[ ] Migrations follow the Expand and Contract pattern with zero downtime
[ ] Every up.sql has a corresponding reversible down.sql
[ ] schema_migrations table tracks applied version numbers and execution timestamps
[ ] All Balance, Inventory, and Ledger mutations execute inside transactions with SELECT FOR UPDATE
[ ] Every database query with rows has a deferred rows.Close() and checks rows.Err()
[ ] Foreign key columns have explicit supporting B-tree indexes
[ ] Database connection pools configure MaxConns, MinConns, and MaxConnIdleTime
```
