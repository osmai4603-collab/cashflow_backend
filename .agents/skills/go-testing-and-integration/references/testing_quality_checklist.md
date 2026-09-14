# Go Testing & Quality Checklist

Review this checklist before merging PRs or finalizing test suites to ensure speed, determinism, and reliability.

---

## 1. Test Isolation & Clean State

- [ ] Does every database test use transactional auto-rollback (`t.Cleanup(func() { _ = tx.Rollback() })`)?
- [ ] Are tests free from destructive database operations (`TRUNCATE`, `DROP`, `DELETE FROM table`)?
- [ ] Can tests be run in random order without failing?
- [ ] Are tests marked with `t.Parallel()` where appropriate?
- [ ] Does the test avoid mutating global package variables or environment variables without reverting them in `t.Cleanup`?

---

## 2. Determinism & Timing

- [ ] Are tests free of arbitrary `time.Sleep()` statements?
- [ ] Are asynchronous operations synchronized using channels, `sync.WaitGroup`, or conditioned timeouts?
- [ ] Do context timeouts in tests have reasonable bounds ($\le 5$s) to prevent hanging CI pipelines?
- [ ] Are mock servers tearing down cleanly via `t.Cleanup(server.Close)`?

---

## 3. Concurrency & Race Safety

- [ ] Does the test suite pass with `go test -race ./...`?
- [ ] Are shared memory structures protected by mutexes or atomic operations during concurrent tests?
- [ ] Are all launched goroutines guaranteed to exit upon test completion?

---

## 4. HTTP & Contract Coverage

- [ ] Are status codes, response headers, and response payloads checked?
- [ ] Are negative test paths (400, 401, 403, 404, 422, 500) validated alongside the happy path?
- [ ] Does the error response match the RFC 7807 Problem Details contract?
- [ ] Are sensitive tokens and credentials redacted from test assertion logs?
