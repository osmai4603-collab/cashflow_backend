# خطة تنفيذ المرحلة 13 — إعادة الطلب التلقائي + تكاليف الشحن الموزعة

> الهدف: أتمتة إدارة المخزون — إعادة طلب تلقائية عند انخفاض الكميات + توزيع تكاليف الشحن على حركات الاستلام.
> مصدر المرجع: `addons/stock/models/stock_orderpoint.py` و`addons/stock_landed_costs/` في Odoo 19.0.
> يعتمد على: المرحلة 6 (المخزون)، المرحلة 12 (تكامل المخزون↔المحاسبة — مفترض مكتملة).

---

## 1) فجوات الخطة الأصلية المستخلصة من مراجعة Odoo 19.0

### إعادة الطلب (Orderpoint)
1. **`qty_forecast` بدل المخزون الحالي**: Odoo يقارن `qty_forecast = on_hand + وارد − منصرف` خلال أفق التوريد `lead_horizon = today + lead_days + horizon_days`، وليس `qty_on_hand` فقط.
2. **معادلة `qty_to_order`**: `max(MinQty, MaxQty) − qty_forecast` ثم تُقرّب للأعلى لمضاعف الطلب.
3. **مصدر التموين (Route)**: `buy` / `pull` / `manufacture` — ننفذ `buy` الآن مع ترك النوع قابلاً للتمديد.
4. **مورد PO التلقائي**: حقل `VendorID` صريح على الـ Orderpoint (قرار المستخدم).
5. **قيود تحقق**: `Min ≤ Max`، `UNIQUE(product_id, location_id, company_id)`، رفض snooze للـ auto-trigger.
6. **دمج procurements**: تجميع طلبات نفس المنتج/المورد في PO وخط واحد وتحديث `origin`.

### تكاليف الشحن (Landed Cost)
1. **ربط بفاتورة المورد**: `vendor_bill_id` على الـ LandedCost، وعلامة `is_landed_costs_line` على سطور الفاتورة، وحقلا `landed_cost_ok` + `split_method_landed_cost` على المنتج.
2. **أعمدة التعديل**: `ValuationAdjustment` يحتاج `quantity`, `weight`, `volume` جدًا للتوزيع.
3. **معادلات التوزيع الخمس + rounding HALF-UP** بتحقيق العملة وإضافة فرق التقريب لآخر سطر (وإلا يفشل `_check_sum`).
4. **صيغة القيد المحاسبي**: مدين حساب تقييم المخزون / دائن حساب مصروف سطر التكلفة، `diff = additional × (remaining_qty / quantity)`، عكس الإشارة للقيم السالبة، لا قيد عند `remaining_qty == 0`.
5. **تحديث التقييم بعد التأكيد** (`move._set_value()`) عبر محرك المرحلة 12.
6. **فلتر التصفية**: منتجات `cost_method ∈ {fifo, average}` فقط بحركات غير ملغاة.
7. **دفتر يومية الشحن** `lc_journal_id` على مستوى الشركة و`account_id` على سطر التكلفة.

## 2) القرارات المعتمدة

| القرار | الاختيار |
|---|---|
| مورد PO التلقائي | حقل `VendorID` صريح على الـ Orderpoint |
| المرحلة 12 | تُعامل كتسليم مكتمل (محرك التقييم موصول بـ `ValidatePicking`) |
| طرق التوزيع | الخمس طرق كاملة (equal, by_quantity, by_current_cost_price, by_weight, by_volume) |

## 3) Domain

### `internal/domain/stock/orderpoint.go` (جديد)
```go
type OrderpointSource string // "buy" فقط الآن (لاحقًا: pull, manufacture)
type OrderpointTrigger string // "auto" | "manual"

type Orderpoint struct {
    ID              int64
    Name            string              // "ROP/2026/00001"
    ProductID       int64
    WarehouseID     int64
    LocationID      int64
    VendorID        *int64              // مورد PO التلقائي
    MinQty          float64
    MaxQty          float64
    QtyMultiple     float64             // 0 = بدون تقريب
    LeadDays        int
    Source          OrderpointSource
    Trigger         OrderpointTrigger
    SnoozedUntil    *time.Time          // manual فقط
    QtyOnHand       float64             // computed
    QtyForecast     float64             // computed
    QtyToOrder      float64             // computed (+ manual override)
    QtyToOrderManual float64
    DeadlineDate    *time.Time          // computed
    Active          bool
    CompanyID       int64
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### `internal/domain/stock/landed_cost.go` (جديد)
```go
type LandedCostState string // "draft" | "done" | "cancel"
type SplitMethod string     // equal | by_quantity | by_current_cost_price | by_weight | by_volume

