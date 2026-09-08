# خطة تنفيذ المرحلة 20: طلبات الشراء المجمعة (Purchase Requisitions)

> **حالة التنفيذ:** المرحلة 20 قيد التنفيذ المرحلي. لا يتم الانتقال إلى المرحلة التالية قبل اجتياز بوابة المرحلة الحالية.
>
> **المرحلة الحالية:** مكتملة.
>
> **الحالة:** المرحلة 20 مكتملة ضمن نطاقها المنفذ، مع إخفاقات تكاملية خارجية موثقة أدناه.

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

## 2.1 العقد السلوكي المعتمد للمرحلة 20

يُعد هذا العقد المرجع الذي ستُبنى عليه اختبارات المراحل اللاحقة، ولا يجوز تغيير السلوك لمجرد توافق أسماء الحقول.

| مفهوم Odoo | عقد Cashflow | القاعدة الإلزامية |
|---|---|---|
| `purchase.requisition` | `PurchaseRequisition` | مستند Purchase Agreement المركزي، وليس كيان Agreement مستقلًا |
| `requisition_type` | `type` | القيم الوحيدة: `blanket_order` و`purchase_template` |
| `draft` | `draft` | الحالة الوحيدة القابلة للتعديل الكامل |
| `confirmed` | `confirmed` | اتفاقية نشطة قابلة لإنشاء RFQ |
| `done` | `done` | مغلقة؛ لا تُغلق مع RFQ مفتوح |
| `cancel` | `cancel` | ملغاة؛ يمكن حذفها وفق قاعدة الحذف المعتمدة |
| `product.product` | `ProductID` | يجب أن يشير إلى منتج قابل للشراء، لا إلى قالب منتج غير محدد |
| `fields.Date` | `DateStart/DateEnd` | تاريخ تقويمي؛ إذا استُخدم `time.Time` في Go فالتخزين والتحويل يجب أن يكونا ثابتين ومعلنين |

### مصفوفة النوعين

| السلوك | `blanket_order` | `purchase_template` |
|---|---|---|
| المورد عند التأكيد | مطلوب | اختياري وفق سلوك Odoo |
| السعر عند التأكيد | أكبر من صفر لكل سطر | يُحسب من supplier info أو `standard_price` عند توفر المورد/المنتج |
| كمية سطر الاتفاقية | أكبر من صفر | تُستخدم ككمية القالب |
| Supplier Info | يُنشأ ويرتبط بسطر الاتفاقية | لا يُنشأ من الاتفاقية نفسها |
| كمية RFQ المنشأ | صفر، مع نسخ السعر | تُنسخ من كمية السطر |
| إغلاق الاتفاقية | يحذف Supplier Info بعد اجتياز فحص RFQs | لا يوجد Supplier Info اتفاقية للحذف |

### مصفوفة انتقالات الحالة

| من | إلى | الشرط |
|---|---|---|
| `draft` | `confirmed` | توجد أسطر؛ وتُطبق قواعد النوع؛ وتنجح كل عمليات Supplier Info اللازمة |
| `confirmed` | `done` | لا يوجد PO/RFQ في `draft` أو `sent` أو `to approve` |
| `draft` أو `confirmed` | `cancel` | صلاحية الإلغاء متوفرة؛ وتُلغى RFQs القابلة للإلغاء ويُنظف Supplier Info |
| `done` | أي حالة | ممنوع |
| `cancel` | `draft` | لا يُسمح به إلا إذا قرر عقد API ذلك صراحة؛ ليس افتراضًا افتراضيًا |

### صلاحيات العقد

- Purchase User: إنشاء وقراءة وتعديل وحذف وفق حالة السجل، وتنفيذ انتقالات التشغيل المسموحة.
- Purchase Manager: قراءة وإدارة أوسع وفق نموذج الصلاحيات المحلي.
- Manage Purchase Alternatives: إدارة مجموعات Alternative POs فقط.
- كل عملية قراءة أو تعديل أو حذف يجب أن تحترم `company_id` في طبقة use case وrepository، وليس في HTTP فقط.

### بوابة إغلاق المرحلة 1

- تم تثبيت mapping الأسماء والحالات والأنواع في هذه الوثيقة.
- تم تثبيت الفرق السلوكي بين Blanket Order وPurchase Template.
- تم تثبيت انتقالات الحالات وقواعد الصلاحيات وتعدد الشركات.
- تمت مطابقة العقد مع ملفات Odoo: `purchase_requisition.py`, `purchase.py`, `product.py`، و`test_purchase_requisition.py`.
- **نتيجة البوابة:** المرحلة 1 مكتملة توثيقيًا، ولا يبدأ تنفيذ المرحلة 2 إلا بعد إضافة اختبارات العقد أو اعتماد الاختبارات الموجودة التي تثبت هذه القواعد.


