# Transactional Test Isolation & Auto-Rollback

Testing database repositories against a live PostgreSQL database is essential for catching SQL syntax errors, constraint violations, and migration discrepancies. However, tests that leave data behind or wipe the database prevent parallel test execution and cause test contamination.

---

## 1. The Rollback Pattern

Instead of inserting records directly into persistent tables and trying to clean them up with `DELETE` or `TRUNCATE`, every test starts an explicit transaction:

```go
func TestRepository_CreateInvoice(t *testing.T) {
 t.Parallel()

 // 1. Obtain isolated transaction from test database pool
 tx := testDB.BeginTx(t)
 // 2. Ensure rollback is guaranteed, even on test panic or failure
 t.Cleanup(func() {
  _ = tx.Rollback()
 })

 // 3. Construct repository wired with this transaction
 repo := NewInvoiceRepository(tx)

 // 4. Perform test mutations
 err := repo.Create(context.Background(), &Invoice{
  ID:        "inv_123",
  CompanyID: "comp_abc",
  Amount:    15000,
 })
 if err != nil {
  t.Fatalf("unexpected error creating invoice: %v", err)
 }

 // 5. Verify query results within the same transaction
 inv, err := repo.FindByID(context.Background(), "inv_123")
 if err != nil {
  t.Fatalf("failed to retrieve invoice: %v", err)
 }
 if inv.Amount != 15000 {
  t.Errorf("expected amount 15000, got %d", inv.Amount)
 }

 // When test finishes, t.Cleanup executes Rollback automatically.
 // The database state remains completely pristine.
}
```

---

## 2. Advantages Over Truncation or DB-per-Test

| Metric | Transactional Rollback | Table Truncation (`TRUNCATE`) | Separate DB per Test |
| :--- | :--- | :--- | :--- |
| **Speed** | Sub-millisecond (instant) | 50ms - 200ms per test | 1s - 5s per test |
| **Concurrency (`t.Parallel()`)** | Fully supported (isolated MVCC) | Impossible (race conditions) | Supported but heavy on memory |
| **Schema Integrity** | Preserved | Reset counters, locks tables | Complete overhead |
| **Cleanup Reliability** | 100% via `t.Cleanup` | Often leaves data on panics | High cleanup burden |

---

## 3. Handling Multi-Table Foreign Keys & Seeds

For tests that require static seed data (e.g. currencies, plan tiers, permissions):

- Seed static lookup tables **once** during test suite initialization (`TestMain`).
- Mutate transactional entities (users, companies, transactions) exclusively inside the test transaction.
- If a test requires a company record to satisfy a foreign key constraint, create the parent company inside the test transaction before inserting child records.
