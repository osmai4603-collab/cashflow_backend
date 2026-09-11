# Implementation Plan - Updating Skill Examples and References

This plan focuses on bringing the code examples and detailed references within the `go-server-lifecycle` skill folder up to date with the newly established production standards (Dual-Mode Logging, Observability, and DB monitoring).

## Proposed Changes

### 1. Update Examples (`examples/`)

#### [MODIFY] [01_initialization.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/.agents/skills/go-server-lifecycle/examples/01_initialization.go)
- Implement `initLogger` with `isatty` detection.
- Add a placeholder for `prettyHandler` to demonstrate the "Dual-Mode" logging principle.

#### [MODIFY] [04_health_checks.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/.agents/skills/go-server-lifecycle/examples/04_health_checks.go)
- Add a `/metrics` handler example.
- Include a `DBStatsProvider` interface in the example.

#### [MODIFY] [complete_server.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/.agents/skills/go-server-lifecycle/examples/complete_server.go)
- Refactor to use the `initLogger` and `prettyHandler` logic.
- Add the `metrics` middleware to the router.
- Register database stats for the metrics handler.
- Suppress logs for healthy probes (`/metrics`, `/livez`, `/readyz`).

### 2. Update References (`references/`)

#### [MODIFY] [detailed_reference.md](file:///home/osm/StudioProjects/cashflow/cashflow_backend/.agents/skills/go-server-lifecycle/references/detailed_reference.md)
- Add a new section on "Professional Logging & TTY Detection".
- Add a new section on "Metrics & Runtime Observability".
- Update the "Anti-Patterns" and "Testing" sections to include metrics and logging validation.

## Verification Plan

### Manual Verification
- Review the code in the modified files to ensure they are syntactically correct and accurately reflect the best practices documented in `SKILL.md`.
- Ensure all links within `detailed_reference.md` are correct.
