# المرحلة 10 — المحاسبة التحليلية (Analytic Accounting): الخطة التنفيذية

**المرجع**: Odoo 19.0 `addons/analytic/models/` (مراجعة مُعمّقة)
**يعتمد على**: المرحلة 3 (المحاسبة الأساسية)
**التقدير المُحدَّث**: **~5 أيام** (بدلاً من 3) — بسبب الحقائق الإضافية المستخلصة من كود Odoo
**الأولوية**: 🔴 حرج

---

## 1. خلاصة مراجعة مصدر Odoo 19.0

تمت مراجعة كامل موديول `analytic` في Odoo 19.0 (ملفات `.py` + بيانات seed + أذونات). الموديول الحقيقي **أغنى بكثير** من خطة `advanced__infrastructured_implementation_plan.md`. الملفات الفعلية:

| ملف Odoo | الغرض |
| ---------- | ------- |
| `analytic_plan.py` | الكيانات `account.analytic.plan` + `account.analytic.applicability` + آلية الأعمدة الديناميكية لكل خطة |
| `analytic_account.py` | كيان `account.analytic.account` + حساب debit/credit/balance مجمّع |
| `analytic_line.py` | كيان `account.analytic.line` + Mixin `analytic.plan.fields.mixin` (عمود لكل خطة) |
| `analytic_distribution_model.py` | كيان `account.analytic.distribution.model` (قاعدة توزيع تلقائية) |
| `analytic_mixin.py` | Mixin `analytic.mixin` → حقل JSON `analytic_distribution` على كل موديل |

---

## 2. الفجوات الحرجة بين الخطة الأصلية وبين Odoo (يجب معالجتها)

الخطة الحالية في `advanced__infrastructured_implementation_plan.md` تُبسّط النظام بشكل كبير. فيما يلي الفجوات التي يجب سدّها:

### G1 — نوع البيانات `decimal.Decimal` غير مستخدم في المشروع

الخطة تستخدم `decimal.Decimal` لكن المشروع الحالي (بما فيه كيان `AccountMove`) يستخدم **`float64`**. يجب توحيد النوع إلى `float64` وتطبيق تقريب 4 خانات (كما في `move.go`).

### G2 — كيان `AnalyticApplicability` مفقود (الأهم من الناحية التشغيلية للتواجد المتعدد الشركات)

الخطة تضع حقل `DefaultApplicability` فقط داخل الخطة. Odoo يفصل قاعدة الانطباق في كيان مستقل `account.analytic.applicability` بمشغّل "كل شركة" (`company_dependent`) مع:

- `BusinessDomain` (قيمة قابلة للتوسّع: `general`, `sale_order`, `purchase_order`, `expense`, `timesheet`...)
- `Applicability` : `optional` / `mandatory` / `unavailable`
- **`CompanyID` موسّع** — أي قاعدة قد تكون عامة أو خاصة بشركة
- **آلية التقييم (score)**: القاعدة التي تحصل على أعلى نقاط هي المُطبَّقة (`_get_score`)، مع ترجيح `company_id` بـ 0.5 ونقطة مقابل `business_domain`.
- `Mandatory` في Odoo تُجبر على اكتمال 100% من التوزيع (تحقّق `_validate_distribution`).

**النتيجة العملية**: عند تشغيل `get_relevant_plans` يتم تحديد الخطة المسموحة/الإلزامية/المحظورة لكل سياق (شركة + مجال أعمال). هذا هو جوهر النظام.

### G3 — آلية `analytic_distribution` (JSON على كل موديل) مفقودة/مبسَّطة جداً

الخاتة تتعامل مع `AnalyticDistribution` كجدول منفصل. في Odoo الحقيقة **JSON map** على كل سطر قابل للتحليل:

```json
{ "<account_id>": 100, "<account_id2>": 50 }
```

المفاتيح قد تكون مفصولة بفواصل لتمثيل **شراكات (combination)** من حسابات من عدّة خطط معاً، ودوال التوزيع هي نسب (sum=100%).
أهم الدوال في `analytic_mixin.py`:

