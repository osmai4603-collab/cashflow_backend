# خطة تنفيذية تفصيلية لإكمال ميزات cashflow_backend المتوافقة مع Odoo 19

تعتمد هذه الخطة على تحليل الفجوات الوارد في [odoo19_partial_features_completion_analysis.md](file:///home/osm/StudioProjects/cashflow/cashflow_backend/docs/odoo19_partial_features_completion_analysis.md). الهدف هو الانتقال من التنفيذ الجزئي إلى التكافؤ الصارم مع سلوك Odoo 19 في المجالات الحيوية.

## User Review Required

> [!IMPORTANT]
> **تغيير مصدر الحقيقة (Source of Truth):** سيتم تعديل حقول مثل `QtyDelivered` و `QtyInvoiced` لتصبح للقراءة فقط (Read-only) في API الطلبات، حيث سيتم حسابها تلقائياً من حركات المخزون والفواتير الفعلية.

> [!WARNING]
> **أمان ZATCA:** يتطلب إكمال المرحلة الثانية (Phase 2) توفير بيئة آمنة لإدارة المفاتيح الخاصة (Private Keys) والشهادات، حيث أن التخزين الحالي في قاعدة البيانات يحتاج إلى طبقة تشفير إضافية.

## Proposed Changes

---

### 1. Mail & Chatter Engine (البنية التحتية المشتركة)
هذا المكون هو "العمود الفقري" لكل المجالات الأخرى لضمان التدقيق والتواصل.

#### [NEW] [thread.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/domain/activity/thread.go)
تعريف كيانات `Thread`, `Follower`, `MessageSubtype`, و `TrackingValue`.
#### [MODIFY] [message.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/domain/activity/message.go)
تحديث الـ `Message` لتدعم `SubtypeID` و `ParentID` و `TrackingValues`.
#### [NEW] [thread_service.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/usecase/activity/thread_service.go)
تنفيذ منطق `PostMessage` الذي يقوم بـ:
- حفظ الرسالة.
- تسجيل التغييرات (Tracking).
- تحديد المستلمين بناءً على المتابعين (Followers) و الـ Subtype.
- إرسال إشعارات (Notifications) عبر الـ Bus أو البريد.

---

### 2. ZATCA & EDI (الفوترة الإلكترونية المرحلة الثانية)
تحويل المعالج الحالي من "تجريبي" إلى "متوافق قانونياً".

#### [MODIFY] [zatca_processor.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/usecase/accounting/zatca_processor.go)
- إحلال `fmt.Sprintf` بقوالب Go Templates لتوليد UBL 2.1 XML كامل.
- دمج مكتبة للتوقيع الرقمي (XAdES-EPES).
- تنفيذ حساب الـ QR Code بناءً على TLV الفعلي وقيم الفاتورة (Seller, VAT, Time, Total, VAT Total, Hash, Signature).
- إضافة منطق التحقق (Validation) باستخدام XSD.

---

### 3. Payment Providers (بوابات الدفع الخارجية)
فصل المدفوعات المحاسبية عن عمليات الدفع الإلكتروني.

#### [NEW] [transaction.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/domain/payment/transaction.go)
تعريف `PaymentTransaction` و `PaymentProvider`.
#### [NEW] [provider_handler.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/usecase/payment/provider_handler.go)
معالج الـ Webhooks مع التحقق من التوقيع (Signature Validation) وضمان عدم التكرار (Idempotency) باستخدام مفاتيح فريدة لكل معاملة.

---

### 4. Inventory Advanced Logic (المخزون المتقدم)
إضافة الذكاء لمسارات المخزون وعمليات الفرز.

#### [NEW] [route_engine.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/usecase/stock/route_engine.go)
محرك القواعد (Rules Engine) الذي يقرر نوع الحركة (Buy, Manufacture, Transfer) بناءً على `RouteID` و `ProcurementGroup`.
#### [NEW] [barcode_service.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/usecase/stock/barcode_service.go)
محلل GS1 للباركود لاستخراج الـ GTIN والـ Lot والـ Expiry تلقائياً.
#### [NEW] [scrap.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/domain/stock/scrap.go)
كيان مستقل لإدارة التوالف (Scrap) مع ربط محاسبي مباشر.

---

### 5. Cross-Domain Integrations (التكاملات العابرة)
ربط الحلقات المفقودة بين البيع والشراء والمخزون والمحاسبة.

#### [MODIFY] [order.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/domain/sale/order.go)
تحديث `UpdateInvoiceStatus` و إضافة `UpdateDeliveryStatus` لتعتمد على بيانات فعلية من الـ Repository بدلاً من التحديث اليدوي.
#### [MODIFY] [production.go](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/domain/mrp/production.go)
إضافة منطق الـ Backorder عند الإنتاج الجزئي وتوليد حركات المخزون للـ By-products.

---

## Verification Plan

### Automated Tests
- **Mail Thread Test:** التأكد من أن تعديل سعر منتج ينشئ `TrackingValue` في الـ Chatter.
- **ZATCA Validation Test:** تمرير الـ XML المولد عبر أداة التحقق الرسمية (أو محاكي لها).
- **Stock Rule Test:** التأكد من أن طلب بيع بمنتج يحتاج تصنيع ينشئ MO (Manufacturing Order) تلقائياً.
- **Idempotency Test:** إرسال Webhook دفع مرتين والتأكد من عدم إنشاء `Payment` مكرر.

### Manual Verification
- تجربة مسح باركود GS1 مركب والتأكد من تعبئة البيانات في واجهة الاستلام.
- فحص شجرة التتبع (Traceability) لمنتج من الشراء إلى البيع.
