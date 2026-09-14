---
name: go-service-configuration
description: "Production-ready configuration engineering for Go HTTP services. Covers layered configuration resolution (defaults < file < env < CLI < overrides), fail-fast validation with multi-error aggregation, secret security and redaction (***), custom duration/size parsing, atomic file persistence (0600 permissions), and declarative schema introspection for self-documenting services."
---

# Go Service Configuration Skill

This skill defines production-grade configuration architecture for Go HTTP services. It establishes
strict guidelines for building resilient, layered, fail-fast, and secure configuration systems that
prevent misconfigured services from booting, protect sensitive credentials from log leakage, and
support diverse deployment targets (local dev, bare-metal, Docker, Kubernetes).

---

## Production Configuration Principles

A production-ready Go configuration system is built upon six foundational pillars:

1. **Layered Resolution Pipeline**:
   Configuration is never resolved from a single source. It merges multiple layers in a deterministic priority order:
   $$\text{Defaults} < \text{Configuration File (JSON/YAML)} < \text{Environment Variables} < \text{CLI Flags} < \text{Runtime Overrides}$$
2. **Fail-Fast Comprehensive Validation**:
   Never boot a service with partial or broken configuration. Validate all invariants (port ranges, required secrets, timeout relationships, database URLs) upfront during Phase 1 of the server lifecycle. Accumulate all errors into a unified diagnostic report rather than failing on the first error.
3. **Strict Secret Security & Redaction**:
   Passwords, API keys, and database credentials must never be printed in plain text in logs, stdout, or serialized JSON debug dumps. Sensitive fields must implement custom string formatting (`***`) and file permissions must be locked to `0600`.
4. **Resilient Type Parsing**:
   Human operators write timeouts as `"5s"` or `"10m"`, and memory limits as `"512MB"`. The configuration system must seamlessly parse both human-readable strings and integer numbers.
5. **Atomic File Persistence**:
   When writing configuration files back to disk, never write directly in-place. Write to a temporary file, flush to disk (`Sync()`), and perform an atomic POSIX rename to prevent file corruption during power cuts or crashes.
6. **Self-Documenting Declarative Schema**:
   Configuration properties should declare their environment key, default value, and description via Go struct tags (`env:`, `default:`, `desc:`), allowing automated generation of `.env.example` templates.

---

## Architecture: The Configuration Pipeline

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                         1. Hardcoded Defaults                            │
│           Embedded safe defaults compiled directly into the binary       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                   2. Configuration File Store (JSON/YAML)                │
│       config/app.json ── unmarshaled onto defaults (sparse overrides)    │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      3. Environment Variables (Env Layer)                │
│        Canonical keys (APP_PORT) + Legacy Aliases (PORT, PGPORT)         │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                       4. Command-Line Arguments (CLI)                    │
│                 Flags (--port 8080, --config path/to/file)               │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                       5. Runtime Programmatic Overrides                  │
│               Dynamic test hooks and ephemeral network adjustments       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                    Fail-Fast Multi-Error Validation Engine               │
│         Verify: Ports (1-65535), Non-Empty Secrets, Timeout Bounds       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
                              Ready Configuration
```

---

## Section 1: The Layered Loading Pipeline

### Resolution Precedence Rule

- **Defaults**: Safe, working fallback values (e.g. `Port: "8080"`, `ReadTimeout: 5s`).
- **File Store**: Sparse JSON/YAML on disk. Only keys explicitly present in the document override the pre-populated defaults. Missing keys retain their default values.
- **Environment**: Container-friendly overrides. Supports canonical keys and database aliases (e.g. `DB_PORT` falls back to standard `PGPORT`).
- **CLI Flags**: Operator command-line flags take absolute precedence over environment and file values.
- **Runtime Overrides**: Passed via functional options `WithRuntimeOverrides(func(cfg *Configuration))` for test suites or specialized orchestration.

---

## Section 2: Fail-Fast Multi-Error Validation

A broken configuration must abort service startup immediately during Phase 1 (`Initialization`) before opening network listeners or database pools:

```go
type ValidationError struct {
    Field   string
    Message string
}

type ValidationReport struct {
    Errors []ValidationError
}

func (r *ValidationReport) Add(field, msg string) {
    r.Errors = append(r.Errors, ValidationError{Field: field, Message: msg})
}