- `_merge_distribution` — دمج توزيع قديم مع جديد مع الحفاظ على القِيم غير المتغيّرة (نسبة-تقليص).
- `_validate_distribution` — فرض 100% للخطط الإلزامية.
- `distribution_analytic_account_ids` — m2m محسوب يستخرج الحسابات من JSON.
- بحث عبر GIN index على مفاتيح JSON.
- `_split_amount_fname` example: عند توزيع سطر بمبلغ، تُقسّم القيمة عبر عدّة أسطر تحليلية (كل سطر بمبلغ × نسبة/100).

### G4 — آلية "عمود لكل خطة" (Mixin `analytic.plan.fields.mixin`) مفقودة

- بنية Odoo تتطلب **عمود واحد لكل خطة** على سطر التحليل (`account_id` للنوع "Project plan" + `x_plan{id}_id` للخطط الأخرى — عمود ديناميكي).
- يوجد حقل "سحري" `auto_account_id` يتغيّر حسب الخطة في السياق.
- قيد `_check_account_id`: **يجب تعيين حساب تحليلي واحد على الأقل**.
- هذا يؤثّر على كل موديلات الاندماج المستقبلية (invoice line, SO line, PO line, expense...) لذا يجب تصميم مخطط الجداول بحيث يستوعب Multi-plan منذ البداية.

### G5 — حقول مفقودة في `AnalyticLine`

مقارنة بخطة الحالية، السطر الحقيقي يضم:

- `name` (شرح العملية) — **إلزامي** في Odoo لكن الخطة لا تذكره.
- `user_id` (من سجّل/المسؤول).
- `product_uom_id` (وحدة القياس).
- `currency_id` (عملة مرتبطة بالشركة — مهم لحساب الأرصدة).
- `category` (قيمة واحدة فقط `other` حالياً).
- `display`/`balance` على مستوى الحساب تُحسب بعملة الشركة (`Convert`).

### G6 — `debit` و `credit` منفصلتان (ليست `balance` فقط)

`AccountAnalyticAccount` يحسب ثلاث قيم: `debit`, `credit`, `balance` (بـ Convert عملة لكل تاريخ). الخطة تكتفي بـ `Balance`.

### G7 — `AnalyticDistributionModel` (قاعدة التوزيع التلقائية) مفقودة كلياً

`account.analytic.distribution.model`:

- عند إنشاء أمر/فاتورة/مصروف، يستخدم النظام **أفضل قاعدة مطابقة** (حسب `partner_id`, `partner_category_id`, `company_id`, `sequence`) ليُعبّئ حقل `analytic_distribution` تلقائياً.
- ترتيب المطابقة: `partner_id` ثم `partner_category_id` ثم فارغ (عام)، مع `sequence`.
- تعديل القِيَم يُجبِر القيد `_check_company_accounts` (لا يمكن استخدام حسابات خاصة بشركة في قاعدة عامة).

### G8 — إعداد "خطة المشاريع" (`project plan`) مفقود

- Odoo يعيّن خطة مميزة `analytic.project_plan` عبر `ir.config_parameter` (id=1 افتراضياً "Project") — هذا العمود هو `account_id` الثابت.
- `_get_all_plans` يفصل خطة المشاريع عن باقي الخطط. هذا له أهمية كبيرة لاحقاً في المرحلة 17 (إدارة المشاريع).

### G9 — BOM/السّياق: قيمة seed و precision

- Odoo يُعرّف `decimal.precision` باسم `Percentage Analytic` بـ 2 خانات.
- بيانات seed تخلق خطة "Project" الأساسية.
- الأذونات عبر `ir.model.access.csv` (قراءة/كتابة على كل نموذج تحليلي حسب مجموعة `analytic.group_analytic_accounting`).

---

## 3. الكيانات المُحدَّثة (Go — متوافقة مع معايير المشروع)

النوع الأساسي: `float64` (وليس `decimal.Decimal`) + `audit.Fields` (نمط المشروع).

### 3.1 `internal/domain/analytic/analytic.go`

