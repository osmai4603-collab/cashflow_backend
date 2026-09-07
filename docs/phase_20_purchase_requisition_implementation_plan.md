# خطة تنفيذ المرحلة 20: طلبات الشراء المجمعة (Purchase Requisitions)

## 1. السياق والهدف

تُعنى هذه المرحلة ببناء نموذج طلبات الشراء المجمعة داخل المشروع الحالي، بما يتوافق مع السلوك الحقيقي في Odoo 19.0.

بعد مراجعة مصدر Odoo 19، تبين أن المفهوم الصحيح ليس وجود "Agreement" مستقل كـ addon منفصل، بل هو نموذج "Purchase Requisition" / "Blanket Order" / "Purchase Template" داخل وحدة المشتريات. لذلك، يجب تنفيذ هذه المرحلة على أساس أن الطلب المجمّع هو مستند شراء مركزي يمكن من خلاله:

- إنشاء طلب شراء مجمّع (Requisition)
- تحديد نوعه: Blanket Order أو Purchase Template
- إضافة أسطر المنتجات والكميات والسعر
- تأكيد الطلب وانتقاله إلى حالة نشطة
- إنشاء أوامر شراء من الطلب المجمّع
- إغلاقه أو إلغاؤه بعد انتهاء التزام المورد
- ربط كل أمر شراء بمصدره عبر requisition_id


## 2. مراجعة Odoo 19 والفرق بين الواقع والمخطط السابق

### ما تم التحقق منه

- توجد وحدة فعلية في Odoo 19 باسم purchase_requisition
- توجد وحدات ذات صلة: purchase_requisition_sale و purchase_requisition_stock
- لا يوجد موديل مستقل باسم agreement في بنية Odoo الأساسية
- الأنواع الأساسية هي:
  - blanket_order
  - purchase_template
- الحقول الفنية الأساسية على requisition هي:
  - name
  - vendor_id
  - user_id
  - date_start / date_end
  - state
  - currency_id
  - company_id
  - line_ids
  - purchase_ids
- الحقول الأساسية على requisition line هي:
  - product_id
  - product_uom_id
  - product_qty
  - price_unit
  - qty_ordered
  - requisition_id

### الخلاصة

المرحلة 20 يجب أن تُبنى على نموذج purchase.requisition وليس على مفهوم agreement منفصل، مع توحيد المصطلحات والربط الداخلي مع Purchase Order.


## 3. النطاق المقترح للتنفيذ

### 3.1 يشمل هذا العمل

- إنشاء نموذج PurchaseRequisition في طبقة domain
- إنشاء نموذج PurchaseRequisitionLine
- إنشاء حالات الطلب: draft, confirmed, done, cancel
- دعم نوعين رئيسيين:
  - blanket_order
  - purchase_template
- دعم التحقق من صحة البيانات وقواعد السلوك
- ربط الطلب المجمّع بأوامر الشراء الناتجة
- إنشاء API CRUD + إجراءات التشغيل
- دعم توثيق السجل وتأمين الوصول
- تنفيذ اختبارات الوحدة والتكامل الأساسية

### 3.2 لا يشمل هذه المرحلة في البداية

- مسابقات عروض الموردين المعقدة
- مقارنة الموردين المتعددة بشكل كامل
- تدفقات tender/auction المتقدمة
- قدرات مخزنية عميقة خارج الربط الأساسي مع الطلبات


## 4. هيكل التنفيذ المقترح داخل المشروع

### 4.1 ملفات domain

- internal/domain/purchase/requisition.go
- internal/domain/purchase/ports.go (تحديث الواجهة)

### 4.2 ملفات usecase

- internal/usecase/purchase/purchase_requisition_usecase.go

### 4.3 ملفات HTTP

- internal/adapters/http/purchase/requisition_handler.go
- internal/adapters/http/purchase/requisition_dto.go
- internal/adapters/http/purchase/requisition_routes.go

### 4.4 ملفات storage

- internal/adapters/storage/purchase/postgres_repo_requisition.go
- internal/adapters/storage/purchase/memory_repo_requisition.go

### 4.5 الترحيل

- migrations/XXXXXX_create_purchase_requisitions.up.sql
- migrations/XXXXXX_create_purchase_requisitions.down.sql

### 4.6 التعديلات على المشتريات الحالية

- توسيع PurchaseOrder في internal/domain/purchase/order.go بإضافة:
  - RequisitionID
  - RequisitionType
  - Origin
  - AlternativePOIDs


## 5. النماذج المقترحة

### 5.1 PurchaseRequisition

