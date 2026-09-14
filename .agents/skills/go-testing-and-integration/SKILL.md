---
name: go-testing-and-integration
description: "Production-ready testing and integration standards for Go backend services. Covers transactional database test isolation (auto-rollback), full HTTP handler integration with httptest, zero-external-dependency third-party service mocking, concurrency race testing with -race, and parallel test execution."
---

# Go Testing & Integration Engineering Skill

This skill defines production standards for creating fast, deterministic, non-flaky automated tests in Go. It establishes patterns for unit tests, repository database integration tests with transactional rollback, end-to-end HTTP API tests using standard library utilities, and external API mocking.

---

## Core Production Testing Principles

1. **Transactional Test Isolation (Auto-Rollback)**:
   Database integration tests must run inside an isolated database transaction (`tx.BeginTx`) and register an automatic rollback in `t.Cleanup(func() { _ = tx.Rollback() })`. Tests must **never** truncate tables, wipe shared databases, or leave dirty rows behind.
2. **Zero-Flakiness Parallel Execution**:
   Tests must be deterministic and capable of running with `t.Parallel()`. No tests should rely on global shared state, hardcoded wall-clock sleeps (`time.Sleep`), or sequential execution order.
3. **HTTP Integration with `httptest`**:
   Test HTTP handlers and middlewares using `net/http/httptest.ResponseRecorder` and `httptest.NewRequestWithContext`. Avoid spinning up real TCP network listeners for internal endpoint tests.
4. **Standard Library External Service Mocking**:
   Mock third-party external dependencies (payment gateways, messaging services) using `httptest.NewServer`. Point service configuration to the mock server URL.
5. **Always Run with `-race` in CI**:
   All Go unit and integration test suites must pass clean execution under the Go race detector (`go test -race ./...`).

---

## The Go Test Pyramid

```text
┌────────────────────────────────────────────────────────┐
│                   End-to-End Tests                     │
│          Full binary / smoke tests (Staging)           │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│               HTTP API Integration Tests               │
│     httptest.ResponseRecorder + Middlewares + Usecases │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│             Database Repository Integration            │
│       Real PostgreSQL / Transactional Rollback         │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                    Pure Unit Tests                     │
│         Domain logic, validators, parsers              │
└────────────────────────────────────────────────────────┘
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| `time.Sleep()` in tests | Slows CI down and causes intermittent race condition failures | Use channels, `sync.WaitGroup`, or polling with timeout |
| Truncating tables between tests | Destroys concurrency; prevents running tests in parallel (`t.Parallel()`) | Run tests inside transactions and rollback in `t.Cleanup` |
| Calling external APIs in CI | External network flakiness fails builds and leaks staging credentials | Mock external APIs with `httptest.NewServer` |
| Ignoring `-race` flag | Hides subtle data races that cause panics under production load | Always execute `go test -race ./...` in local and CI builds |
| Testing private unexported details | Fragile tests that break upon simple refactoring | Test public behavior and contract interfaces |

---

## Test Execution Commands

```bash
# Run all unit tests with race detection
go test -v -race -timeout 30s ./...

# Run tests in parallel with coverage report
go test -race -covermode=atomic -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out
```
