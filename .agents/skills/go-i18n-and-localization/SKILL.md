---
name: go-i18n-and-localization
description: "Production-ready internationalization (i18n), localization (l10n), and multilingual backend architecture for Go services. Covers BCP 47 language matching (golang.org/x/text/language), Unicode CLDR pluralization (including Arabic 6-form rules), message catalog management (x/text/message and go-i18n/v2), HTTP/gRPC locale negotiation middleware, context-scoped printer propagation, dynamic relational data persistence (PostgreSQL JSONB vs sidecar tables), RFC 7807 localized problem details, and server-side document rendering."
---

# Go Internationalization (i18n) & Localization (l10n) Architecture Skill

This skill defines production standards for multi-language negotiation, text formatting, and localized data persistence in Go HTTP/gRPC backends. It guarantees that language negotiation is deterministic, pluralization strictly adheres to Unicode CLDR standards (including all 6 Arabic forms), and API transport boundaries remain decoupled from raw presentation strings.

The architecture synthesizes:

- Official Go language text processing standards from [go.dev](https://go.dev) and **`golang.org/x/text`**.
- The enterprise standard **`github.com/nicksnyder/go-i18n/v2`**.
- International standards: **IETF BCP 47 (RFC 5646)**, **Unicode CLDR**, and **RFC 7807 (Problem Details for HTTP APIs)**.

---

## Production Architectural Principles

1. **Deterministic BCP 47 Negotiation & Fallback**:
   Incoming language tags must be matched against supported server tags using a pre-warmed `language.Matcher`. Dialects must hierarchically fall back to parent languages (e.g. `ar-EG` $\rightarrow$ `ar` $\rightarrow$ default) to prevent unhandled crashes.
2. **Decoupled API Boundaries (Semantic Machine Keys)**:
   API endpoints must emit **machine-readable error codes** (e.g. `INSUFFICIENT_FUNDS`) allowing client apps (Flutter, React) to format UI presentation. Backend translation is strictly reserved for outbound documents (PDFs, emails, SMS), audit feeds, and B2B webhooks.
3. **Strict Unicode CLDR Pluralization**:
   Services must avoid naive binary logic (`if count == 1`). Arabic requires comprehensive handling of all **six grammatical categories** (`zero`, `one`, `two`, `few`, `many`, `other`).
4. **Thread-Safe Context Injection**:
   Resolved locale tags and printers (`*message.Printer` or `*i18n.Localizer`) must be propagated via `context.Context` using unexported private struct key types.
5. **Relational Storage Isolation (Persistence Strategy)**:
   Dynamic domain business data (e.g. Products, Accounts) must be stored in relational databases using **Sidecar Translation Tables** or **In-Row JSONB**, isolated from static application code catalogs.

---

## Architectural Topology: The Localization Flow

```text
┌──────────────────────────────────────────────────────────────────────────┐
│                         Incoming HTTP Request                            │
│           Accept-Language Header ─── Query Param (?lang=ar)              │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                       Negotiation Middleware (PEP)                       │
│      BCP 47 Matcher (golang.org/x/text/language) ─── Fallback Engine     │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                        Context Locale Injection                          │
│        ctx = WithLocale(ctx, resolvedTag, message.NewPrinter(tag))       │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                   Domain / Use Case / Persistence Layer                  │
│       Sidecar Queries: SELECT COALESCE(curr.name, fb.name) ...           │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      RFC 7807 Localized Problem Details                  │
│       Content-Language: ar-SA ─── { "code": "STOCK_INSUFFICIENT", ... }  │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Modular Documentation & Detailed Guides

To explore individual architectural components in depth, refer to the classified documents in `docs/`:

- [**01. BCP 47 Negotiation & Hierarchical Fallback**](docs/01_bcp47_negotiation_and_hierarchical_fallback.md): Details the BCP 47 standard, `language.Matcher`, confidence ratings, and the strict 6-stage precedence pipeline.
- [**02. Unicode CLDR & Arabic Pluralization Specification**](docs/02_unicode_cldr_and_arabic_pluralization.md): Comprehensive reference for Unicode CLDR plural rules, with detailed analysis of Arabic's 6 grammatical categories and regional number/currency formatting.
- [**03. PEP Middleware, Context Injection & RFC 7807 Guards**](docs/03_pep_middleware_and_rfc7807_guards.md): Production HTTP middleware design, collision-free context key typing, and RFC 7807 problem details error mapping.
- [**04. Multilingual Database Persistence & Architectural Boundaries**](docs/04_multilingual_database_persistence_and_boundaries.md): Detailed comparison of PostgreSQL JSONB vs. Sidecar tables vs. Odoo-style global dictionaries, plus the 5 mandatory backend translation scenarios.

---

## Production Code Examples

Ready-to-use, production-grade Go source files are available in `examples/`:

- [**`examples/i18n_manager_and_middleware.go`**](examples/i18n_manager_and_middleware.go): Complete thread-safe I18n Manager, HTTP negotiation middleware, context accessors, and RFC 7807 error writer.
- [**`examples/i18n_manager_test.go`**](examples/i18n_manager_test.go): Comprehensive unit tests covering the negotiation precedence matrix, dialect fallbacks, and concurrent goroutine race safety (`go test -race`).
- [**`examples/sidecar_repository.go`**](examples/sidecar_repository.go): PostgreSQL repository implementation querying multi-tenant sidecar translation tables with dual fallback.

---

## Standards & Official References

Authoritative standards, RFCs, and official documentation are indexed in:

- [**`references/official_references.md`**](references/official_references.md): Curated links to official Go blog articles, `golang.org/x/text` packages, Unicode CLDR specifications, and RFC 7807.

---

## Production Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails in Production | Correct Approach |
| :--- | :--- | :--- |
| Naive string prefix matching (`strings.HasPrefix(lang, "ar")`) | Fails to match dialects like `ar-EG` or complex scripts (`zh-Hant`) | Use official `language.NewMatcher` |
| Using primitive string context keys (`ctx.WithValue("lang", "ar")`) | Susceptible to context key collision across external packages | Declare an unexported struct key (`type localeKey struct{}`) |
| Binary pluralization (`if count == 1 ... else ...`) | Produces broken grammar in Arabic (which has 6 forms) and Slavic languages | Use Unicode CLDR plural engines (`go-i18n/v2` or `feature/plural`) |
| Sending raw localized UI errors in API payload | Couples backend to frontend UI copy; breaks mobile caching and localization | Return machine semantic keys (`STOCK_INSUFFICIENT`) and metadata |
| Hardcoding translations in database tables | Inflexible; cannot support full-text search dictionaries per language | Use Sidecar translation tables with language-specific FTS indexes |
| Translating outbound emails with operator's locale | Customer receives emails in the language of the admin who clicked the button | Always use the recipient customer's preferred language |

---

## Verification & Deployment Checklist

```text
[ ] Language negotiation validates BCP 47 tags using language.NewMatcher
[ ] Dialect fallback gracefully resolves parent language before default fallback
[ ] Unexported struct keys are used for all context value injections
[ ] Arabic message catalogs account for all 6 CLDR plural categories
[ ] Error responses conform to RFC 7807 (application/problem+json)
[ ] Concurrent safety verified under heavy load using `go test -race ./...`
[ ] Outbound PDFs and emails resolve the recipient's locale, not the operator's
[ ] Database entities implement Sidecar translation tables with language-specific FTS indexes
```
