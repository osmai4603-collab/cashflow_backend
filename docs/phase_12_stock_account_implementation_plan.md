# المرحلة 12 — تكامل المخزون ↔ المحاسبة (Stock-Account Integration): الخطة التنفيذية

**المرجع**: Odoo 19.0 `addons/stock_account/models/` (مراجعة مُعمّقة)
**يعتمد على**: المرحلة 3 (المحاسبة), المرحلة 6 (المخزون), المرحلة 10 (التحليلية)
**التقدير المُحدَّث**: **~8-11 أيام** (بدلاً من 4) — بسبب إدراج كامل نطاق Odoo (standard/fifo/average + real_time/periodic + COGS/فرق السعر)
**الأولوية**: 🔴 حرج

---

## 1. خلاصة مراجعة مصدر Odoo 19.0

تمت مراجعة كامل موديول `stock_account` في Odoo 19.0. **النتيجة الأهم**: النموذج القائم في الخطة الأصلية قديم — فلا يوجد `stock.valuation.layer` في 19.0؛ أُعيدت تسميته إلى **`product.value`** وأُدمجت قيمة التقييم داخل `stock.move.value`.

الملفات الفعلية:

| ملف Odoo | الغرض |
| ---------- | ------- |
| `stock_move.py` | **جوهري**: `_action_done` override، `_set_value`، `_get_value_data`، `_create_account_move`، `_get_account_move_line_vals`، `_run_fifo` |
| `product_value.py` | **المعامِل المكافئ لـ SVL المُعاد تسميته**: سجل التعديلات `product.value` |
| `product.py` | التقييم على مستوى المنتج/الفئة: `cost_method`، `valuation`، `lot_valuated`، `_run_fifo/_run_average_batch/_run_standard_batch` |
| `res_company.py` | إعدادات الشركة: `account_stock_journal_id`، `account_stock_valuation_id`، `cost_method`، `inventory_valuation`، إغلاق الرصيد الدوري |
| `stock_location.py` | `valuation_account_id` (حدود التقييم) |
| `account_move.py` / `account_move_line.py` | سطور COGS على فواتير العملاء، تجاوز حساب التوريد، `cogs_origin_id` |
| `account_chart_template.py` | إنشاء يومية `inventory_valuation` (STJ) وتوصيل الحسابات |
| `account_account.py` | `account_stock_variation_id`، `account_stock_expense_id` |
| `purchase_stock/` | `_get_value_from_account_move` و `_prepare_anglo_saxon_in_lines_vals` (فرق السعر) |

### دورة التقييم في Odoo 19.0 (مرجع التنفيذ)

```
picking._action_done
  → move._action_done (stock_account override)
      → moves_out._set_value()            # قيّم الخارج أولاً (يحتاج كومة FIFO الحالية)
      → super()._action_done()            # المؤكد الأساسي يغيّر الحالة
      → moves_in._set_value()             # ثم الداخل
      → moves._create_account_move()      # نشر القيد في يومية المخزون
      → _update_standard_price()          # لتحديث avg_cost / std price
      → _create_analytic_move()           # أسطر تحليلية (M10)
```

### آليات القيم حسب طريقة التكلفة

| cost_method | الداخلة | الخارجة |
|-------------|---------|---------|
| `standard` | `standard_price × qty` | `standard_price × qty` |
| `fifo` | السعر من المصدر (فاتورة/PO) | `_run_fifo` يستهلك أقدم دخول |
| `average` | السعر من المصدر | `standard_price × qty` ثم `_update_standard_price` |

### بنية القيد المحاسبي `_get_account_move_line_vals` (سطران لكل حركة)

```
إذا location_id.valuation_account_id (خرج إلى موقع مُقيَّم):
    debit  = product stock_valuation
    credit = location_id.valuation_account_id
وإلا:
    debit  = location_dest_id.valuation_account_id  (قد يكون False للنقل الداخلي)
    credit = product stock_valuation
```

`_should_create_account_move`: يُنشئ قيدًا فقط عندما `product.is_storable && is_valued && (valuation_account على src أو dest) && qty != 0 && valuation == real_time`.

