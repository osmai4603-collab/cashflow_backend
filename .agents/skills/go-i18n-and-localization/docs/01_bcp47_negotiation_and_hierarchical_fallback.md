# BCP 47 Language Negotiation & Hierarchical Fallback

---

## 1. Overview & Standard Specifications

Internationalization in Go backends begins with deterministic language identification. The standard accepted protocol globally is **IETF BCP 47** (encompassing **RFC 5646** and **RFC 4647**), which defines the canonical syntax for language tags and matching algorithms.

A BCP 47 language tag consists of one or more subtags:
`language-Script-REGION-variant-extension-privateuse`

- **Language**: 2 or 3-letter ISO 639 code (`ar`, `en`, `zh`).
- **Script**: 4-letter ISO 15924 code (`Arab`, `Latn`, `Hans`, `Hant`).
- **Region**: 2-letter ISO 3166-1 country code or 3-digit UN M.49 region code (`SA`, `EG`, `US`, `GB`, `419`).

The official Go text repository provides native BCP 47 parsing and matching via:

```go
import "golang.org/x/text/language"
```

---

## 2. The Language Matcher & Confidence Scores

The `language.Matcher` interface implements the Unicode Locale Matching algorithm. Unlike basic string prefix comparison, `language.Matcher` understands dialect trees, macro-languages, and regional affinities.

When comparing a requested language against supported server tags, it evaluates a **Confidence Score**:

| Confidence Level | Value | Meaning |
| :--- | :--- | :--- |
| `language.Exact` | 3 | Exact match (e.g., requested `ar-SA` and server supports `ar-SA`). |
| `language.High` | 2 | Close linguistic affinity (e.g., requested `ar-EG` and server supports `ar` or related regional dialect). |
| `language.Low` | 1 | Macro-language or distant parent match. |
| `language.No` | 0 | No relationship. Fallback tag is selected. |

### Hierarchical Fallback Examples

1. Client requests `ar-KW` (Kuwait): Server supports `[en-US, ar, fr]`. Matcher resolves `ar` with `High` confidence.
2. Client requests `zh-HK` (Traditional Chinese, Hong Kong): Server supports `[zh-Hans, zh-Hant]`. Matcher resolves `zh-Hant` (Traditional Chinese) rather than falling back to English or Simplified Chinese.
3. Client requests `de-AT` (Austrian German): Server supports `[en, ar]`. Matcher yields `language.No` and returns the designated default fallback (`en`).

---

## 3. Strict Precedence Pipeline for HTTP / gRPC Backends

To prevent client spoofing while supporting flexible overrides, the negotiation pipeline MUST evaluate language sources in the following strict order of precedence:

```text
┌────────────────────────────────────────────────────────────────────────┐
│ 1. URL Query Parameter (?lang=ar-SA)   Highest Priority (Ad-hoc test) │
├────────────────────────────────────────────────────────────────────────┤
│ 2. Custom Protocol Header (X-App-Language: ar-SA)                      │
├────────────────────────────────────────────────────────────────────────┤
│ 3. Standard Protocol Header (Accept-Language: ar-SA,ar;q=0.9,en;q=0.8) │
├────────────────────────────────────────────────────────────────────────┤
│ 4. Authenticated User Profile Preference (from JWT Claims / Session)   │
├────────────────────────────────────────────────────────────────────────┤
│ 5. Tenant / Organization Default Language (Multi-Tenant Setting)       │
├────────────────────────────────────────────────────────────────────────┤
│ 6. Global Server Default Fallback (Fail-safe default: e.g., ar-SA)     │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Production Go Implementation

```go
package i18n

import (
 "net/http"
 "golang.org/x/text/language"
)

// SupportedLocales defines all languages compiled into the service.
var SupportedLocales = []language.Tag{
 language.MustParse("ar-SA"), // Primary default
 language.Arabic,             // General Arabic (ar)
 language.AmericanEnglish,    // en-US
 language.English,            // General English (en)
 language.French,             // fr
}

// ServerMatcher is the thread-safe, pre-warmed BCP 47 matcher.
var ServerMatcher = language.NewMatcher(SupportedLocales)

// ResolveRequestLocale determines the best language.Tag from an HTTP request.
func ResolveRequestLocale(r *http.Request, defaultTag language.Tag) (language.Tag, language.Confidence) {
 var candidates []language.Tag

 // Step 1: Explicit URL parameter (?lang=...)
 if q := r.URL.Query().Get("lang"); q != "" {
  if tag, err := language.Parse(q); err == nil {
   candidates = append(candidates, tag)
  }
 }

 // Step 2: Custom Application Header (X-App-Language)
 if custom := r.Header.Get("X-App-Language"); custom != "" && len(candidates) == 0 {
  if tag, err := language.Parse(custom); err == nil {
   candidates = append(candidates, tag)
  }
 }

 // Step 3: Standard Accept-Language header
 if len(candidates) == 0 {
  if accept := r.Header.Get("Accept-Language"); accept != "" {
   tags, _, _ := language.ParseAcceptLanguage(accept)
   candidates = append(candidates, tags...)
  }
 }

 // Step 4: If candidates exist, execute BCP 47 matching
 if len(candidates) > 0 {
  matched, _, confidence := ServerMatcher.Match(candidates...)
  return matched, confidence
 }

 // Step 5: Global fallback
 return defaultTag, language.No
}
```
