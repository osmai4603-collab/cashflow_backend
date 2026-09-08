# خطة تنفيذ المرحلة 21: التسليم والشحن (Delivery Carriers)

تستهدف هذه المرحلة بناء نظام إدارة شركات الشحن وحساب تكاليف التوصيل تلقائياً، بما يتماشى بشكل صارم مع سلوك Odoo 19.0 (موديول `delivery`).

## الأهداف
- إدارة طرق التوصيل (Delivery Methods/Carriers).
- دعم التسعير الثابت (Fixed Price) والتسعير القائم على القواعد (Based on Rules).
- حساب تكاليف الشحن بناءً على الوزن، الحجم، السعر، أو الكمية.
- تكامل الشحن مع أوامر البيع (Sale Orders) وعمليات المخزون (Stock Pickings).
- دعم الشحن المجاني عند تجاوز مبلغ محدد.

## الكيانات البرمجية (Domain Entities)

### [NEW] `internal/domain/delivery/carrier.go`

```go
type CarrierType string
const (
    CarrierTypeFixed      CarrierType = "fixed"
    CarrierTypeBaseOnRule CarrierType = "base_on_rule"
)

type DeliveryCarrier struct {
    ID              int64
    Name            string
    Active          bool
    Sequence        int
    DeliveryType    CarrierType     // fixed, base_on_rule
    ProductID       int64            // المنتج المرتبط في سطر الفاتورة/الطلب
    FixedPrice      float64
    Margin          float64          // نسبة مئوية تضاف للتكلفة
    FixedMargin     float64          // مبلغ ثابت يضاف للتكلفة
    FreeOver        bool             // شحن مجاني إذا تجاوز الطلب مبلغاً معيناً
    Amount          float64          // المبلغ المطلوب للشحن المجاني
    
    // فلترة جغرافية
    CountryIDs      []int64
    StateIDs        []int64
    ZipPrefixIDs    []int64

    // قيود
    MaxWeight       float64
    MaxVolume       float64

    Rules           []DeliveryPriceRule
    CompanyID       *int64
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type DeliveryPriceRule struct {
    ID             int64
    CarrierID      int64
    Variable       string // weight, volume, wv, price, quantity
    Operator       string // ==, <=, <, >=, >
    MaxValue       float64
    ListBasePrice  float64
    ListPrice      float64
    VariableFactor string // العامل المتغير للحساب
}

type DeliveryZipPrefix struct {
    ID        int64
    Name      string
}
```

## التعديلات على الكيانات الموجودة

### [MODIFY] [sale.order](file:///home/osm/StudioProjects/cashflow_backend/internal/domain/sale/order.go)
- إضافة `CarrierID *int64`
- إضافة `ShippingWeight float64`
- إضافة `DeliveryRatingSuccess bool`

### [MODIFY] [stock.picking](file:///home/osm/StudioProjects/cashflow_backend/internal/domain/stock/picking.go)
- إضافة `CarrierID *int64`
- إضافة `CarrierTrackingRef string`
- إضافة `CarrierTrackingURL string`
- إضافة `Weight float64`
- إضافة `ShippingWeight float64`
- إضافة `NumberOfPackages int`

## الطبقات البرمجية (Infrastructure & Usecases)

### [NEW] `internal/usecase/delivery/carrier_usecase.go`
- `CalculateRate(ctx, orderID)`: حساب التكلفة بناءً على القواعد أو السعر الثابت.
- `Match(carrier, order)`: التحقق من توافق الشاحن مع عنوان العميل والقيود (وزن/حجم).
- `AddShippingToOrder(ctx, orderID, carrierID)`: إضافة سطر شحن لأمر البيع.

### [NEW] `internal/adapters/storage/delivery/postgres_repo.go`
- تنفيذ مستودع البيانات لـ `DeliveryCarrier` و `Rules`.

### [NEW] `internal/adapters/http/delivery/handler.go`
- `GET /api/v1/delivery-carriers`
- `POST /api/v1/delivery-carriers/{id}/rate`
- `POST /api/v1/sale-orders/{id}/add-shipping`

## منطق الحساب (Rules Engine)
سيتم تنفيذ محرك قواعد بسيط يدعم المتغيرات:
- `weight`: مجموع أوزان المنتجات.
- `volume`: مجموع أحجام المنتجات.
- `price`: إجمالي الطلب قبل الشحن.
- `quantity`: مجموع كميات المنتجات.

معادلة السعر: `Price = ListBasePrice + (ListPrice * VariableFactorValue)`

## خطة العمل التنفيذية

1.  **قاعدة البيانات**: إنشاء جداول `delivery_carriers`, `delivery_price_rules`, `delivery_zip_prefixes`. [ ]
2.  **Domain**: تعريف الكيانات والواجهات في `internal/domain/delivery`. [ ]
3.  **Storage**: تنفيذ مستودع PostgreSQL. [ ]
4.  **Usecase**: تنفيذ منطق المطابقة وحساب الأسعار. [ ]
5.  **Integration**: تعديل `SaleOrder` و `StockPicking` لدعم بيانات الشحن. [ ]
6.  **API**: توفير منافذ التحكم والاختبار. [ ]
7.  **Testing**: اختبار سيناريوهات (شحن مجاني، شحن بالقواعد، شحن ثابت). [ ]

## معايير القبول
- ظهور خيارات الشحن المتاحة بناءً على عنوان العميل فقط.
- حساب تكلفة الشحن بدقة بناءً على وزن المنتجات في سلة التسوق.
- إضافة سطر منتج الشحن تلقائياً لأمر البيع عند اختيار طريقة التوصيل.
- انتقال معلومات الشاحن من أمر البيع إلى عملية المخزون (Picking) عند التأكيد.