```go
package analytic

import "time"

// Applicability — قيمة الانطباق (G2)
type Applicability string
const (
    AppOptional   Applicability = "optional"
    AppMandatory  Applicability = "mandatory"
    AppUnavailable Applicability = "unavailable"
)

// BusinessDomain — مجال الأعمال الذي تنطبق عليه القاعدة (قابل للتوسّع)
type BusinessDomain string
const (
    DomainGeneral       BusinessDomain = "general"
    DomainSaleOrder     BusinessDomain = "sale_order"     // تُضاف لاحقاً (M4)
    DomainPurchaseOrder BusinessDomain = "purchase_order" // تُضاف لاحقاً (M5)
    DomainExpense       BusinessDomain = "expense"        // تُضاف لاحقاً (M19)
)

// AnalyticPlan — خطة تحليلية (مطابقة لـ account.analytic.plan)
type AnalyticPlan struct {
    ID                int64
    Name              string
    Description       string
    ParentID          *int64
    ParentPath        string
    RootID            int64
    CompleteName      string   // حسابي: "الوالد / الاسم"
    Sequence          int
    Color             int
    DefaultApplicability Applicability // company-dependent افتراضياً
    Applicabilities   []AnalyticApplicability // كل الشركات
    Active            bool
    Audit             audit.Fields
}

// AnalyticApplicability — قاعدة انطباق مستقلة (G2)
type AnalyticApplicability struct {
    ID               int64
    PlanID           int64
    BusinessDomain   BusinessDomain
    Applicability    Applicability
    CompanyID        *int64 // nil = لكل الشركات
    Sequence         int
}

// AnalyticAccount — حساب تحليلي (account.analytic.account)
type AnalyticAccount struct {
    ID          int64
    Name        string
    Code        string // مرجع اختياري
    PlanID      int64
    RootPlanID  int64
    PartnerID   *int64
    Color       int
    CompanyID   *int64
    Active      bool
    Debit       float64 // محسوب (بعملة الشركة)
    Credit      float64 // محسوب
    Balance     float64 // محسوب = credit - debit
    Currency    string  // عملة الشركة
    Audit       audit.Fields
}

// AnalyticLine — سطر تحليلي (account.analytic.line)
type AnalyticLine struct {
    ID         int64
    Name       string       // شرح العملية (G5)
    Date       time.Time
    Amount     float64
    UnitAmount float64      // الكمية
    ProductUoMID *int64
    PartnerID  *int64
    UserID     int64
    CompanyID  int64
    Currency   string       // عملة الشركة
    Category   string       // "other"
    AccountIDs []int64      // حساب لكل خطة (G4: [account_id, x_plan2_id, ...])
    MoveLineID *int64       // ربط بسطر القيد المحاسبي
    GeneralAccountID *int64 // الحساب العام
    Sources    LineSource   // invoice, vendor_bill, manual, employee, sale_order...
    Audit      audit.Fields
}

type LineSource string
const (
    SourceManual   LineSource = "manual"
    SourceInvoice  LineSource = "invoice"
    SourceVendorBill LineSource = "vendor_bill"
    SourceSaleOrder  LineSource = "sale_order"
    SourceEmployee   LineSource = "employee"
)
```

### 3.2 `internal/domain/analytic/distribution.go` (G3, G7)

```go
package analytic

// AnalyticDistribution — خريطة JSON {account_ids: percentage}
// تُمثَّل في DB كـ JSONB. المفتاح قد يكون "1,5" (شراكة حسابات من عدّة خطط).
type AnalyticDistribution map[string]float64

// DistributionModel — قاعدة التوزيع التلقائي (account.analytic.distribution.model) (G7)
type DistributionModel struct {
    ID                int64
    Sequence          int
    PartnerID         *int64
    PartnerCategoryID *int64
    CompanyID         *int64
    Distribution      AnalyticDistribution
    Active            bool
    Audit             audit.Fields
}

// MergeResult — نتيجة دمج توزيعين (تنفيذ _merge_distribution)
type MergeResult struct {
    Merged AnalyticDistribution
    Changed bool
}
```

