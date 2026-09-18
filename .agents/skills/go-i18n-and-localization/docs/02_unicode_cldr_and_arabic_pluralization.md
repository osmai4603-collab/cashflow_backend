# Unicode CLDR & Arabic Pluralization Specification

---

## 1. Unicode CLDR Plural Rules Overview

Pluralization in natural languages cannot be represented by naive binary logic (`if count == 1`). The **Unicode Common Locale Data Repository (CLDR)** specifies six distinct plural categories:

- `zero`
- `one`
- `two`
- `few`
- `many`
- `other`

Different languages utilize different subsets of these categories:

- **English**: 2 categories (`one`, `other`).
- **French**: 2 categories (`one` [includes 0 and 1], `other`).
- **Russian / Polish**: 3 categories (`one`, `few`, `many`).
- **Arabic**: All **6 categories** (`zero`, `one`, `two`, `few`, `many`, `other`).

---

## 2. Arabic CLDR Pluralization Specification

In standard Arabic, the noun form changes radically depending on the integer count:

```text
┌──────────┬─────────────────────────────┬────────────────────────┬────────────────────────────────────────────┐
│ Category │ CLDR Condition              │ Integer Range          │ Example (Book / كتاب)                      │
├──────────┼─────────────────────────────┼────────────────────────┼────────────────────────────────────────────┤
│ zero     │ n = 0                       │ 0                      │ "لا توجد كتب" (No books)                   │
│ one      │ n = 1                       │ 1                      │ "كتاب واحد" (1 book)                       │
│ two      │ n = 2                       │ 2                      │ "كتابان اثنان" (2 books)                   │
│ few      │ n % 100 in 3..10            │ 3-10, 103-110, 203-210 │ "{{.Count}} كتب" (3-10 books, جمع قلة)    │
│ many     │ n % 100 in 11..99           │ 11-99, 111-199...      │ "{{.Count}} كتاباً" (11-99 books, تمييز مفرد منصوب)│
│ other    │ everything else             │ 100, 200, 1000, 1000000│ "{{.Count}} كتاب" (100 books, مفرد مجرور)  │
└──────────┴─────────────────────────────┴────────────────────────┴────────────────────────────────────────────┘
```

> [!IMPORTANT]
> Failure to account for the `few` (جمع قلة) and `many` (تمييز مفرد منصوب) categories in financial and enterprise reports produces ungrammatical outputs that undermine software quality.

---

## 3. Catalog Definition in `go-i18n/v2`

When authoring JSON or TOML translation catalogs for `github.com/nicksnyder/go-i18n/v2`, Arabic entries MUST provide all six branches:

```json
{
  "InvoiceCount": {
    "description": "Displays the number of outstanding invoices",
    "zero": "لا توجد فواتير مستحقة",
    "one": "توجد فاتورة واحدة مستحقة",
    "two": "توجد فاتورتان مستحقتان",
    "few": "توجد {{.Count}} فواتير مستحقة",
    "many": "توجد {{.Count}} فاتورةً مستحقة",
    "other": "توجد {{.Count}} فاتورة مستحقة"
  },
  "TransactionAmountRemaining": {
    "description": "Currency formatted balance warning",
    "zero": "تم سداد كامل المبلغ",
    "one": "المتبقي ريال واحد",
    "two": "المتبقي ريالان اثنان",
    "few": "المتبقي {{.Amount}} ريالات",
    "many": "المتبقي {{.Amount}} ريالاً",
    "other": "المتبقي {{.Amount}} ريال"
  }
}
```

---

## 4. Official Go Engine Integration: `golang.org/x/text/feature/plural`

The official `golang.org/x/text` sub-repository compiles CLDR tables directly into Go. When using `message.Printer`, plural rules are selected automatically based on the integer arguments:

```go
package main

import (
 "golang.org/x/text/feature/plural"
 "golang.org/x/text/language"
 "golang.org/x/text/message"
)

func init() {
 // Registering plural forms into the official message catalog
 _ = message.Set(language.Arabic, "UnreadNotificationCount",
  plural.Selectf(1, "%d",
   plural.Zero, "لا توجد إشعارات جديدة",
   plural.One, "لديك إشعار واحد جديد",
   plural.Two, "لديك إشعاران جديدان",
   plural.Few, "لديك %d إشعارات جديدة",
   plural.Many, "لديك %d إشعاراً جديداً",
   plural.Other, "لديك %d إشعار جديد",
  ),
 )
}

func PrintNotifications(count int) string {
 p := message.NewPrinter(language.Arabic)
 return p.Sprintf("UnreadNotificationCount", count)
}
```

---

## 5. Locale Data Formatting: Numbers, Currency, and BiDi

### Number and Decimal Separators (`x/text/number`)

- Arabic (Saudi Arabia / Eastern): Decimal separator is Arabic comma `٫` (U+066B), thousands separator is `٬` (U+066C).
- Arabic (North Africa): Uses western standard comma `,` and period `.`.
- Automated handling via `message.Printer` ensures formatting adheres to regional defaults.

### Currency Formatting (`x/text/currency`)

- ISO code `SAR` vs Symbol `ر.س`.
- Currency position (prefix vs suffix depending on locale rules).
