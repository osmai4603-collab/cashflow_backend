---
name: go-rest-api-design
description: "Production-ready REST API design standards for Go HTTP backends. Covers unified JSON response envelopes, RFC 7807 problem details error mapping, secure pagination/filtering/sorting, and idempotency key enforcement for financial and mutating operations."
---

# Go REST API Design & Error Handling Skill

This skill defines production standards for designing RESTful HTTP APIs in Go services. It ensures
consistent, predictable, and self-describing interfaces across all domain modules, implementing
standard response envelopes, RFC 7807 Problem Details for errors, secure pagination/filtering,
and idempotency keys for financial mutations.

---

## Production API Principles

1. **Unified Response Envelope**:
   All successful HTTP JSON responses follow a consistent structural contract:
   - Resource payloads are enclosed in `"data"`.
   - Metadata (pagination, counts, execution time) is enclosed in `"meta"`.
2. **RFC 7807 Problem Details for HTTP Errors**:
   Never return arbitrary strings like `{"error": "bad request"}`. Return structured Problem Details (`application/problem+json`) containing `type`, `title`, `status`, `detail`, `instance`, and domain-specific `code`.
3. **Safe & Bounded Pagination**:
   Never allow unbounded database scans. Provide default limits (e.g. 20) and enforce hard maximum limits (e.g. 100). Support both Offset/Limit and Cursor-based pagination.
4. **Idempotency for Mutating Operations**:
   Payment creation, invoice issuing, and financial transfers must enforce the `Idempotency-Key` header. If a network retry occurs with the exact same key, the server returns the cached original result without re-executing the financial mutation.
5. **Decoupled HTTP Status Mapping**:
   Domain business rules do not know about HTTP codes. The API adapter layer explicitly maps domain sentinel errors (`ErrNotFound` -> 404, `ErrInvalidState` -> 422, `ErrUnauthorized` -> 401).

---

## Architecture: The API Contract

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                             Incoming Request                             │
│       HTTP Method + Path + Idempotency-Key Header + JSON Payload         │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           API Adapter Middlewares                        │
│   Idempotency Cache ─── Request Validation ─── Pagination Parser         │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                   Domain & Use Case Execution Plane                      │
│                  Executes business invariants cleanly                    │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                         Standard Response Formatter                      │
├────────────────────────────────────┬─────────────────────────────────────┤
│   Success: 200 / 201 Created       │   Failure: 4xx / 5xx                │
│   {                                │   Content-Type: application/problem │
│     "data": { ... },               │   {                                 │
│     "meta": { "total": 100 }       │     "type": "urn:problem:not_found",│
│   }                                │     "title": "Invoice Not Found",   │
│                                    │     "status": 404,                  │
│                                    │     "detail": "Invoice 123 absent"  │
│                                    │   }                                 │
└────────────────────────────────────┴─────────────────────────────────────┘
```

---

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| Unbounded `SELECT *` | Massive database queries exhaust memory and CPU | Enforce strict `limit` and `offset` with upper caps (e.g. 100) |
| Plain string error responses | Frontends and SDKs cannot parse error details reliably | Return RFC 7807 JSON with structured `code` and `detail` |
| Returning 200 OK for errors | Breaks monitoring, load balancer health, and caching | Return appropriate HTTP status codes (404, 422, 401, 500) |
| Missing Idempotency on payments | Network retry causes customer to be double-charged | Enforce `Idempotency-Key` header and cache completed results |
| Leaking database column names | Couples API clients directly to internal schema | Use explicit JSON DTOs with descriptive, camelCase or snake_case keys |

---

## API Verification Checklist

```text
[ ] All success responses return a standard { "data": ..., "meta": ... } JSON envelope
[ ] All error responses use RFC 7807 (application/problem+json) with type, title, status, and code
[ ] Pagination parameters have sane defaults (20) and strict maximum bounds (100)
[ ] Financial mutations require and enforce the Idempotency-Key header
[ ] HTTP handlers do not execute business logic or direct SQL queries
[ ] All date/time fields serialize in ISO 8601 / RFC 3339 format
```
