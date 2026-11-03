# الخطة التنفيذية التفصيلية — المرحلة 0: البنية التحتية، الدقة الحسابية وترحيل قاعدة البيانات

## (Phase 0: Foundation, Precision & Database Schema)

> **مسار التوثيق:** `docs/impl/phase_0_foundation_and_migrations.md`  
> **الحالة:** معتمدة للتنفيذ الفوري  
> **الهدف:** تجهيز مكتبات الدقة العشرية، معايير العملات وفق ISO 4217، وتطبيق سكربتات ترحيل قاعدة البيانات الشاملة لجميع جداول وحدة المحاسبة المالية.

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. اعتماد مكتبة الحسابات العشرية الدقيقة `github.com/shopspring/decimal` ومنع أي استخدام للأعداد العائمة `float32/float64`.
2. إنشاء حزمة منصة موحدة للعملات `internal/platform/money` تحدد أدق وحدة كسرية (Minor Unit Exponent) وفق معيار ISO 4217.
3. كتابة ملف الترحيل الشامل `000002_create_accounting_schema.up.sql` وملف الإلغاء العكسي `000002_create_accounting_schema.down.sql`.
4. التحقق من تطبيق الترحيل والتراجع بنجاح على قاعدة بيانات PostgreSQL.

---

## 2. الإجراءات الفعلية خطوة بخطوة

### الخطوة 1: تثبيت وتحديث الاعتماديات

تنفيذ الأمر التالي في الطرفية:

```bash
go get github.com/shopspring/decimal
go get github.com/google/uuid
go mod tidy
```

### الخطوة 2: حزمة العملات والدقة الحسابية (`internal/platform/money/money.go`)

إنشاء الحزمة لتوفير القواعد الحسابية الآمنة:

```go
package money

import (
 "errors"
 "strings"

 "github.com/shopspring/decimal"
)

var (
 ErrInvalidCurrency = errors.New("unsupported or invalid ISO 4217 currency")
 ErrNegativeAmount  = errors.New("negative amount is not permitted")
)

// CurrencySpec مواصفات العملة وفق معيار ISO 4217
type CurrencySpec struct {
 Code        string
 Exponent    int32 // عدد الخانات الكسرية (2 للريال والدولار، 3 للدينار، 0 للين)
 Description string
}

var SupportedCurrencies = map[string]CurrencySpec{
 "SAR": {Code: "SAR", Exponent: 2, Description: "Saudi Riyal"},
 "USD": {Code: "USD", Exponent: 2, Description: "US Dollar"},
 "EUR": {Code: "EUR", Exponent: 2, Description: "Euro"},
 "GBP": {Code: "GBP", Exponent: 2, Description: "British Pound"},
 "KWD": {Code: "KWD", Exponent: 3, Description: "Kuwaiti Dinar"},
 "BHD": {Code: "BHD", Exponent: 3, Description: "Bahraini Dinar"},
 "JPY": {Code: "JPY", Exponent: 0, Description: "Japanese Yen"},
}

// RoundToCurrency تقريب المبلغ بدقة طبقا لعدد خانات العملة
func RoundToCurrency(amount decimal.Decimal, currencyCode string) (decimal.Decimal, error) {
 cur, ok := SupportedCurrencies[strings.ToUpper(currencyCode)]
 if !ok {
  return decimal.Zero, ErrInvalidCurrency
 }
 return amount.Round(cur.Exponent), nil
}
```

### الخطوة 3: ملف ترحيل قاعدة البيانات الصاعد (`migrations/000002_create_accounting_schema.up.sql`)

إنشاء الجداول والفهارس والقيود الصارمة:

