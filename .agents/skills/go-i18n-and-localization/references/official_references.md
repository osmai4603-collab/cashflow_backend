# Official Standards & Architecture References for Go i18n & Localization

This document catalogs the authoritative standards, official Go documentation, and reference implementations underpinning this internationalization and localization architecture.

---

## 1. Official Go Language Documentation (`go.dev`)

- [The Go Blog: Matchmaking with Language Tags](https://go.dev/blog/matchlang): Foundational architecture explanation by Marcel van Lohuizen on BCP 47 matching algorithms, confidence ratings, and hierarchical dialect resolution.
- [The Go Blog: Text Normalization in Go](https://go.dev/blog/normalization): Handling Unicode canonical decomposition, accents, and collation.
- [Go Official Sub-Repository `golang.org/x/text`](https://pkg.go.dev/golang.org/x/text): The core repository for text processing, Unicode CLDR data, and localization.
  - [`golang.org/x/text/language`](https://pkg.go.dev/golang.org/x/text/language): BCP 47 parsing and matching.
  - [`golang.org/x/text/message`](https://pkg.go.dev/golang.org/x/text/message): Localized formatted printing (`message.Printer`).
  - [`golang.org/x/text/feature/plural`](https://pkg.go.dev/golang.org/x/text/feature/plural): Unicode CLDR pluralization rules engine.
  - [`golang.org/x/text/cmd/gotext`](https://pkg.go.dev/golang.org/x/text/cmd/gotext): Official CLI toolchain for source code message extraction and AOT catalog compilation.
  - [`golang.org/x/text/collate`](https://pkg.go.dev/golang.org/x/text/collate): Culturally accurate linguistic sorting.
  - [`golang.org/x/text/currency`](https://pkg.go.dev/golang.org/x/text/currency): Currency formatting and symbols.

---

## 2. Formal International Standards

- **IETF BCP 47 (RFC 5646 & RFC 4647)**: *Tags for Identifying Languages* and *Matching of Language Tags*. The global standard for locale representation.
- **Unicode CLDR (Common Locale Data Repository)**: The worldwide industry standard for locale data, providing plural rules, month names, decimal separators, and collation data.
- **RFC 7807 / RFC 9457**: *Problem Details for HTTP APIs*. Standardized machine-readable error representations.
- **OASIS XLIFF 1.2 / 2.0**: *XML Localization Interchange File Format*. The global standard format for translation interchange between developer tools and CAT software.
- **GNU gettext**: Traditional translation framework utilizing `.po` and `.mo` message catalogs.

---

## 3. Leading Production Go Implementations

- [`github.com/nicksnyder/go-i18n/v2`](https://github.com/nicksnyder/go-i18n): The de facto enterprise standard for Go backend localization supporting CLDR plural rules, Go templates, JSON/TOML/YAML, and embedded filesystems (`embed.FS`).
- [`github.com/leonelquinteros/gotext`](https://github.com/leonelquinteros/gotext): Pure Go GNU gettext implementation for systems integrating with Poedit, Crowdin, or Transifex.