- `Validate(100%)` دالة تُرجع خطأ إذا مجموع نسب التوزيع للخطة الإلزامية ≠ 100% (ما يعادل `_validate_distribution`).
- `Merge(old, new, plans)` تنفيذ `_merge_distribution`.
- `MapFromAccountIDs([]int64)` → `{"id": 100}` / و `DistributionKey(ids)` → `"1,5"`.

---

## 4. واجهة التخزين (Ports)

```go
// internal/domain/analytic/ports.go
type Repository interface {
    // Plans
    CreatePlan(ctx context.Context, p *AnalyticPlan) error
    GetPlanByID(ctx context.Context, id int64) (*AnalyticPlan, error)
    UpdatePlan(ctx context.Context, p *AnalyticPlan) error
    DeletePlan(ctx context.Context, id int64) error
    ListPlans(ctx context.Context, includeInactive bool) ([]AnalyticPlan, error)
    GetChildrenPlans(ctx context.Context, parentID int64) ([]AnalyticPlan, error)

    // Applicabilities (G2)
    SetApplicability(ctx context.Context, a *AnalyticApplicability) error
    GetApplicabilities(ctx context.Context, planID int64) ([]AnalyticApplicability, error)
    // يعيد الخطط ذات الصلة بسياق (شركة + مجال أعمال) — ما يعادل get_relevant_plans
    GetRelevantPlans(ctx context.Context, companyID int64, businessDomain BusinessDomain) ([]RelevantPlan, error)

    // Accounts
    CreateAccount(ctx context.Context, a *AnalyticAccount) error
    GetAccountByID(ctx context.Context, id int64) (*AnalyticAccount, error)
    UpdateAccount(ctx context.Context, a *AnalyticAccount) error
    DeleteAccount(ctx context.Context, id int64) error
    ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[AnalyticAccount], error)
    GetAccountTotals(ctx context.Context, accountID int64, fromDate, toDate *time.Time) (DebitCreditBalance, error)

    // Lines
    CreateLine(ctx context.Context, l *AnalyticLine) error
    CreateLines(ctx context.Context, lines []AnalyticLine) error // دفعة
    GetLineByID(ctx context.Context, id int64) (*AnalyticLine, error)
    UpdateLine(ctx context.Context, l *AnalyticLine) error
    ListLines(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[AnalyticLine], error)

    // Distribution Models (G7)
    CreateDistributionModel(ctx context.Context, m *DistributionModel) error
    GetDistributionModelByID(ctx context.Context, id int64) (*DistributionModel, error)
    UpdateDistributionModel(ctx context.Context, m *DistributionModel) error
    DeleteDistributionModel(ctx context.Context, id int64) error
    ListDistributionModels(ctx context.Context) ([]DistributionModel, error)
    // أفضل قاعدة مطابقة لسياق (شريك/فئة/شركة) — ما يعادل _get_distribution
    MatchDistribution(ctx context.Context, partnerID, partnerCategoryID *int64, companyID int64) (AnalyticDistribution, error)

    // Project Plan (G8)
    GetProjectPlanID(ctx context.Context) (int64, error)
    SetProjectPlanID(ctx context.Context, planID int64) error
}
```

نوع مساعد:

```go
type RelevantPlan struct {
    ID            int64
    Name          string
    Applicability Applicability
    ColumnName    string
}
```

---

## 5. منطق الاستخدام (Usecase)

`internal/usecase/analytic/analytic_usecase.go` — يحوي:

1. **`ReconcileLineDistribution(line, distribution)`** (G3)
   تنفيذ `_split_amount_fname`: يوزّع المبلغ عبر خطوط متعددة بنسبة كل حساب، ويُنشئ سطراً واحداً لكل شريحة.