```sql
-- 1. جدول دليل الحسابات
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    account_type VARCHAR(50) NOT NULL, -- asset, liability, equity, income, expense
    normal_balance VARCHAR(6) NOT NULL, -- debit, credit
    currency_id VARCHAR(3) NOT NULL DEFAULT 'SAR',
    parent_id UUID REFERENCES accounts(id) ON DELETE RESTRICT,
    is_reconcilable BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_accounts_code UNIQUE (code),
    CONSTRAINT chk_account_type CHECK (account_type IN ('asset', 'liability', 'equity', 'income', 'expense')),
    CONSTRAINT chk_normal_balance CHECK (normal_balance IN ('debit', 'credit'))
);

CREATE INDEX IF NOT EXISTS idx_accounts_code ON accounts(code);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(account_type);
CREATE INDEX IF NOT EXISTS idx_accounts_parent ON accounts(parent_id);

-- 2. جدول الفترات المالية والسنوات
CREATE TABLE IF NOT EXISTS fiscal_years (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    year INT NOT NULL UNIQUE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'open', -- open, closed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fiscal_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fiscal_year_id UUID NOT NULL REFERENCES fiscal_years(id) ON DELETE RESTRICT,
    period_number INT NOT NULL,
    name VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'open', -- open, soft_closed, hard_closed
    closed_at TIMESTAMPTZ,
    closed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_fiscal_period_num UNIQUE (fiscal_year_id, period_number),
    CONSTRAINT chk_period_state CHECK (state IN ('open', 'soft_closed', 'hard_closed'))
);

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_dates ON fiscal_periods(start_date, end_date);

-- 3. جدول دفاتر اليومية وسلاسل الترقيم القانوني
CREATE TABLE IF NOT EXISTS journals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL, -- general, bank, cash, sale, purchase
    sequence_prefix VARCHAR(20) NOT NULL,
    next_number BIGINT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_journal_type CHECK (type IN ('general', 'bank', 'cash', 'sale', 'purchase'))
);

-- 4. جدول القيود اليومية (الأستاذ العام)
CREATE TABLE IF NOT EXISTS account_moves (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL, -- الرقم التسلسلي القانوني (مثل MISC/2026/00001)
    journal_id UUID NOT NULL REFERENCES journals(id) ON DELETE RESTRICT,
    period_id UUID NOT NULL REFERENCES fiscal_periods(id) ON DELETE RESTRICT,
    date DATE NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, posted, reversed
    ref VARCHAR(255),
    reversed_move_id UUID REFERENCES account_moves(id) ON DELETE RESTRICT,
    hash_signature VARCHAR(128),
    posted_at TIMESTAMPTZ,
    posted_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_move_state CHECK (state IN ('draft', 'posted', 'reversed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_posted_move_name ON account_moves(name) WHERE state = 'posted';
CREATE INDEX IF NOT EXISTS idx_moves_date_state ON account_moves(date, state);
CREATE INDEX IF NOT EXISTS idx_moves_journal ON account_moves(journal_id);

-- 5. جدول أسطر القيود المالية
CREATE TABLE IF NOT EXISTS account_move_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    move_id UUID NOT NULL REFERENCES account_moves(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    partner_id UUID,
    debit NUMERIC(28, 4) NOT NULL DEFAULT 0,
    credit NUMERIC(28, 4) NOT NULL DEFAULT 0,
    amount_currency NUMERIC(28, 4),
    currency_id VARCHAR(3) NOT NULL DEFAULT 'SAR',
    cost_center_id UUID,
    matching_number VARCHAR(64),
    date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_line_debit_positive CHECK (debit >= 0),
    CONSTRAINT chk_line_credit_positive CHECK (credit >= 0),
    CONSTRAINT chk_line_has_value CHECK (debit > 0 OR credit > 0)
);

CREATE INDEX IF NOT EXISTS idx_lines_account_date ON account_move_lines(account_id, date);
CREATE INDEX IF NOT EXISTS idx_lines_move_id ON account_move_lines(move_id);
CREATE INDEX IF NOT EXISTS idx_lines_matching ON account_move_lines(matching_number);

-- 6. جدول سجل التدقيق المالي المشفر (SOX Compliance)
CREATE TABLE IF NOT EXISTS financial_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL, -- CREATE, POST, REVERSE, LOCK
    actor_id UUID NOT NULL,
    payload_before JSONB,
    payload_after JSONB,
    ip_address INET,
    previous_log_hash VARCHAR(64),
    current_log_hash VARCHAR(64) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON financial_audit_logs(entity_type, entity_id);

-- 7. جدول سجلات استيعاب الأحداث الاقتصادية (Idempotency)
CREATE TABLE IF NOT EXISTS subledger_processed_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    move_id UUID NOT NULL REFERENCES account_moves(id) ON DELETE RESTRICT,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### الخطوة 4: ملف التراجع العكسي (`migrations/000002_create_accounting_schema.down.sql`)

```sql
DROP TABLE IF EXISTS subledger_processed_events CASCADE;
DROP TABLE IF EXISTS financial_audit_logs CASCADE;
DROP TABLE IF EXISTS account_move_lines CASCADE;
DROP TABLE IF EXISTS account_moves CASCADE;
DROP TABLE IF EXISTS journals CASCADE;
DROP TABLE IF EXISTS fiscal_periods CASCADE;
DROP TABLE IF EXISTS fiscal_years CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [ ] تمرير أوامر تشغيل الترحيل بنجاح دون أي خطأ في بنية الـ SQL.
- [ ] التأكد من دعم نوع `NUMERIC(28, 4)` وتوافق دقة العملات في اختبارات `internal/platform/money`.
- [ ] إجراء تجربة إلغاء الترحيل (`down`) وإعادة تطبيقه (`up`) للتأكد من قدرة التراجع بنسبة 100%.
