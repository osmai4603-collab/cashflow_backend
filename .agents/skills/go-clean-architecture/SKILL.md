---
name: go-clean-architecture
description: "Production-ready Clean Architecture standards for Go services. Covers the 5-layer separation (domain, usecase, adapters, infrastructure, platform), dependency inversion rule, port-and-adapter interfaces, transaction boundaries, DTO mapping, composition root wiring, and isolated unit testing with mocks."
---

# Go Clean Architecture Skill

This skill defines production-grade Clean Architecture and Hexagonal (Ports & Adapters) standards
specifically tailored for Go HTTP backends. It provides strict guidelines to maintain complete decoupling
between core business logic and external technologies (databases, web frameworks, external APIs).

---

## Core Principles

1. **The Dependency Inversion Rule**:
   Source code dependencies must point **strictly inward**. High-level policy (Domain & Use Cases) must never know about low-level delivery mechanisms (HTTP, SQL, JSON, Redis).
2. **Framework Independence**:
   The business logic is not married to Chi, Gin, pgx, GORM, or AWS. External libraries can be swapped without touching core enterprise rules.
3. **Consumer-Defined Interfaces (Go Idiom)**:
   Interfaces belong to the consumer, not the implementer. The `usecase` layer declares the interfaces (Ports) it requires; the `adapters` layer provides the concrete implementations.
4. **Testability in Total Isolation**:
   Every usecase interactor can be thoroughly tested with 100% test coverage using lightweight in-memory mocks without spinning up Docker, PostgreSQL, or HTTP servers.
5. **Clear Separation of DTOs and Domain Entities**:
   HTTP request/response schemas (DTOs) never escape into the Domain or Usecase layers. Mappers explicitly translate between DTOs and Domain Models.

---

## The 5-Layer Architecture in Go

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                   Layer 5: Platform (Composition Root)                   │
│   internal/platform/app ── Dependency Injection ── Configuration Wiring  │
├────────────────────────────────────┬─────────────────────────────────────┤
│   Layer 4: Infrastructure          │   Layer 3: Adapters (Interface)     │
│   - PostgreSQL Pool (pgxpool)      │   - HTTP Handlers (Chi / net/http)  │
│   - Redis Cache Client             │   - Repository Implementations      │
│   - Email / SMS Gateways           │   - External Service Adapters       │
│   - Event Bus Producers            │   - Request/Response DTOs & Mappers │
├────────────────────────────────────┴─────────────────────────────────────┤
│                   Layer 2: Use Case (Application Rules)                  │
│   - Business Interactors & Workflows                                     │
│   - Input / Output Boundary Ports (Interfaces)                           │
│   - Application Transaction Management                                   │
├──────────────────────────────────────────────────────────────────────────┤
│                   Layer 1: Domain (Enterprise Rules)                     │
│   - Pure Entities & Value Objects                                        │
│   - Domain Invariants & Validations                                      │
│   - Domain-Specific Error Types                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Allowed vs Forbidden Imports (Dependency Inversion Matrix)

| Layer | Can Import | STRICTLY FORBIDDEN from Importing |
| :--- | :--- | :--- |
| **`domain`** | Standard Library only (`time`, `math`, etc.) | `usecase`, `adapters`, `infrastructure`, `platform`, ANY 3rd party lib |
| **`usecase`** | `domain` | `adapters`, `infrastructure`, `platform`, `net/http`, `sql`, `pgx` |
| **`adapters`** | `domain`, `usecase` | `infrastructure`, `platform` (unless passed as interface) |
| **`infrastructure`** | `domain` (rarely `usecase` types) | Direct coupling to `adapters` HTTP logic |
| **`platform`** | ALL layers (Composition Root only) | Business logic implementation inside platform |

---

## Layer-by-Layer Standards

### Layer 1: Domain (`internal/domain`)

- Contains pure structs representing business entities (e.g. `Invoice`, `Customer`, `Account`).
- Methods enforce business invariants (e.g. `invoice.MarkAsPaid()`, `account.Withdraw(amount)`).
- Defines sentinel domain errors (e.g. `ErrInsufficientFunds`, `ErrInvoiceAlreadyPaid`).
- **Forbidden**: `json:` tags, SQL annotations, or dependencies on external packages.