2. **`ValidateDistribution(lines, companyID)`** — فرض 100% على الخطط الإلزامية في السياق.
3. **`GetRelevantPlans(ctx, companyID, businessDomain)`** — يجمع الخطط الجذرية ذات الحسابات > 0 والمسموحة، مع دفع الخطط "المفروضة سابقاً" إلى `optional` (سلوك `forced_plans` في Odoo).
4. **`AutoCompleteDistribution(ctx, partnerID, categoryID, companyID, existing)`** (G7) — يسحب أفضل `DistributionModel` مطابق ويُدمجها مع التوزيع الموجود.
5. **`ComputeAccountTotals`** — حساب `debit/credit/balance` بتجميع الأسطر (مع تحويل العملة لكل تاريخ).
6. **`CreateLinesFromMoveLine`** — تكامل مع المرحلة 3: عند ترحيل قيد بخطوط تحمل `analytic_distribution`، تُطابق/تُنشأ أسطر التحليل وربطها بـ `move_line_id`.
7. **`RegisterManualLine`** — إدخال يدوي سطر تحليلي مع إنشاء/عدم إنشاء قيد محاسبي حسب الحاجة.

---

## 6. نِقاط API (المُحدَّثة)

```
# Analytic Plans
CRUD   /api/v1/analytic-plans
GET    /api/v1/analytic-plans/{id}/structure       (G4) بنية الخطة + كل الحسابات
PUT    /api/v1/analytic-plans/{id}/applicability   (G2) تعيين قاعدة انطباق لشركة/مجال

# Analytic Accounts
CRUD   /api/v1/analytic-accounts
GET    /api/v1/analytic-accounts/{id}/balance      رصيد + debit + credit (بعملة الشركة)
GET    /api/v1/analytic-accounts/{id}/lines

# Analytic Lines
POST   /api/v1/analytic-lines
GET    /api/v1/analytic-lines          فلترة بالتاريخ/الحساب/الشريك/الشركة
GET    /api/v1/analytic-lines/{id}
PUT    /api/v1/analytic-lines/{id}

# Distribution Models (G7) — جديد
CRUD   /api/v1/analytic-distribution-models
POST   /api/v1/analytic-distribution-models/match   جلب أفضل قاعدة مطابقة

# Integrations
GET    /api/v1/analytic/relevant-plans?company_id=&business_domain=   (G2)
GET    /api/v1/move-lines/{id}/analytic            أسطر التحليل المرتبطة بقيد محاسبي
```

---

## 7. بنية الملفات (النمط الفعلي للمشروع)

```
internal/domain/analytic/
├── analytic.go            # Entities + Applicability + Accounts + Lines (G1, G2, G5, G6)
├── distribution.go        # AnalyticDistribution + DistributionModel (G3, G7)
├── ports.go               # Repository interface
├── analytic_test.go
internal/usecase/analytic/
├── analytic_usecase.go    # Business logic (توزيع/تحقق/ملاءمة)
├── analytic_usecase_test.go
internal/adapters/http/analytic/
├── handler.go
├── dto.go
├── routes.go
├── handler_test.go
internal/adapters/storage/analytic/
├── postgres_repo.go
├── memory_repo.go
migrations/
├── 000012_create_analytic_schema.up.sql   # NEW (بعد 011)
├── 000012_create_analytic_schema.down.sql
```

> [!NOTE]
> ترقيم الـ migration: عند المسح الفعلي لمجلد `migrations/` وُجد أن آخر هجرة حالية هي `000011_create_core_infrastructure_schema` (وليس `000013` كما افترضنا في المخطّط). إذن الهجرة الجديدة هي **`000012_create_analytic_schema`**. الهجرات تُضمَّن تلقائياً عبر `//go:embed *.sql` في `migrations/fs.go` (لا حاجة لتعديل الكود).

---

## 8. مخطط قاعدة البيانات (PostgreSQL — جاهزة للـ multi-plan & multi-company)