func (r *ValidationReport) Error() string {
    // Formats all errors into a structured multi-line report
}
```

### Essential Invariants to Check

- **Port Ranges**: `1 <= Port <= 65535`.
- **Database Connectivity**: Required driver set (`postgres`), non-empty host, user, and database name (unless memory mode).
- **Timeouts**: `ReadTimeout > 0`, `WriteTimeout > 0`, `ShutdownTimeout > 0`.
- **Drain Relationships**: Ensure `ShutdownTimeout > DrainDuration`.
- **Mandatory Secrets**: JWT signing secret length $\ge 32$ characters in production environments.

---

## Section 3: Secret Protection & Redaction

### 1. The `Redacted()` Pattern

Never log configuration objects directly with `fmt.Sprintf("%+v", cfg)` or `slog.Any("config", cfg)`:

```go
func (c *Configuration) Redacted() *Configuration {
    clone := *c
    if clone.Database.Password != "" {
        clone.Database.Password = "***"
    }
    if clone.Auth.JWTSecret != "" {
        clone.Auth.JWTSecret = "***"
    }
    if clone.Database.DatabaseURL != "" {
        clone.Database.DatabaseURL = maskURL(clone.Database.DatabaseURL)
    }
    return &clone
}
```

### 2. Custom `String()` and `MarshalJSON()`

For sensitive string types, define a custom type that masks itself automatically:

```go
type SecretString string

func (s SecretString) String() string {
    if s == "" {
        return ""
    }
    return "***"
}
```

### 3. File Permissions

Any configuration file containing credentials written to disk must use strict POSIX permissions:

```go
const ConfigFilePerm = os.FileMode(0o600) // Read/Write only for the current user
```

---

## Section 4: Human-Friendly Custom Type Parsing

Services must accept human-readable string values for time durations and memory sizes:

### 1. Flexible Duration Unmarshaling

Support both string representations (`"5s"`, `"10m"`, `"1h"`) and raw integer seconds (`5`, `60`):

```go
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
    var v interface{}
    if err := json.Unmarshal(b, &v); err != nil {
        return err
    }
    switch val := v.(type) {
    case float64:
        *d = Duration(time.Duration(val) * time.Second)
        return nil
    case string:
        parsed, err := time.ParseDuration(val)
        if err != nil {
            return err
        }
        *d = Duration(parsed)
        return nil
    default:
        return errors.New("invalid duration format")
    }
}
```

---

## Section 5: Atomic File Persistence

Writing configuration files to disk directly via `os.WriteFile(path, ...)` can corrupt files if the server crashes mid-write:

### The Safe Atomic Write Pattern

1. Create a temporary file in the same directory: `path + ".tmp.PID"`.
2. Write serialized JSON with `0600` permissions.
3. Call `tmpFile.Sync()` to flush kernel buffers to storage.
4. Close the temporary file.
5. Perform an atomic rename: `os.Rename(tmpPath, path)`.

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| Single-source `os.Getenv` | Inflexible; cannot set defaults or use JSON files | Layered pipeline: Defaults < File < Env < CLI |
| Exiting on first config error | Operator must restart N times to find N missing keys | Multi-error aggregator reporting all issues at once |
| Plaintext secrets in logs | Leaks database and JWT credentials to log aggregators | Implement `.Redacted()` and mask URLs/passwords |
| In-place `os.WriteFile` | Power cut or crash leaves empty or corrupted config file | Atomic write to temp file + `Sync()` + `os.Rename` |
| Permissive file permissions | Other local users on shared hosts can read passwords | Enforce `0600` (User read/write only) |
| Tight coupling to frameworks | Binds configuration structs to specific third-party tools | Pure Go structs with standard tags |

---

## Configuration Verification Checklist

```text
[ ] Configuration merges defaults, optional file, environment, and CLI flags in correct order
[ ] Fail-fast validation checks all fields and returns a unified multi-line error report
[ ] Password, tokens, and database connection strings are masked with *** in logs
[ ] Configuration files are written with 0600 permissions using atomic rename
[ ] Timeouts support both string durations ("5s", "10m") and integer seconds
[ ] Database settings support standard environment aliases (e.g. PGHOST, PGPORT)
[ ] .env.example template can be generated directly from struct tag specifications
[ ] All tests pass with race detector: go test -v -race ./...
```