### Layer 2: Use Case (`internal/usecase`)

- Represents specific business operations (e.g. `CreateInvoiceUseCase`, `ApproveFleetContractUseCase`).
- Declares **Ports** (interfaces for persistence and external integrations):

  ```go
  type InvoiceRepository interface {
      GetByID(ctx context.Context, id string) (*domain.Invoice, error)
      Save(ctx context.Context, invoice *domain.Invoice) error
  }
  ```

- Coordinates the flow of data to and from entities, and directs them to use their domain rules.

### Layer 3: Adapters (`internal/adapters`)

- **Primary / Driving Adapters**:
  - HTTP Handlers (`internal/adapters/http/`): Parse JSON requests into DTOs, validate parameters, invoke the Use Case, and format the HTTP response.
- **Secondary / Driven Adapters**:
  - Storage Repositories (`internal/adapters/storage/`): Implement the `usecase` interfaces using concrete database drivers (e.g., executing SQL queries via `pgxpool`).

### Layer 4: Infrastructure (`internal/infrastructure`)

- Concrete drivers, third-party SDKs, and network clients:
  - Database pool initialization (`pgxpool.Connect`).
  - Cache clients (Redis).
  - Messaging queues, SMTP providers.

### Layer 5: Platform (`internal/platform`)

- Acts as the **Composition Root**.
- Instantiates configuration, database connections, repositories, use cases, handlers, and routers.
- Injects dependencies cleanly via constructors (`New...`) without reliance on magical reflection-based containers.

---

## Unit Testing & Mocking Strategy

Because use cases depend only on interfaces, testing requires zero external infrastructure:

```go
type mockInvoiceRepo struct {
    saveFunc func(ctx context.Context, inv *domain.Invoice) error
}

func (m *mockInvoiceRepo) Save(ctx context.Context, inv *domain.Invoice) error {
    return m.saveFunc(ctx, inv)
}

func TestCreateInvoice_Success(t *testing.T) {
    repo := &mockInvoiceRepo{
        saveFunc: func(ctx context.Context, inv *domain.Invoice) error {
            return nil
        },
    }
    uc := usecase.NewCreateInvoiceUseCase(repo)
    err := uc.Execute(context.Background(), usecase.CreateInvoiceCmd{Amount: 100})
    if err != nil {
        t.Fatalf("expected nil error, got %v", err)
    }
}
```

---

## Clean Architecture Anti-Patterns to Avoid

| Anti-Pattern | Why It Breaks Architecture | Correct Approach |
| :--- | :--- | :--- |
| Importing `pgx` in `usecase` | Couples business logic directly to PostgreSQL | Define repository interface in `usecase`; implement in `adapters/storage` |
| Returning DTOs from `usecase` | Exposes delivery format into application logic | Usecase accepts Command/Query structs and returns Domain Entities or domain results |
| Passing HTTP Context blindly | Leaks HTTP concepts into persistence or domain | Pass standard `context.Context` |
| Defining interfaces in the adapter | Violates Dependency Inversion (interface belongs to consumer) | Declare interfaces in `usecase` where they are consumed |
| Fat Handlers with business logic | Prevents reuse, makes logic untestable without HTTP | Handlers only do parsing, validation, calling usecase, and serialization |

---

## Clean Architecture Verification Checklist

```text
[ ] internal/domain has ZERO imports of external packages or internal/adapters/infrastructure
[ ] All persistence operations in internal/usecase are represented by interface contracts (Ports)
[ ] Concrete SQL and database calls exist ONLY within internal/adapters/storage or internal/infrastructure
[ ] HTTP handlers do not execute business rules or direct database queries
[ ] Handlers map request DTOs -> Usecase Commands and Domain Entities -> response DTOs
[ ] Dependencies are wired explicitly in internal/platform/app (Composition Root)
[ ] Use case unit tests run completely in-memory with zero Docker or network dependencies
```