## حالة المرحلتين 1 و2

- **المرحلة 1 - العقد السلوكي والمطابقة:** مكتملة. تم تثبيت mapping الأنواع والحالات وقواعد Blanket Order وPurchase Template والصلاحيات وتعدد الشركات، مع بوابة قبول موثقة.
- **المرحلة 2 - Domain والقيود:** مكتملة. تم تنفيذ والتحقق من القيم المسموحة، قواعد تأكيد Blanket Order، انتقال الإلغاء، منع تعديل تعريف الاتفاقية بعد التأكيد، والحقول الأساسية `active`, `reference`, `order_count`, `qty_ordered` وبيانات وصف/ربط السطر. نجحت اختبارات `go test ./internal/domain/purchase ./internal/usecase/purchase ./internal/adapters/http/purchase`.
- **المرحلة 3 - قاعدة البيانات والهجرات:** مكتملة. أضيفت migration `000044_align_purchase_requisition_odoo19` بحقول الاتفاقية وقيود النوع/الحالة/التاريخ/الكميات، والفهارس وتسلسلا Blanket/Template. نجحت اختبارات migrations، والتطبيق الحي، وrollback ثم replay على PostgreSQL.
- **المرحلة 4 - Repository والمعاملات:** مكتملة. تم حفظ الحقول الجديدة، وتطبيق company scope على PostgreSQL وMemory، وإضافة اختبار عزل الشركات. نجحت اختبارات `go test -mod=mod ./internal/adapters/storage/purchase ./internal/usecase/purchase ./internal/adapters/http/purchase`، مع إبقاء تغيير `go.mod` خارج نطاق المرحلة.
- **المرحلة 5 - دورة الأعمال وSupplier Info:** مكتملة. تم تصحيح الإغلاق قبل تغيير الحالة، إلغاء RFQs draft، وإضافة Supplier Info لـ Blanket Order في PostgreSQL وMemory مع migration `000045` واختبارات lifecycle وrollback/replay.
- **المرحلة 6 - إنشاء RFQ وPO:** مكتملة. فُرضت requisition المؤكدة، وطُبق الفرق بين كمية Blanket الصفرية وكمية Template المنسوخة، وأصبح تأكيد PO يرفض الصفر. نجحت اختبارات Domain وUse Case وRepository وHTTP للمسار.
- **المرحلة 7 - البدائل وتكامل Purchase Orders:** مكتملة. أضيفت مجموعات بدائل مستقلة عبر migration `000046`، وعمليات الإنشاء/العرض/الفك في PostgreSQL وMemory، وإلغاء RFQs البديلة عند تأكيد PO، مع اختبارات lifecycle وrollback/replay.
- **المرحلة 8 - HTTP وACL:** مكتملة. أضيفت endpoints إنشاء/عرض/فك البدائل، واختبارات HTTP الحالية تمر، وطُبقت migration `000047` بصلاحيات `purchase.order.alternative`.
- **المرحلة 9 - اختبارات المطابقة والإغلاق:** مكتملة. نجحت الاختبارات المركزة `go test -mod=mod ./internal/domain/purchase ./internal/usecase/purchase ./internal/adapters/storage/purchase ./internal/adapters/http/purchase ./migrations`، ونجحت migrations 000044-000047 في PostgreSQL مع rollback/replay حيث اختُبرت. فشل `go test ./...` محصور في موديول `delivery` ومشكلة import في اختبار `loyalty` خارج نطاق المرحلة 20.

### نتيجة الإغلاق

- **السلوك المنفذ:** Purchase Requisition، Blanket Order، Purchase Template، Supplier Info، دورة الحالات، إنشاء RFQ/PO، الكميات الصفرية للـ Blanket RFQ، `qty_ordered`، company scope، Alternative PO Groups، HTTP، ACL، واختبارات المطابقة الأساسية.
- **Migrations المنفذة:** `000044_align_purchase_requisition_odoo19`, `000045_create_purchase_supplier_info`, `000046_create_purchase_order_groups`, `000047_seed_purchase_requisition_alternative_acl`.
- **الفشل الخارجي:** `delivery` يحتوي imports وواجهات غير متوافقة (`regexp`, `strings`, وواجهات Repository ناقصة، و`sql.ErrNoNotFound`)؛ واختبار `loyalty` لديه import build failure. لم تُعدّل هذه الملفات لأنها خارج المرحلة 20.
- **قرار المرحلة 25:** لم تُحدّث إلى مكتملة؛ تحديثها مشروط بتنفيذ جميع المراحل والتحقق منها فعليًا.

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
