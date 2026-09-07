# خطة تنفيذ المرحلة 15: الأنشطة والتنبيهات

## 1. الهدف والنطاق

تنفيذ نظام أنشطة متوافق مع مبادئ Odoo 19، قابل للربط بالكيانات التجارية في Cashflow، مع منظومة تنبيهات كاملة.

يشمل النطاق:

- Activities polymorphic مرتبطة بـ `res_model` و`res_id`.
- أنواع الأنشطة الافتراضية والقابلة للإدارة.
- حالات محسوبة حسب timezone المستخدم: `overdue`, `today`, `planned`.
- عزل البيانات حسب الشركة والصلاحيات حسب المستخدم والدور.
- إكمال النشاط مع الاحتفاظ بتاريخه وأرشفته بدلا من حذفه.
- إشعارات داخل النظام، البريد الإلكتروني، الطابور، وإعادة المحاولة.
- Worker/Cron للتذكيرات والتنظيف.
- Bus/WebSocket للتحديث اللحظي.

## 2. قرارات التصميم

- `active=false` هو الأرشفة؛ الإكمال يحفظ `date_done` و`feedback` ولا يحذف السجل.
- `DELETE` في API يعني الأرشفة، والحذف النهائي عملية إدارية منفصلة إن لزم.
- `state` قيمة محسوبة ولا يقبلها الخادم من العميل.
- كل Activity وNotification مرتبطان بـ `company_id`.
- `res_model` لا يقبل أي قيمة عشوائية؛ يستخدم whitelist ويتحقق من صلاحية الوصول إلى السجل الهدف.
- `assigned_user_id` هو المسؤول، و`created_by` هو المنشئ. لا يستخدم اسم غامض مثل `RequestPartnerID`.
- حساب اليوم والموعد يعتمد على timezone المستخدم، مع تخزين التواريخ بصيغة UTC.
- البريد والتذكيرات idempotent وقابلة لإعادة المحاولة، ولا يرتبط Domain مباشرة بوسيلة النقل.
- chaining للأنشطة يؤجل إلى عقد مستقل، إلا إذا تطلبت متطلبات المنتج إدخاله ضمن هذه المرحلة.

## 3. الاعتماديات

تعتمد المرحلة على:

- البنية الأساسية وPostgreSQL والمهاجرات.
- المستخدمين والشركات وJWT claims.
- RBAC وrecord rules وسياق الشركة.
- المرفقات polymorphic.
- Worker الحالي وإعدادات SMTP.
- المرحلة 11 عند الحاجة إلى عرض أنشطة كشوف البنك.

مرجع Odoo 19 المحلي:

- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/models/mail_activity.py`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/models/mail_activity_type.py`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/models/mail_activity_mixin.py`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/models/mail_message.py`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/models/mail_notification.py`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/models/mail_thread.py`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/data/mail_activity_type_data.xml`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/security/ir.model.access.csv`
- `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/mail/tests/test_mail_activity.py`

## 4. خطة التنفيذ

### 15.1 العقد والسياسات

- تثبيت نموذج Activity وActivityType وNotification وMailMessage.
- تعريف whitelist للنماذج المدعومة.
- تعريف سياسة الرؤية والتعديل والإنجاز والأرشفة.
- توثيق API payloads، filters، pagination، sorting، والأخطاء.

### 15.2 قاعدة البيانات

إضافة:

- `migrations/000020_create_activity_schema.up.sql`
- `migrations/000020_create_activity_schema.down.sql`

الجداول المقترحة:

- `mail_activity_types`
- `mail_activities`
- `mail_messages`
- `mail_notifications`
- جداول طابور البريد عند الحاجة إلى فصلها عن الرسائل.

القيود والفهارس المطلوبة:

- `company_id` مرجع إلزامي في السجلات المناسبة.
- `res_model` و`res_id` إما موجودان معا أو غير موجودين معا.
- `date_deadline` إلزامي للنشاط.
- `date_done` و`feedback` يملآن عند الإكمال.
- فهارس على `(company_id, assigned_user_id, active, date_deadline)` و`(company_id, res_model, res_id, active)` و`(activity_type_id, active)`.
- seed للأنواع: Email, Call, Meeting, To-Do, Document, Exception.

### 15.3 Domain وPorts

إنشاء `internal/domain/activity/` ويتضمن:

- `ActivityType` و`Activity` و`Notification`.
- validation للمرجع والموعد والنوع.
- حساب الحالة حسب timezone.
- transitions للإنجاز وإعادة الجدولة والأرشفة.
- ports للتخزين، الإشعار، البريد، والوقت عند الحاجة للاختبار.

### 15.4 التخزين

إنشاء PostgreSQL وMemory repositories في `internal/adapters/storage/activity/`.

المتطلبات:

- تمرير company scope إلى كل استعلام.
- تطبيق record rules قبل إرجاع البيانات.
- pagination وfiltering وsorting حتمية.
- إكمال ذري عبر تحديث مشروط بـ `active=true`.
- اختبارات عزل الشركات والتزامن وعدم العثور على السجل.

### 15.5 Use Cases

إنشاء `internal/usecase/activity/` للعمليات التالية:

- CRUD لأنواع الأنشطة.
- إنشاء وجدولة النشاط.
- قائمة أنشطة المستخدم والمتأخرة وأنشطة الكيان.
- إعادة الجدولة وتغيير المسؤول.
- الإكمال مع feedback.
- الأرشفة.
- إصدار إشعار عند الإسناد أو تغيير المسؤول.