```sql
-- 000012_create_analytic_schema.up.sql

CREATE TABLE IF NOT EXISTS account_analytic_plan (
    id                      BIGSERIAL PRIMARY KEY,
    name                    TEXT NOT NULL,
    description             TEXT,
    parent_id               BIGINT REFERENCES account_analytic_plan(id) ON DELETE CASCADE,
    parent_path             TEXT,
    root_id                 BIGINT,
    sequence                INT DEFAULT 10,
    color                   INT,
    default_applicability   TEXT,          -- optional|mandatory|unavailable
    active                  BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_analytic_plan_parent ON account_analytic_plan(parent_id);
CREATE INDEX idx_analytic_plan_root ON account_analytic_plan(root_id);

-- G2: قواعد الانطباق المستقلة (لكل شركة/مجال)
CREATE TABLE IF NOT EXISTS account_analytic_applicability (
    id                BIGSERIAL PRIMARY KEY,
    analytic_plan_id  BIGINT NOT NULL REFERENCES account_analytic_plan(id) ON DELETE CASCADE,
    business_domain   TEXT NOT NULL DEFAULT 'general',
    applicability     TEXT NOT NULL,
    company_id        BIGINT,             -- NULL = لكل الشركات
    sequence          INT DEFAULT 10,
    audit_created_at  TIMESTAMPTZ,
    audit_created_by  BIGINT,
    audit_updated_at  TIMESTAMPTZ,
    audit_updated_by  BIGINT
);
CREATE INDEX idx_analytic_applicability_plan ON account_analytic_applicability(analytic_plan_id);

CREATE TABLE IF NOT EXISTS account_analytic_account (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    code           TEXT,
    plan_id        BIGINT NOT NULL REFERENCES account_analytic_plan(id),
    root_plan_id   BIGINT,
    partner_id     BIGINT,
    color          INT,
    company_id     BIGINT,               -- NULL = لكل الشركات
    active         BOOLEAN DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_analytic_account_plan ON account_analytic_account(plan_id);
CREATE INDEX idx_analytic_account_partner ON account_analytic_account(partner_id);

-- G4: سطر تحليلي بعمود واحد لكل خطة + JSON للتوزيع
CREATE TABLE IF NOT EXISTS account_analytic_line (
    id                   BIGSERIAL PRIMARY KEY,
    name                 TEXT NOT NULL,                 -- G5
    date                 DATE NOT NULL,
    amount               DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit_amount          DOUBLE PRECISION NOT NULL DEFAULT 0,
    product_uom_id       BIGINT,
    partner_id           BIGINT,
    user_id              BIGINT NOT NULL,
    company_id           BIGINT NOT NULL,
    currency_code        TEXT,
    category             TEXT NOT NULL DEFAULT 'other',
    -- أعمدة الخطط: account_id (Project plan) + x_plan{id}_id لكل خطة أخرى (G4)
    account_id           BIGINT REFERENCES account_analytic_account(id) ON DELETE RESTRICT,
    -- G3: التوزيع JSONB المفتاح = account_ids مفصولة بفواصل، القيمة = النسبة
    analytic_distribution JSONB,
    move_line_id         BIGINT,          -- ربط بسطر القيد المحاسبي
    general_account_id   BIGINT,          -- الحساب العام
    source               TEXT NOT NULL DEFAULT 'manual',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_analytic_line_date ON account_analytic_line(date);
CREATE INDEX idx_analytic_line_account ON account_analytic_line(account_id);
CREATE INDEX idx_analytic_line_move ON account_analytic_line(move_line_id);
-- G3: GIN index لبحث JSON (مطابق لـ _query_analytic_accounts في Odoo)
CREATE INDEX idx_analytic_line_dist_gin ON account_analytic_line
    USING gin(regexp_split_to_array(jsonb_path_query_array(
        analytic_distribution, '$.keyvalue()."key"')::text, '\D+'));

-- G7: قواعد التوزيع التلقائي
CREATE TABLE IF NOT EXISTS account_analytic_distribution_model (
    id                    BIGSERIAL PRIMARY KEY,
    sequence              INT DEFAULT 10,
    partner_id            BIGINT,
    partner_category_id   BIGINT,
    company_id            BIGINT,
    analytic_distribution JSONB NOT NULL,
    active                BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_analytic_dist_model_partner ON account_analytic_distribution_model(partner_id);
```

