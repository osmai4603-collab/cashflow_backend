# خطة تنفيذ المرحلة 14: تكامل المبيعات والمشتريات مع المخزون

تستهدف هذه المرحلة الربط العملي بين دورة المبيعات/المشتريات والعمليات اللوجستية في المخزون، مما يضمن تدفق البيانات التلقائي من أمر البيع إلى التسليم، ومن أمر الشراء إلى الاستلام.

## مراجعة مشروع Odoo 19.0 والملاحظات الإضافية

بعد مراجعة `addons/sale_stock` و `addons/purchase_stock` في Odoo 19.0، تبين وجود بعض النقاط الهامة التي يجب تضمينها لضمان تكامل احترافي:

1.  **مجموعات التوريد (Procurement Groups):** ضرورة وجود `StockProcurementGroup` لربط جميع التحركات المخزنية (Stock Moves) الناتجة عن أمر بيع واحد أو أمر شراء واحد، مما يسهل تتبع الحالة الإجمالية.
2.  **مواعيد الالتزام (Commitment Dates):** حساب تاريخ التسليم المتوقع بناءً على "أيام التوريد" (Lead Times) المحددة في المنتج أو المورد.
3.  **المخزون المحجوز (Reserved Quantity):** عند تأكيد أمر البيع، يجب حجز الكمية في المخزون لضمان عدم بيعها لعميل آخر.
4.  **معالجة الطلبات المتأخرة (Backorders):** آلية للتعامل مع التسليم الجزئي وفتح "طلب متأخر" للكميات المتبقية.
5.  **قواعد المسارات (Routes & Rules):** تنفيذ محرك بسيط للقواعد (Pull/Push Rules) لتحديد ما إذا كان المنتج سيُسحب من المخزون أو سيُطلب من مورد (MTO - Make To Order).

## Proposed Changes

### [Component] Domain Layer Extensions

تحديث الكيانات الحالية لدعم الربط المخزني.

#### [MODIFY] [sale.go](file:///home/osm/StudioProjects/cashflow_backend/internal/domain/sale/sale.go)
- إضافة `ProcurementGroupID int64` لربط العمليات.
- إضافة `PickingIDs []int64` لتتبع عمليات التسليم.
- إضافة `DeliveryStatus string` (Draft, Waiting, Partially Available, Assigned, Done, Cancelled).
- في `SaleOrderLine`: إضافة `QtyDelivered`, `QtyInvoiced`, `ProductUomQty`.

#### [MODIFY] [purchase.go](file:///home/osm/StudioProjects/cashflow_backend/internal/domain/purchase/purchase.go)
- إضافة `PickingIDs []int64`.
- إضافة `ReceiptStatus string`.
- في `PurchaseOrderLine`: إضافة `QtyReceived`, `QtyInvoiced`.

#### [NEW] [procurement.go](file:///home/osm/StudioProjects/cashflow_backend/internal/domain/stock/procurement.go)
- تعريف `ProcurementGroup`: يجمع التحركات المرتبطة بمستند واحد (SO/PO).

---

### [Component] Usecase Layer (The Integration Logic)

هذا هو المحرك الأساسي للمرحلة.

#### [NEW] [sale_stock_usecase.go](file:///home/osm/StudioProjects/cashflow_backend/internal/usecase/sale/sale_stock_usecase.go)
- دالة `ConfirmSaleOrder`:
    1. إنشاء `ProcurementGroup`.
    2. لكل سطر في الطلب، تحديد "المسار" (Route).
    3. إذا كان المسار "Stock": إنشاء `StockPicking` من نوع `outgoing`.
    4. إنشاء `StockMove` لكل سطر وربطه بـ `SaleOrderLine`.
    5. محاولة حجز المخزون (Reserve).

#### [NEW] [purchase_stock_usecase.go](file:///home/osm/StudioProjects/cashflow_backend/internal/usecase/purchase/purchase_stock_usecase.go)
- دالة `ConfirmPurchaseOrder`:
    1. إنشاء `StockPicking` من نوع `incoming`.
    2. إنشاء `StockMove` لكل سطر وربطه بـ `PurchaseOrderLine`.

---

### [Component] Storage Layer

#### [MODIFY] [PostgreSQL Migrations](file:///home/osm/StudioProjects/cashflow_backend/migrations/)
- تحديث جداول `sale_orders`, `sale_order_lines`, `purchase_orders`, `purchase_order_lines` بالأعمدة الجديدة.
- إضافة علاقة (Foreign Key) في `stock_moves` تربطها بأسطر المبيعات والمشتريات.

## Verification Plan

### Automated Tests
- `TestSaleToDelivery`: تأكيد أمر بيع والتحقق من إنشاء `Picking` صحيح.
- `TestPurchaseToReceipt`: تأكيد أمر شراء والتحقق من إنشاء `Picking` استلام.
- `TestPartialDelivery`: تسليم جزء من البضاعة والتحقق من تحديث `QtyDelivered` في SO.
- `TestValuationIntegration`: التأكد من استدعاء `valuation_usecase` (من المرحلة 12) عند تنفيذ الـ `Picking`.

### Manual Verification
1. إنشاء Sale Order بمنتجين.
2. تأكيد الطلب -> الانتقال إلى صفحة المخزون لرؤية Delivery Order الجديد.
3. تنفيذ التسليم (Validate) -> العودة للـ SO والتأكد من تحول الحالة إلى "Delivered".
4. التحقق من القيد المحاسبي التلقائي (من المرحلة 12).