type LandedCost struct {
    ID                  int64
    Name                string              // "LC/2026/00001"
    Date                time.Time
    State               LandedCostState
    PickingIDs          []int64
    CostLines           []LandedCostLine
    ValuationAdjustments []ValuationAdjustment
    Description         string
    AmountTotal         float64             // computed = Σ cost_lines.price_unit
    AccountMoveID       *int64
    JournalID           int64               // lc_journal_id
    VendorBillID        *int64              // فاتورة المورد المرتبطة
    CompanyID           int64
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

type LandedCostLine struct {
    ID            int64
    LandedCostID  int64
    Name          string
    ProductID     int64                     // منتج التكلفة (شحن/جمارك/تأمين)
    AccountID     int64                     // حساب المصروف
    PriceUnit     float64
    SplitMethod   SplitMethod
}

type ValuationAdjustment struct {
    ID                 int64
    LandedCostID       int64
    CostLineID         int64
    MoveID             int64
    ProductID          int64
    Quantity           float64              // بالوحدة الأساسية للمنتج
    Weight             float64              // product.weight × quantity
    Volume             float64              // product.volume × quantity
    FormerCost         float64              // قيمة الحركة الأصلية
    AdditionalCost     float64              // التكلفة الموزعة
    FinalCost          float64              // Former + Additional
    MoveRemainingQty   float64              // لصيغة القيد المحاسبي
}
```

### تعديلات على كيانات قائمة
- `product.ProductTemplate`: + `LandedCostOK bool`, `DefaultSplitMethod stock.SplitMethod`
- `accounting.AccountMoveLine`: + `IsLandedCostsLine bool`
- `company`: + `LandedCostJournalID *int64` (عمود `lc_journal_id`)
- `stock.Repository` (ports.go): دوال CRUD للـ Orderpoint وLandedCost وNextSequence("orderpoint")

## 4) المعادلات المرجعية (من Odoo)

### توزيع التكلفة لكل تعديل × سطر تكلفة
| split_method | value |
|---|---|
| by_quantity | `price_unit / total_qty × quantity` |
| by_weight | `price_unit / total_weight × weight` |
| by_volume | `price_unit / total_volume × volume` |
| equal | `price_unit / total_line` |
| by_current_cost_price | `price_unit / total_cost × former_cost` |

`round HALF-UP` بقيمة تحقيق العملة لكل قيمة؛ `rounding_diff = price_unit − Σ values` تضاف لآخر تعديل.

### القيد المحاسبي (لكل تعديل ذي `move_id` ومنتج real_time)
```
diff = additional_landed_cost × (remaining_qty / quantity)   # تجاهل إذا remaining_qty == 0
مدين: حساب تقييم المخزون   دائن: cost_line.account_id (أو حساب مصروف منتج الشحن)
# قيمة سالبة → عكس الخصم/الدائن (قيد معكوس)
```
بعد إنشاء القيد وترحيله: `move._set_value()` (تحديث تقييم الحركة وضبط تكلفة المنتج عبر محرك المرحلة 12).

## 5) طبقة الـ Usecase

### `internal/usecase/stock/orderpoint_usecase.go` (جديد)
- CRUD مع تحقق: `Min ≤ Max`، `UNIQUE(product, location, company)`، رفض snooze مع auto.
- `ComputeQty(ctx, id)`: `qty_forecast = on_hand + Σ(incoming خلال الأفق) − Σ(outgoing خلال الأفق)`.
- `RunReorderRules(ctx)`: لكل orderpoint نشط (auto، غير snoozed) حيث `QtyToOrder > 0` → تقريب لأعلى للمضاعف → PO draft عبر `purchaseCreateOrder` مع دمج: PO مفتوح بنفس المورد/الشركة؛ سطر موجود لنفس المنتج تُضاف الكمية وإلا يُنشأ سطر؛ `origin = orderpoint.Name`.
- `Suggestions(ctx)`: القواعد بكمياتها المحسوبة للعرض.

### `internal/usecase/stock/landed_cost_usecase.go` (جديد)
- يتسلم `stock.Repository` + `AccountingService` (نمط `purchaseusecase.New`).
- `Compute(ctx, id)`: مسح التعديلات السابقة → الفلترة → الحساب → التوزيع → rounding.
- `Validate(ctx, id)`: تحقق (draft + pickings + `_check_sum`) → القيد المحاسبي → ترحيل → تحديث التقييم → `state = done`.
- `Cancel(ctx, id)`: منع غير المعلّق.

## 6) Migration رقم 000019

جداول: `stock_orderpoints` (مع UNIQUE المركب)، `stock_landed_costs`، `stock_landed_cost_lines`، `stock_valuation_adjustment_lines`.
تعديلات: `products += (landed_cost_ok, split_method_landed_cost)`, `companies += lc_journal_id`, `account_move_lines += is_landed_costs_line`.

## 7) HTTP

```
CRUD  /api/v1/reorder-rules             (+ POST /run, GET /suggestions)
CRUD  /api/v1/landed-costs              (+ POST /{id}/compute, /validate, /cancel)
POST  /api/v1/invoices/{id}/create-landed-cost
```
ملفات: `internal/adapters/http/stock/{orderpoint_handler.go, landed_cost_handler.go, dto.go}` + تحديث `routes.go` و`NewRouter` و`cmd/server/main.go`.

## 8) الـ Background Worker

`reorder-checker` في `main.go` عبر `wm.Start` مع `time.NewTicker(cfg.Stock.ReorderInterval)`، وإضافة `StockSettings{ReorderEnabled, ReorderInterval}` في `internal/platform/config`.

## 9) التحقق

- وحدة: صيغ التوزيع الخمس + rounding parity، `check_sum`، حساب `qty_to_order`، قيود Orderpoint.
- تكامل: Orderpoint→Run→PO؛ Picking→Compute→Validate→قيد متوازن→قيمة الحركة ترتفع؛ فاتورة مورد فيها صنف `landed_cost_ok`.
- أوامر: `make lint`, `make test`, `make test-race`.

## 10) التسلسل والجهد

| # | الخطوة | الجهد |
|---|---|---|
| 1 | Migration + Storage | يوم |
| 2 | Domain + Ports | نصف يوم |
| 3 | Orderpoint usecase + worker | يوم ونصف |
| 4 | Landed cost usecase | يومان |
| 5 | HTTP + توصيل | يوم |
| 6 | تدفق فاتورة المورد | نصف يوم |
| 7 | اختبارات + إصلاحات | يوم |

**الإجمالي: ~7 أيام عمل.**