# المرحلة 19: إدارة المصروفات - خطة التنفيذ المرحلية

## الهدف والنطاق

تنفيذ سلوك `hr.expense` في Odoo 19 داخل المشروع، بدءاً من إنشاء المصروف وحتى الاعتماد والترحيل والدفع، مع الضرائب والمرفقات والتوزيع التحليلي وكشف التكرار والأنشطة.

النموذج الأساسي هو `Expense` موحد. لا يُنفذ `ExpenseReport` كنموذج بديل؛ يمكن إضافة تجميع للمصروفات لاحقاً فوق المصروفات الفردية.

## قواعد Odoo المعتمدة

- الحالات: `draft`, `submitted`, `approved`, `posted`, `in_payment`, `paid`, `refused`.
- حالة `posted` و`paid` مشتقتان من حالة القيد/الدفع، وليستا انتقالاً يدوياً مستقلاً فقط.
- `own_account`: المصروف مدفوع من الموظف ويُنشأ له قيد مستحقات قابل للدفع.
- `company_account`: المصروف مدفوع من الشركة، ويصل إلى `paid` عند ترحيل حركة الشركة.
- عند عدم وجود مدير مصروفات، يتجاوز الإرسال مرحلة `submitted` ويُعتمد تلقائياً.
- مدير المصروفات هو `employee.expense_manager_id`، مع fallback إلى مدير الموظف/مدير القسم حسب صلاحية المستخدم.
- لا يُعتمد مصروف غير صفري؛ المصروف الصفري مسموح في `draft` فقط.
- التكرار وتكرار receipt يطلقان تحذيراً قبل الاعتماد، مع قرار صريح لاحقاً هل يكون المنع قابلاً للتجاوز.
- الضرائب في Odoo للمصروفات تعامل السعر المدخل كسعر شامل عند حساب إجمالي المصروف.
- تقسيم المصروف ينشئ مصروفات جديدة مرتبطة بـ `split_origin_id` ولا يسمح بتقسيم مصروف مرحّل أو مدفوع.

## المراحل وحالة التنفيذ

| المرحلة | النطاق | الحالة | معيار الإغلاق |
|---|---|---|---|
| 0 | تثبيت السلوك والعقد ومقارنة Odoo | مكتملة | توثيق الحالات، الصلاحيات، الضرائب، الدفع والتقسيم |
| 1 | Domain وHR contract | مكتملة | Domain/HR tests و`ExpenseManagerID` عبر DTO/SQL |
| 2 | Schema والتخزين | مكتملة | migrations تصحيحية، PostgreSQL/Memory repository واختبارات mapping |
| 3 | Use Cases ودورة الحالات | مكتملة | CRUD وSubmit/Approve/Refuse/Split مع duplicate |
| 4 | الضرائب والمحاسبة والدفع | مكتملة | قيد `in_receipt` متوازن، idempotency واختبار الدفع المحاسبي |
| 5 | HTTP وACL والتسجيل | مكتملة | routes وhandlers وcomposition root وACL وActivity عند Submit |
| 6 | اختبارات القبول والإغلاق | مكتملة للمصروفات | اختبارات Expense وPostgreSQL تمر؛ `go test ./...` متوقف بسبب أخطاء Loyalty خارج المرحلة |

تم تنفيذ المراحل بالتتابع ولم يتم الانتقال قبل نجاح اختبار المرحلة الحالية. النواة البرمجية ودورة المصروف والمحاسبة وActivity وschema PostgreSQL مكتملة. الفحص اليدوي لرفع Receipt ودورة API الكاملة يبقى خطوة تشغيلية، بينما فشل `go test ./...` الحالي محصور في أخطاء Loyalty خارج المرحلة 19.

## المرحلة 1: Domain وHR contract

1. إضافة `in_payment` إلى عقد الحالة، والتحقق الصريح من قيم الحالة وطريقة الدفع.
2. إضافة Tax IDs وبيانات receipt اللازمة لكشف التكرار دون نسخ نظام المرفقات.
3. تطبيق قواعد Odoo الأساسية: المبلغ غير السالب، الكمية الافتراضية، منع الصفر بعد `draft`، والتحقق من split.
4. منع الموظف من تعيين نفسه كمدير مصروفات.
5. تمرير `ExpenseManagerID` عبر Create/Update Employee DTOs وUse Case وPostgreSQL/Memory mapping.
6. إضافة اختبارات Domain وHR.

## المرحلة 2: Schema والتخزين

1. إنشاء migration تصحيحية لا تعديل migration مطبقة.
2. إضافة قيود الحالات، payment mode، المبالغ، والفهارس.
3. تحديد علاقة receipt بالمرفق وتخزين checksum عبر attachment subsystem.
4. تنفيذ PostgreSQL وMemory repositories مع filters وpagination وtax/attachment joins.
5. تسجيل repositories في composition root واختبار PostgreSQL.

## المرحلة 3: Use Cases ودورة الحالات

1. CRUD مع ownership وcompany scope ومنع التعديل بعد الإرسال/الترحيل.
2. Submit مع تعيين المدير أو autovalidation وإنشاء Activity.
3. Approve مع الصلاحيات، self-approval، zero amount، duplicate date/amount/employee وreceipt checksum.
4. Refuse بسبب إلزامي، وreset وفق قواعد Odoo.
5. Split داخل transaction مع rounding ومساواة المجموع وربط الأصل.

## المرحلة 4: الضرائب والمحاسبة والدفع

1. إعادة استخدام خدمة الضرائب الموجودة ودعم included/excluded مع rounding المحاسبي.
2. إنشاء قيد متوازن: حساب المصروف والضريبة مقابل payable للموظف أو حساب الشركة.
3. ربط `AccountMoveID` ومنع الترحيل المكرر.
4. تمرير التحليلية إلى AccountMoveLine/analytic lines.
5. ربط `posted -> in_payment -> paid` بنتيجة Payment/Reconciliation الفعلية.

## المرحلة 5: HTTP وACL والتسجيل

1. تنفيذ DTOs وhandlers وroutes لـ CRUD وsubmit/approve/refuse/split/post و`to-approve`.
2. تسجيل الوحدة في repositories/usecases/handlers/router.
3. إضافة صلاحيات `hr.expense` وربطها بعلاقة الموظف والمدير والشركة.
4. ربط activities والـ notification queue دون إفساد transaction الأساسية.

## المرحلة 6: الاختبارات والإغلاق

- Unit: validation، taxes، rounding، split، duplicate، checksum.
- Use Case: كل انتقال حالة والصلاحيات وautovalidation وidempotency.
- Integration: migration، repository، قيد متوازن، الدفع، rollback.
- HTTP: DTOs، status codes، ACL، filters.
- قبول يدوي: receipt، split، activity، approve، post، ledger.

## مراجع التنفيذ

- المشروع: `internal/domain/expense`, `internal/domain/hr`, `internal/usecase/accounting`, `internal/usecase/activity`.
- Odoo: `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/hr_expense/models/hr_expense.py`.
- اختبارات Odoo: `addons/hr_expense/tests/test_expenses.py` و`test_expenses_states.py`.
- صلاحيات Odoo: `addons/hr_expense/security/hr_expense_security.xml`.