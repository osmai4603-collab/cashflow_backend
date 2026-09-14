# Concurrency, Race Testing & Goroutine Leak Prevention

Go's primary concurrency features (goroutines and channels) make high-throughput servers possible, but concurrent code without proper synchronization leads to data races, deadlocks, and silent goroutine leaks.

---

## 1. Running with the Race Detector (`-race`)

The Go race detector instruments memory accesses to catch unsynchronized read/write operations on the fly:

```bash
# Execute unit and integration tests with the race detector enabled
go test -race -v ./...
```

> **Production Rule**: Every continuous integration (CI) pipeline must fail builds if `go test -race` detects a race. Data races in Go cause indeterminate memory corruption and unpredictable crashes in production.

---

## 2. Writing Concurrent Stress Tests

To expose race conditions and lock contention in shared components (in-memory caches, rate limiters, connection pools), run parallel worker routines in tests:

```go
func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
 t.Parallel()

 store := NewSharedStore()
 const numGoroutines = 50
 const opsPerGoroutine = 100

 var wg sync.WaitGroup
 wg.Add(numGoroutines)

 for i := 0; i < numGoroutines; i++ {
  go func(workerID int) {
   defer wg.Done()
   for j := 0; j < opsPerGoroutine; j++ {
    key := fmt.Sprintf("key-%d", j%10)
    val := fmt.Sprintf("val-%d-%d", workerID, j)
    
    // Write and read concurrently
    store.Set(key, val)
    _ = store.Get(key)
   }
  }(i)
 }

 wg.Wait()
}
```

---

## 3. Detecting Goroutine Leaks in Tests

A goroutine leak occurs when a goroutine is launched but blocked indefinitely waiting on an unbuffered channel or an uncancelled context.

### Prevention Rules

1. **Always Cancel Contexts**: Ensure tests call `cancel()` when testing functions accepting cancellable contexts:

   ```go
   ctx, cancel := context.WithCancel(context.Background())
   t.Cleanup(cancel)
   ```

2. **Buffer Done Channels**: If a goroutine sends an exit or result token to a channel, ensure the channel has capacity 1 or the receiver is guaranteed to drain it:

   ```go
   errCh := make(chan error, 1) // Non-blocking send
   ```

3. **Audit Active Goroutines**: In long test runs, compare `runtime.NumGoroutine()` before and after running background worker components.