```go
type PurchaseRequisition struct {
    ID              int64
    Name            string
    Type            RequisitionType
    VendorID        *int64
    UserID          int64
    DateStart       *time.Time
    DateEnd         *time.Time
    State           RequisitionState
    CurrencyID      int64
    CompanyID       int64
    Description     string
    PurchaseOrderIDs []int64
    Lines           []PurchaseRequisitionLine
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### 5.2 RequisitionType

```go
type RequisitionType string

const (
    RequisitionBlanketOrder RequisitionType = "blanket_order"
    RequisitionTemplate     RequisitionType = "purchase_template"
)
```

### 5.3 RequisitionState

```go
type RequisitionState string

const (
    RequisitionDraft    RequisitionState = "draft"
    RequisitionConfirmed RequisitionState = "confirmed"
    RequisitionDone     RequisitionState = "done"
    RequisitionCancel   RequisitionState = "cancel"
)
```

### 5.4 PurchaseRequisitionLine

```go
type PurchaseRequisitionLine struct {
    ID            int64
    RequisitionID int64
    ProductID     int64
    ProductQty    float64
    ProductUOMID  *int64
    PriceUnit     float64
    ScheduleDate  *time.Time
    SupplierID    *int64
    Description   string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```


## 6. منطق العمل المقترح

### 6.1 إنشاء الطلب المجمّع

- التحقق من نوع الطلب
- التحقق من وجود السطور
- التحقق من أن المنتج صالح
- التحقق من أن الكمية > 0
- تعيين الحالة إلى draft
- إنشاء السجل في قاعدة البيانات

### 6.2 تأكيد الطلب

- لا يحق التأكيد إذا كانت الأسطر فارغة
- لا يحق التأكيد إذا كانت الكميات غير صالحة
- يجب التحقق من أن تاريخ النهاية لا يقل عن تاريخ البداية
- إذا كان الطلب من نوع blanket_order فيجب التحقق من المورد إن كان موجودًا
- بعد التأكيد يتم تعيين الحالة إلى confirmed

### 6.3 إنشاء أمر شراء من الاتفاقية

- تحديد requisition_id
- نسخ بيانات الطلب إلى PurchaseOrder
- نسخ السطور كما هي مع السعر والكميات المبدئية
- إنشاء تهيئة داخل domain layer و usecase layer
- ربط كل PO بـ requisition_id

### 6.4 الإغلاق والإلغاء

- Close: عندما يكتمل الطلب أو انتهت الاتفاقية
- Cancel: عند إلغاء القرار أو إلغاء الاتفاقية
- يمنع الإغلاق إذا كانت هناك أوامر شراء مفتوحة مرتبطة بالطلب


## 7. واجهات API المقترحة

### CRUD

- GET /api/v1/purchase-requisitions
- GET /api/v1/purchase-requisitions/{id}
- POST /api/v1/purchase-requisitions
- PUT /api/v1/purchase-requisitions/{id}
- DELETE /api/v1/purchase-requisitions/{id}

### إجراءات التشغيل

- POST /api/v1/purchase-requisitions/{id}/confirm
- POST /api/v1/purchase-requisitions/{id}/close
- POST /api/v1/purchase-requisitions/{id}/cancel
- POST /api/v1/purchase-requisitions/{id}/create-po

### ملاحظات على التصميم

- يجب أن تكون الـ routes متوافقة مع أنماط المشروع الحالي في [internal/adapters/http/purchase/routes.go](internal/adapters/http/purchase/routes.go)
- يجب الحفاظ على access control عبر auth.Authorizer مثل باقي الوحدات


## 8. التحديثات المقترحة على Repository

يجب تحديث interface Repository في [internal/domain/purchase/ports.go](internal/domain/purchase/ports.go) بما يلي:

```go
type RequisitionRepository interface {
    CreateRequisition(ctx context.Context, req *PurchaseRequisition) error
    GetRequisitionByID(ctx context.Context, id int64) (*PurchaseRequisition, error)
    ListRequisitions(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[PurchaseRequisition], error)
    UpdateRequisition(ctx context.Context, req *PurchaseRequisition) error
    DeleteRequisition(ctx context.Context, id int64) error
    CreatePOFromRequisition(ctx context.Context, reqID int64, po *PurchaseOrder) error
}
```

يُفضّل الاحتفاظ بنوع Repository الحالي كما هو مع إلحاق دوال requisition عليه، لأن المشروع يعتمد على repository واحد لكل domain.


## 9. تكامل المشروع الحالي

### 9.1 التوافق مع PurchaseOrder الحالي

النموذج الحالي في [internal/domain/purchase/order.go](internal/domain/purchase/order.go) يضم أوامر الشراء الأساسية فعليًا، لذا يجب تعديلها لإضافة الحقول التالية:

```go
type PurchaseOrder struct {
    ID              int64
    Name            string
    PartnerID       int64
    DateOrder       time.Time
    State           PurchaseOrderState
    ...
    RequisitionID   *int64
    RequisitionType *string
    AlternativePOIDs []int64
}
```

### 9.2 التوافق مع UseCase الحالي

يُفضل توسيع usecase الحالي داخل [internal/usecase/purchase/purchase_usecase.go](internal/usecase/purchase/purchase_usecase.go) بدوال مساعدة مثل:

- CreatePOFromRequisition
- LinkRequisitionToOrder
- ValidateRequisitionCompatibility

هذا يقلل من التغييرات الجذرية ويحافظ على البنية الحالية.


## 10. خطة التنفيذ المرحلية

### المرحلة 1: إعداد النموذج

- إنشاء PurchaseRequisition و PurchaseRequisitionLine
- إضافة حالات الطلب والأنواع
- توسيع PurchaseOrder بإضافة requisition linkage

### المرحلة 2: قاعدة البيانات

- إنشاء جداول requisitions و requisition_lines
- إضافة requisition_id في purchase_orders
- إضافة فهارس وسجلات audit

### المرحلة 3: repository

- تنفيذ إنشاء/قراءة/تحديث/حذف للطلبات المجمعة
- تنفيذ ربط PO بـ requisition

### المرحلة 4: usecase

- تنفيذ CreateRequisition
- تنفيذ ConfirmRequisition
- تنفيذ CreatePOFromRequisition
- تنفيذ CloseRequisition و CancelRequisition

### المرحلة 5: HTTP

- إنشاء handler و DTOs
- تسجيل routes
- اختبار الوصول من خلال API

### المرحلة 6: الاختبارات

- اختبار happy path
- اختبار حالات الفشل
- اختبار الربط بين Requisition و PurchaseOrder


## 11. اختبارات الجودة المطلوبة

### اختبارات الوحدة

- TestCreateRequisition_Valid
- TestCreateRequisition_RejectsEmptyLines
- TestConfirmRequisition_ChangesState
- TestCloseRequisition_WithOpenPOsFails
- TestCreatePOFromRequisition_LinksParentRequisition

### اختبارات التكامل

- إنشاء requisition
- تأكيده
- إنشاء PO منه
- ربط PO بـ requisition_id
- غلق requisition بنجاح
- إلغاء requisition عندما لا توجد أوامر مفتوحة

### التحقق التشغيلي

- go test ./internal/usecase/purchase/...
- go test ./...


## 12. المخاطر المحتملة وكيفية التعامل معها

### 12.1 التباس المصطلحات

الحل: استخدام اصطلاح PurchaseRequisition في المشروع، مع توضيح Blanket Order في التعليقات والـ API.

### 12.2 الربط الخاطئ مع PurchaseOrder

الحل: ضمان أن كل PO له requisition_id، وأن استخدام CreatePOFromRequisition يثبت هذا الربط فورياً.

### 12.3 حالات الطلب غير متناسقة

الحل: تنفيذ validation صارمة في domain layer قبل أي تحديث قاعدة.

### 12.4 إساءة فهم دور المخزون

الحل: لا يتم تعقيد المرحلة بالمخزون الآن، بل يُترك الربط الأساسي فقط، مع إمكانية توسعة لاحقة.


## 13. الخلاصة

المرحلة 20 هي مرحلة أساسية في دورة المشتريات، لأنها تُنشئ القاعدة لنظام Agreements / Requisitions كآلية شراء مركزية داخل المشروع. التنفيذ الصحيح يعتمد على فهم Odoo 19 الحقيقي: purchase.requisition كمستند أساسي، مع ربط مباشر بأوامر الشراء والرحلة من draft إلى confirmed إلى done/cancel.

بناءً على هذا، يُعد التنفيذ الأمثل هو:

1. إضافة model requisition
2. توسيع PurchaseOrder بربط requisition_id
3. تنفيذ usecase و HTTP
4. تجميعها في طبقة repository
5. التحقق عبر اختبارات الوحدة والتكامل


## 14. ملف التنفيذ المطلوب

هذا الملف يُعد وثيقة التنفيذ الرسمية للمرحلة 20، ويُسجل في مجلد docs داخل المشروع ليكون مرجعاً واضحاً قبل بدء التطوير الفعلي.