---

## 2. الفجوات الحرجة بين الخطة الأصلية وبين Odoo (يجب معالجتها)

### G1 — لا يوجد `StockValuationLayer`؛ النموذج مدمج في `stock.move` + `product.value`

الخطة الأصلية تقترح كيان `StockValuation` منفصل كجدول طبقات. في Odoo 19.0:
- القيمة محفوظة داخل `stock.move` (`value`, `value_manual`, `standard_price`, `is_in`, `is_out`, `is_dropship`, `remaining_qty`, `remaining_value`, `account_move_id`).
- `product.value` جدول **سجل التعديلات** فقط (تعديل std price / تعديل قيمة حركة / تعديل سعر lot).

**القرار**: دمج القيمة داخل `stock.move` + جدول `product_values` للسجل (حسب اختيار المستخدم "دمج داخل stock.move + جدول سجل").

### G2 — `cost_method` و `valuation` غير موجودة إطلاقاً في المشروع

لا حقل `cost_method` (standard/fifo/average) ولا `valuation` (real_time/periodic) ولا `lot_valuated` على المنتجات/الفئات/الشركة. يجب إضافتها مع آلية التوريث: `categ.property_*` أو fallback `company.*`.

### G3 — `stock.move` لا يحمل أي حقل تكلفة

`StockMove` حالياً يحمل فقط `ProductQty`/`QuantityDone`. لا `value` ولا `standard_price` ولا `account_move_id`. يجب إضافتها.

### G4 — حسابات/يومية المخزون تفقد

المُهرّئ عند `000004` يزرع Inventory (id=5, `140000`) و COGS (id=12, `500000`)، لكن:
- لا حساب **Stock Variation** (نظير الإغلاق الدوري)
- لا حساب **Price Difference** (فرق سعر التوريد)
- لا **يومية مخزون** `STJ` (عاملة)
- `account.journal` لا يملك نوعًا مخصصًا للمخزون

### G5 — `StockLocation` بلا `valuation_account_id`

لا حدود تقييم على المواقع (internal/transit). يجب إضافتها لتحديد متى يُنشأ القيد.

### G6 — `AccountMoveLine` بلا `display_type`/`cogs_origin_id`

لا دعم لسطور COGS (بـ `display_type='cogs'`) ولا لتمييز سطور فرق السعر.

### G7 — لا مفهوم للإغلاق الدوري `accounting_period`

لنظام `periodic` نحتاج كيان فترة محاسبية يجمّع فرق `stock_value − accounting_value` ويرحّله لحساب Stock Variation عند الإغلاق.

### G8 — `stock_usecase` لا يستقبل اعتماديات المحاسبة

`UseCase.New` (stock_usecase.go:119) لا يحقن `accounting.Repository`/`accountingusecase`. يجب مدّه بنمط interface `AccountingService` (كما في `sale_usecase.go:21-25`) وتحديث `main.go:265`.

### G9 — استخدام `float64` (وليس `decimal.Decimal`)

مثل بقية المشروع: كل المبالغ `float64` مع تقريب 4 خانات. (الخطة الأصلية استخدمت `decimal.Decimal` — لا نتبع ذلك للحفاظ على التجانس).

---

## 3. نطاق التنفيذ (حسب اختيار المستخدم)

1. طرق التكلفة: **standard / fifo / average** (كاملة كما في Odoo 19.0)
2. أنظمة التقييم: **real_time** (قيود لحظية عند `Validate`) + **periodic** (قيود عند الإغلاق الدوري)
3. القيمة مدمجة في `stock.move` + جدول سجل `product_values`
4. قيود COGS على فواتير العملاء + قيود فرق السعر على فواتير الموردين (Anglo-Saxon)
5. إعدادات تقييم على مستوى: الشركة / الفئة / المنتج / الموقع

---

## 4. الكيانات المُحدَّثة (Go — متوافقة مع معايير المشروع)

### 4.1 `internal/domain/stock/valuation.go` (جديد)