> [!TIP] **قرار متعدّد الخطط (G4)**: ابدأ بعمود `account_id` واحد (خطة المشاريع) + JSONB `analytic_distribution` للتوزيع. أضف `x_plan{id}_id` كأعمدة إضافية فقط عند الحاجة الفعلية لخطة ثانية (للحفاظ على المرونة دون تعقيد migration مبكّر). هذا يتوافق مع سلوك Odoo الديناميكي مع تقليل التعقيد في النسخة الأولى.

---

## 9. بيانات Seed (مثل `analytic_data.xml`)

في الـ migration (قسم `data`) أو دالة seed:

- دقة `Percentage Analytic` = 2 خانات.
- خطة "Project" الأساسية (`default_applicability=optional`).
- إعداد `analytic.project_plan` → id خطة المشاريع.
- خطة تجريبية "Departments" + "Internal" (اختياري) وحسابات تجريبية.

---

## 10. خطوات التنفيذ (Task Breakdown) — ~5 أيام

| اليوم | المهمة |
| ------- | -------- |
| **1** | تهيئة `internal/domain/analytic/` (كيانات + Applicability + توزيع + ports) + وحدات الاختبار للدوال النقية (`Merge`, `Validate100%`, `DistributionKey`). |
| **2** | الـ migration `000012` (جداول + فهارس + gin index + seed) وتحديث `migrations/fs.go`. |
| **3** | `postgres_repo.go` + `memory_repo.go` (تنفيذ كل دوال الـ Repository incl. `GetRelevantPlans`, `MatchDistribution`, `GetAccountTotals`). |
| **4** | `analytic_usecase.go` (توزيع/تحقق/ملاءمة/ربط MoveLine) + `analytic_usecase_test.go`. |
| **5** | HTTP handler + DTO + routes + integration tests + فحص `make lint` / `make test`. |

---

## 11. التحقق (Verification)

```bash
make test-race
make lint
make test                       # وحدات domain + usecase + adapter
```

**اختبارات محددة** (في `internal/domain/analytic/analytic_test.go` و `usecase`):

- `Merge` يحافظ على قِيم الخطط غير المتغيّرة عند دمج توزيع جديد (سلوك `_merge_distribution`).
- `Validate100%` يُفشل عندما يكون مجموع نسب خطة إلزامية ≠ 100%.
- `MatchDistribution` يختار القاعدة الأكثر تحديداً (شريك > فئة > عام) حسب `sequence`.
- `CreateLinesFromMoveLine` يفصل السطر بقيمة `1000` وتوزيع `{1:50, 2:50}` إلى سطرين كلٍّ بمبلغ 500.
- `GetRelevantPlans` يستبعد الخطط `unavailable` ويُدرج الخطط `mandatory`.

**تحقق يدوي**: `make run` + curl لإنشاء خطة ← حساب ← سطر تحليلي ← ربطه بقيد محاسبي ← قراءة الرصيد.

---

## 12. تحقّق: هل للفجوات مرحلة مستقلة في الخطة الأصلية؟

تم مسح كامل ملف `advanced__infrastructured_implementation_plan.md` (المراحل 10–25) بحثاً عن أي معالجة مستقلة لمفاهيم التحليل التي ذكرتُها (G1–G9). النتيجة: **لا توجد أي مرحلة مستقلة** لأيٍّ من هذه المفاهيم. الإشارات الوحيدة إليها في مراحل أخرى هي **تبعيات بسيطة فقط** وليست تطبيقات:

| المرحلة | الإشارة في الخطة الأصلية | النوع |
| --------- | -------------------------- | ------- |
| M12 (مخزون↔محاسبة) | `M3, M6, M10` في "يعتمد على" | تبعية فحسب (لا تطبيق) |
| M17 (مشاريع) | `AnalyticAccountID *int64` في `Project` | حقل مرجعي (يشير لحساب تُنشئه M10) |
| M19 (مصروفات) | `AnalyticAccountID *int64` في `ExpenseLine` | حقل مرجعي (يشير لحساب تُنشئه M10) |

