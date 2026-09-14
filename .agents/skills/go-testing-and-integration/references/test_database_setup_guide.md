# Test Database Setup & Migration Guide

Running integration tests against a real PostgreSQL instance is significantly more dependable than mocking the SQL driver or using SQLite (which has different dialect syntax, locking semantics, and type systems).

---

## 1. Ephemeral Test Database Strategy

### Option A: Local Docker Compose (Development & CI)
Run a dedicated test database container with tmpfs (in-memory storage) for maximum speed:

```yaml
version: '3.8'
services:
  postgres_test:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: test_user
      POSTGRES_PASSWORD: test_password
      POSTGRES_DB: test_db
    tmpfs:
      - /var/lib/postgresql/data # High speed in-memory execution
    ports:
      - "5433:5432"
```

---

## 2. Managing Test Migrations with `TestMain`

Use Go's `TestMain` entrypoint in your integration test package to run schema migrations **once** before any tests run, and clean up afterwards:

```go
// +build integration

package tests

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://test_user:test_password@localhost:5433/test_db?sslmode=disable"
	}

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	// Apply database schema migrations
	if err := runMigrations(testDB); err != nil {
		panic(err)
	}

	// Run test suite
	exitCode := m.Run()

	_ = testDB.Close()
	os.Exit(exitCode)
}
```

---

## 3. Environment Variables for Test Isolation

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `TEST_DATABASE_URL` | `postgres://localhost:5433/test_db` | Dedicated test database connection |
| `TEST_INTEGRATION` | `1` | Build/run flag for skipping slow tests in fast unit mode |
| `TEST_LOG_LEVEL` | `error` | Suppress noisy debug logs during test runs |
