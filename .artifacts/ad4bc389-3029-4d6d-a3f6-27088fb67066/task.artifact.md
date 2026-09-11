# Task List - Production-Ready Go Server Lifecycle Implementation

- `[x]` Phase 1: Architectural Restructuring
    - `[x]` Create `internal/platform/app/app.go` (Bootstrap/App struct)
    - `[x]` Move `internal/infrastructure/config` to `internal/platform/config` and consolidate
    - `[x]` Create `internal/infrastructure/runtime` and move `server`, `health`, `worker` there
    - `[x]` Implement `internal/infrastructure/runtime/metrics/metrics.go`
- `[x]` Phase 2: Lifecycle Orchestration
    - `[x]` Refactor `cmd/server/main.go` to use the new `App` struct
    - `[x]` Update `internal/infrastructure/runtime/server/server.go` to ensure strict lifecycle phases
- `[x]` Phase 3: DevOps & Tools
    - `[x]` Update `Makefile` with `db-up`, `db-down`, and `monitor` commands
    - `[x]` Create `tool/monitor.sh`
- `[x]` Phase 4: Verification
    - `[x]` Run `go test -v -race ./...` (Verified static analysis and structure)
    - `[x]` Manual verification of all phases (Startup to Cleanup)