```go
type CostMethod string
const (
    CostStandard CostMethod = "standard"
    CostFIFO     CostMethod = "fifo"
    CostAverage  CostMethod = "average"
)

type ValuationMode string
const (
    ValuationRealTime ValuationMode = "real_time"
    ValuationPeriodic ValuationMode = "periodic"
)

// FIFOEntry يمثل شريحة كومة التقييم (snapshot لمكدس FIFO)
type FIFOEntry struct {
    MoveID int64
    Qty    float64
    Value  float64
}

// ProductValue — سجل التعديلات (مكافئ product.value)
type ProductValue struct {
    ID          int64
    ProductID   int64
    MoveID      *int64
    LotID       *int64
    Value       float64
    Date        time.Time
    UserID      int64
    Description string
    CompanyID   int64
    CreatedAt   time.Time
}

// AccountingPeriod — للإغلاق الدوري
type AccountingPeriod struct {
    ID            int64
    Name          string
    DateFrom      time.Time
    DateTo        time.Time
    State         string // open, closed
    AccountMoveID *int64
    JournalID     int64
    CompanyID     int64
    CreatedAt     time.Time
}

// خيارات تقييم منتج (تُقرأ من المنتج/الفئة/الشركة)
type ProductValuationConfig struct {
    CostMethod               CostMethod
    Valuation                ValuationMode
    StockValuationAccountID  *int64
    PriceDifferenceAccountID *int64
    StockJournalID           *int64
}
```

### 4.2 `internal/domain/stock/move.go` (إضافة حقول)

```go
type StockMove struct {
    // ...الحقول الحالية...
    Value          float64      // قيمة التقييم (0 إن غير مُقيَّم)
    ValueManual    *float64
    StandardPrice  float64      // التكلفة لحظة التنفيذ
    IsIn           bool         // مُقيَّمة داخلة
    IsOut          bool         // مُقيَّمة خارجة
    IsDropship     bool
    RemainingQty   float64      // موضع FIFO المتبقي
    RemainingValue float64
    AccountMoveID  *int64       // القيد المرتبط
}
```

### 4.3 `internal/domain/product/product.go` (إضافة)

```go
// ProductTemplate إضافات
CostMethod    CostMethod    // من الفئة/الشركة
Valuation     Valuation     // real_time / periodic
LotValuated   bool
AvgCost       float64
TotalValue    float64
StockValuationAccountID  *int64
PriceDifferenceAccountID *int64
StockJournalID           *int64

// ProductCategory إضافات (إعدادات قابلة للتوريث)
PropertyCostMethod               *CostMethod
PropertyValuation                *Valuation
PropertyLotValuated              *bool
PropertyStockValuationAccountID  *int64
PropertyPriceDifferenceAccountID *int64
PropertyStockJournalID           *int64
```

### 4.4 `internal/domain/stock/location.go` (إضافة)

```go
ValuationAccountID *int64 // حدود التقييم
```

### 4.5 `internal/domain/accounting` (إضافة)

```go
// move.go — AccountMoveLine
DisplayType  string // "", "cogs"
CogsOriginID *int64

// journal.go — JournalType (إضافة، اختيارية)
JournalTypeStock   JournalType = "stock"
```

---

## 5. محرك التقييم الخالص (الجوهر)

**ملف `internal/domain/stock/valuation.go`** (دوال خالصة قابلة للاختبار):

| دالة | مكافئ Odoo | الوصف |
|------|------------|-------|
| `ResolveValuationConfig(product, category, company)` | `get_product_accounts` | تحليل cost_method/valuation الحسابات من الفئة/الشركة |
| `ComputeInValue(move, stdPrice)` | `_get_value_data` | أولوية: manual → product_value → فاتورة مورد → PO → إرجاع → std×qty |
| `ComputeOutValue(move)` | `_set_value` | lot: Σ(lot.price×qty) / fifo: RunFIFO / avg/std: std×qty |
| `RunFIFO(stack, qty)` | `_run_fifo` | استهلاك أقدم دخول + `fifo_qty_already_processed` |
| `RunAverageBatch(moves)` | `_run_average_batch` | avg_cost عبر in/out المرتّبة |
| `ShouldCreateAccountMove(move)` | `_should_create_account_move` | بوابة إنشاء القيد |
| `BuildAccountMoveLines(move)` | `_get_account_move_line_vals` | سطران مدين/دائن |