### فحص كل فجوة على حدة

| الفجوة | المرحلة 10 | أي مرحلة أخرى (11–25) |
| -------- | :----------: | :----------------------: |
| G1 نوع المال `float64` | ✅ تُعالج | ⬜ **لا** — تقليد عام يؤثر على كل المراحل |
| G2 `AnalyticApplicability` | ✅ تُعالج | ⬜ **لا** |
| G3 `analytic_distribution` JSON | ✅ تُعالج | ⬜ **لا** |
| G4 عمود لكل خطة (multi-plan) | ✅ تُعالج | ⬜ **لا** |
| G5 حقول `AnalyticLine` | ✅ تُعالج | ⬜ **لا** |
| G6 debit/credit/balance | ✅ تُعالج | ⬜ **لا** |
| G7 `AnalyticDistributionModel` | ✅ تُعالج | ⬜ **لا** |
| G8 إعداد `project_plan` | ✅ تُعالج | ⬜ **لا** (م17 تستخدم النتيجة فقط) |
| G9 seed/décimal.precision | ✅ تُعالج | ⬜ **لا** |

### خلاصة

جميع الفجوات G2–G9 هي **آليات تحليلية** من جوهر المرحلة 10 ولا يملكها أي من المراحل 11–25 ككيان/واجهة/منطق مستقل. فقط `G1` (تقييس النوع إلى `float64` بدل `decimal.Decimal`) هي **مسألة عابرة للمراحل** (cross-cutting) — إذ إن الخطة الأصلية تستخدم `decimal.Decimal` في كل المراحل (11، 13، 16، 19، 20، 21، 23، 24). هذا قرار توحيد على مستوى المشروع يجب اتخاذه في M10 ويتكرر تطبيقه في كل مرحلة تالية، لكنه **ليس ميزةً ذات مرحلة مستقلة** بل تنظيم عام.

> الملاحظة: بما أن G2–G7 مكونات بنيوية لنظام التحليل نفسه، فإن تجاهلها في المرحلة 10 يعني إعادة بناء أساسي لاحقاً (لا سيّما للربط مع M12/M14/M17/M19). لذلك أُدرجت جميعها ضمن نطاق M10 بدلاً من تفريقها على مراحل أخرى.

---

## 13. ملاحظة للتكاملات المستقبلية (تؤثر على M12, M14, M17, M19)

- **M14/M19**: موديلات `sale_order_line`, `purchase_order_line`, `hr_expense` ستحتاج إلى `analytic_distribution` JSONB (ميراث `analytic.mixin`). صمّم جداول التحليل الآن لتكون قابلة للربط بـ `move_line_id` و `general_account_id`.
- **M17**: خطة "Project" `account_id` ستُرتبط لاحقاً بكيان `project`.
- **business_domain** قابل للتوسّع: أبقِه `TEXT` مع قيم canonical بدل `ENUM` لتسهيل الإضافة عبر المراحل.

---

## 14. المعايير المتفق عليها vs. الخطة الأصلية (مختصر الفجوات)

| البند | الخطة الأصلية | الخطة المُحدَّثة |
| ------- | --------------- | ------------------ |
| نوع المال | `decimal.Decimal` | `float64` (نمط المشروع) |
| Applicability | حقل داخل الخطة | **كيان مستقل** لكل شركة/مجال (G2) |
| التوزيع | جدول منفصل | **JSONB map** + GIN index + دمج (G3) |
| Multi-plan | `PlanID` واحد | **عمود لكل خطة** + `auto_account_id` (G4) |
| حقول السطر | ناقصة | + name, user_id, product_uom_id, currency, source (G5) |
| الأرصدة | `Balance` فقط | debit + credit + balance (بعملة الشركة) (G6) |
| توزيع تلقائي | — | **DistributionModel** جديد (G7) |
| خطة المشاريع | — | إعداد `project_plan` (G8) |
| Seed/precision | — | دقة 2 + خطة Project + أذونات (G9) |
| التقدير | 3 أيام | **5 أيام** |