### 15.6 HTTP والتسجيل

إنشاء `internal/adapters/http/activity/`:

- `handler.go`
- `dto.go`
- `routes.go`

المسارات:

- `/api/v1/activity-types`
- `/api/v1/activities`
- `/api/v1/activities/my`
- `/api/v1/activities/overdue`
- `/api/v1/activities/{id}/done`
- `/api/v1/{model}/{id}/activities`
- مسارات notifications وmark-read.

يتم تسجيل الاعتماديات في `internal/adapters/http/router.go` و`cmd/server/main.go`.

### 15.7 البريد والتنبيهات اللحظية

- إنشاء Notification inbox مع `read_at`.
- بناء mail message وemail queue بحالات retry وdead-letter.
- استخدام SMTP config الموجود دون تسريب أسرار الإعدادات.
- إضافة worker/Cron idempotent للتذكيرات والتنظيف.
- إضافة Bus/WebSocket adapter لتحديث عدادات الأنشطة والتنبيهات.
- إبقاء Domain مستقلا عن SMTP وWebSocket.

### 15.8 الصلاحيات والاختبارات

- إضافة ACL وrecord rules للأنشطة والأنواع والتنبيهات.
- unit tests للـ Domain وUse Cases.
- repository integration tests للمهاجرات والعزل والذرية.
- HTTP tests للعقود والclaims والصلاحيات.
- اختبار البريد الفاشل وإعادة المحاولة والتحديث اللحظي.

## 5. معايير القبول

- لا يستطيع المستخدم قراءة أو تعديل نشاط خارج شركته.
- لا يستطيع العميل فرض `company_id` أو `state`.
- لا يقبل النظام نموذجا polymorphic غير مسجل أو سجلا لا يملك المستخدم صلاحية الوصول إليه.
- الإكمال المتزامن ينتج انتقالا واحدا ورسالة واحدة فقط.
- النشاط المكتمل محفوظ تاريخيا ومؤرشف.
- `overdue/today/planned` صحيحة حسب timezone المستخدم.
- البريد يعاد إرساله وفق سياسة retry ولا يكرر الإشعار عند نجاحه.
- `go test ./...` ينجح، وتنجح اختبارات migration up/down.

## 6. حالة التنفيذ

| البند | الحالة | الملاحظات |
|---|---|---|
| وثيقة الخطة | مكتمل | تم إنشاء هذا الملف وتثبيت نطاق المرحلة |
| عقد السياسات وAPI | قيد التنفيذ | سيؤكد أثناء بناء Domain وHTTP |
| Migration 000020 | قيد التنفيذ | أول شريحة برمجية |
| Domain وPorts | لم يبدأ | بعد تثبيت schema |
| Repositories | لم يبدأ | يعتمد على Domain وschema |
| Use Cases | لم يبدأ | يعتمد على repositories |
| HTTP والتسجيل | لم يبدأ | يعتمد على use cases |
| Notifications/Mail/Worker/WebSocket | لم يبدأ | تنفيذ متتابع بعد Activity الأساسية |
| ACL والاختبارات الشاملة | لم يبدأ | تنفذ مع كل طبقة ثم تجمع نهائيا |

## 7. سجل التغييرات أثناء التنفيذ

### 2026-09-07

- إنشاء وثيقة الخطة التنفيذية.
- تثبيت نطاق التنبيهات الكامل: inbox وSMTP وemail queue وworker/cron وrealtime.
- اعتماد Odoo 19 المحلي من `/home/osm/Downloads/odoo-19.0/odoo-19.0` كمصدر مقارنة.
- بدء تنفيذ migration `000020`.
- إكمال جلب بريد المستخدم الفعلي عند إسناد النشاط وإضافته إلى طابور البريد بعد التحقق من صحة العنوان.
- إضافة ناقل إشعارات user-scoped داخل العملية، وربطه بـ Activity UseCase.
- إضافة بث SSE محمي عبر `/api/v1/notifications/stream` للتنبيهات اللحظية.
- إضافة اختبارات لتسليم الأحداث للمستخدم الصحيح وإغلاق الاشتراك.
- إضافة تفضيل `email_notifications_enabled` للمستخدم مع migration `000028`، ودعم opt-out دون تعطيل إشعارات inbox أو البث اللحظي.
- إضافة PostgreSQL relay عبر `LISTEN/NOTIFY` لتوزيع الأحداث بين نسخ الخدمة مع منع تكرار الحدث داخل النسخة المنشئة.
- إضافة اختبارات HTTP لمسار SSE تشمل المصادقة، الاتصال، تسليم الحدث، وإغلاق الاتصال.

## 8. الحالة الحالية والمتبقي

| البند | الحالة | الملاحظات |
|---|---|---|
| البريد الفعلي وطابور الإرسال | مكتمل | يتم حل البريد من `res.users` مع التحقق قبل الإدراج في الطابور |
| تفضيلات إرسال المستخدم | مكتمل | `email_notifications_enabled` افتراضيًا مفعّل وقابل للتعطيل من API المستخدم |
| التحديث اللحظي داخل العملية | مكتمل | Bus user-scoped وSSE endpoint محمي بالصلاحيات |
| التحديث اللحظي متعدد العمليات | مكتمل | PostgreSQL `LISTEN/NOTIFY` relay اختياري عند تشغيل PostgreSQL |
| اختبارات HTTP التكاملية | مكتمل جزئيًا | تمت تغطية مسار SSE الأساسي، وتبقى حالات الصلاحيات التفصيلية للتوسع لاحقًا |