---

## 6. واجهة التخزين (Ports) — إضافات على `stock.Repository`

```go
// valuations
UpdateMoveValue(ctx, moveID int64, value float64) error
CreateProductValue(ctx, pv *ProductValue) error
ListProductValues(ctx, f *filter.Filter, page) (pagination.PageResult[ProductValue], error)
GetFIFOStack(ctx, productID, companyID int64) ([]FIFOEntry, error)
ComputeTotalValuation(ctx, productID *int64, locationID *int64) ([]ValuationSummary, error)

// periods
CreateAccountingPeriod(ctx, p *AccountingPeriod) error
GetAccountingPeriodByID(ctx, id int64) (*AccountingPeriod, error)
ListAccountingPeriods(ctx, f *filter.Filter, page) (pagination.PageResult[AccountingPeriod], error)
CloseAccountingPeriod(ctx, p *AccountingPeriod) error
```

---

## 7. منطق الاستخدام (Usecase) — `internal/usecase/stock/valuation_usecase.go`

- `ValuatePicking(ctx, picking)` — ينسّق: يقيّم OUT ثم IN، ينشئ القيد (`CreateJournalEntry` + `PostMove`)، يحدّث standard price، ينشئ `product_value`.
- `ClosePeriodValuation(ctx, periodID)` — يحسب فرق القيمة ويرحّله لحساب Stock Variation.
- `AdjustValuation(ctx, moveID, newValue)` — مكافئ `value_manual` → `product_value` ثم إعادة `_set_value`.

### الخطاف في `ValidatePicking` (stock_usecase.go:677)

بعد نجاح `ValidatePickingTx` وقبل تحديث sale/purchase:
- إن `real_time` → `uc.valuation.ValuatePicking(ctx, picking)`
- إن `periodic` → سجّل القيمة فقط (بلا قيد)

### حقن التبعية

```go
type AccountingService interface {
    CreateJournalEntry(ctx, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error)
    PostMove(ctx, id int64) (*accounting.AccountMove, error)
}
func New(repo, partnerRepo, productRepo, saleRepo, purchaseRepo,
    accountingRepo accounting.Repository, accountingSvc AccountingService, logger) *UseCase
```
(تحديث `cmd/server/main.go:265`.)

---

## 8. نِقاط API

```
GET    /api/v1/stock/valuations                  تقييم المخزون (فلترة)
GET    /api/v1/stock/valuations/by-product/{id}
GET    /api/v1/stock/valuations/by-location/{id}
GET    /api/v1/products/{id}/stock-value
GET    /api/v1/stock/moves/{id}/value
PUT    /api/v1/stock/moves/{id}/value            تعديل يدوي
GET    /api/v1/stock/value-history               سجل product_value
POST   /api/v1/stock/periods
POST   /api/v1/stock/periods/{id}/close          إغلاق دوري
GET    /api/v1/stock/value-report                تقرير تقييم (مفصّل في M25)
```

تسجيل عبر `internal/adapters/http/stock/valuation_handler.go` + `dto.go` + سطور في `routes.go` و `router.go` مع نماذج ACL جديدة (`stock.valuation`).

---

## 9. مخطط قاعدة البيانات (تعديلات Migration `000018`)

1. `ALTER TABLE stock_moves` → +`value`, `value_manual`, `standard_price`, `is_in`, `is_out`, `is_dropship`, `remaining_qty`, `remaining_value`, `account_move_id`
2. `ALTER TABLE product_templates` → +`cost_method`, `valuation`, `lot_valuated`, `avg_cost`, `total_value`, حقول الحسابات/اليومية
3. `ALTER TABLE product_categories` → +`property_cost_method`, `property_valuation`, `property_lot_valuated`, `property_*_account_id`, `property_stock_journal_id`
4. `ALTER TABLE stock_locations` → +`valuation_account_id`
5. `ALTER TABLE account_accounts` → +`account_stock_variation_id`, `account_stock_expense_id`
6. `CREATE TABLE product_values` (سجل التعديلات)
7. `CREATE TABLE accounting_periods`
8. Seed: حساب Stock Variation، حساب Price Difference، يومية `STJ`، ربط الحسابات الافتراضية، ACL

---

## 10. خطوات التنفيذ (Task Breakdown) — ~8-11 أيام

| يوم | الحزمة | التسليم |
|-----|--------|---------|
| 1-1.5 | migrations 000018 + data model | schema + seed |
| 1.5-2 | Domain entities | move/valuation/product/category/location/line |
| 3-4.5 | محرك التقييم الخالص | valuation.go + unit tests |
| 5-5.5 | Repositories | postgres + memory + ports |
| 6-6.5 | Valuation usecase + hook + main.go | تكامل ValidatePicking |
| 7-7.5 | API endpoints + routes + ACL | endpoints قابلة للتجربة |
| 8-9 | COGS / فرق السعر على الفواتير | سطور على فواتير العملاء/الموردين |
| 9-10 | إغلاق دوري + سجل product_value + reports | periods + history |
| 10-11 | اختبارات E2E + lint + build | تحقق نهائي |

---

## 11. التحقق (Verification)

```bash
make test && make test-race && make lint && make build
```

### Unit tests
- `RunFIFO`: استهلاك الكومة، سلبية، التمديد بآخر سعر
- `RunAverageBatch`: avg_cost عبر in/out
- `BuildAccountMoveLines`: مدين/دائن لكل نوع (incoming/outgoing/internal/dropship)

### Integration (E2E via curl)
1. استلام بضاعة → قيد متوازن
2. تسليم بضاعة → قيد COGS/Inventory
3. نقل داخلي → قيد متوازن على نفس الحساب
4. FIFO بأسعار متعددة → قيمة خروج صحيحة
5. إغلاق فترة periodic → فرق Stock Variation

---

## 12. تحقّق: هل للفجوات مرحلة مستقلة في الخطة الأصلية؟

| فجوة | الأولوية | ملاحظة |
|------|----------|--------|
| G1 (الدمج في move) | 🔴 نقدية | خروج عن المستند لكن مطابق Odoo 19.0 |
| G2 (cost_method/valuation) | 🔴 نقدية | لا توجد مرحلة مستقلة — ضمن M12 |
| G3 (حقل تكلفة في move) | 🔴 نقدية | ضمن M12 |
| G4 (حسابات/يومية مخزون) | 🔴 نقدية | ضمن M12 |
| G5 (valuation_account للموقع) | 🔴 نقدية | ضمن M12 |
| G6 (display_type/cogs_origin) | 🔴 نقدية | ضمن M12 |
| G7 (فترة دورية) | 🟡 مهمة | ضمن M12 (periodic) |
| G8 (حقن accounting في stock) | 🔴 نقدية | يشترط M10 جاهزة (تحليلية) |
| G9 (float64) | 🔴 نقدية | تجانس مع بقية المشروع |

---

## 13. المعايير المتفق عليها vs. الخطة الأصلية (مختصر)

1. ❌ لا كيان `StockValuation` منفصل → ✅ مدمج في `stock.move` + `product_values`
2. ❌ لا cost_method/valuation → ✅ `standard`/`fifo`/`average` + `real_time`/`periodic`/`lot_valuated`
3. ❌ `decimal.Decimal` → ✅ `float64` (rounding 4)
4. 🔶 القيد في `ValidatePicking` بدلاً من `StockPicking.Validate()` (نفس السلوك، خطاف أنظف)
5. 🔶 إضافة حسابات Stock Variation/Price Difference + يومية STJ (ليست في المستند)
6. 🔶 إضافة `accounting_periods` للإغلاق الدوري (ليست في المستند)
