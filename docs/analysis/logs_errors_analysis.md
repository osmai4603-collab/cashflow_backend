# تحليل أخطاء ملفات السجلات (logs)

> المصدر: `logs/app.log` (71,242 سطر) و `logs/error.log` (417 سطر) — آخر تحديث 2026-09-16 (حتى 05:52).
> الغرض: تحليل الأخطاء الواردة في السجلات وتحديد الأسباب الجذرية والتوصيات.
> ملاحظة: هذا التحليل محدّث فوق النسخة السابقة (كانت حتى 04:18)؛ تمت إضافة أحداث 05:48 (leads) و 05:52 (account.budget).

---

## 0) ملخص تنفيذي

- **406 سجل ERROR** + **11 سجل WARN** في الفترة كلها، ولا يوجد أي FATAL أو PANIC.
- الأخطاء موزعة على **3 تشغيلات** مختلفة للعملية (مرّات بدء/إعادة تشغيل):
  - تشغيل 2026-09-14 05:31–05:32 → `404` اختباري + فيضان `503 /readyz` أثناء الـ drain (378 مرة).
  - تشغيل 2026-09-16 02:41–04:44 → مشاكل RBAC، `500 /notifications/stream`، `500 /crm/stages`، مسارات غير موجودة.
  - تشغيل 2026-09-16 05:16– → **جديد:** `500 /api/v1/leads/` (4 مرات) و `400 /api/v1/account.budget` (مرتين).
- أغلب أخطاء 500 حدثت في كود **قبل إصلاحات العمل الجاري** (غير المكرّسة). إصلاحات الوسوم التالية موجودة في شجرة العمل (Working Tree) لكن **لم تُنشر بعد**.
- أسبقية قصوى لمشكلتين مفتوحتين: **قراءة lead بلا COALESCE** (خطأ 500 متكرر)، و**مسار account.budget** (لا يوجد له Route أصلاً).

---

## 1) ملخص إحصائي

| الحالة | العدد | المسار | التفسير |
| ------- | ------- | -------- | --------- |
| 503 | 378 | `GET /readyz` | فشلُ فحص جاهزية متوقّع أثناء drain (إعادة تشغيل) في 2026-09-14 05:32 |
| 500 | 20 | `GET /api/v1/notifications/stream` | فشل SSE — `streaming is not supported` (91 بايت) |
| 500 | 4 | `GET /api/v1/crm/stages` | `failed to scan stage` (85 بايت) — قيمة NULL في `requirements` |
| 500 | 4 | `GET /api/v1/leads/` | `failed to scan lead` (84 بايت) — عمود nullable بدون COALESCE *(جديد)* |
| 403 | 2 | `POST /api/v1/moves/3/cancel` | نقص صلاحية `account.move` (write) |
| 403 | 2 | `GET /api/v1/bank-statements/` | نقص صلاحية `account.bank.statement` (read) |
| 400 | 2 | `GET /api/v1/account.budget` | خطأ Validation/طلب غير مدعوم — لا يوجد Route لها في الكود الحالي *(جديد)* |
| 404 | 2 | `GET /api/v1/account.budget/` | مسار غير موجود أصلاً (بلا نقطة نهاية Budget) |
| 404 | 1 | `GET /api/v1/partners/suppliers/` | شرطة مائلة زائدة (trailing slash) |
| 404 | 1 | `GET /nonexistent-endpoint` | اختبار/فحص قصير |
| 401 | 1 | `GET /api/v1/reports/dashboard` | طلب بدون توكن صالح |

المجموع: **406 ERROR + 11 WARN** — لا توجد أخطاء FATAL/PANIC في هذه الفترة.

---

## 2) المشاكل الجذرية

### 2.1 أخطاء 500 على `/api/v1/notifications/stream` (20 مرة) — **مُعالَجة في العمل الجاري، تحتاج نشراً**

**الموقع:** `internal/adapters/http/activity/handler.go:187` (`StreamNotifications`).

- حجم الجسم = **91 بايت** = مطابقة تامة لـ `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"streaming is not supported"}}`.
- الكود كان يصل لمسار الـ type assertion الفاشل:

  ```go
  flusher, ok := w.(http.Flusher)
  if !ok { ... }
  ```

  رغم أن الـ `http.ResponseWriter` الأصلي يدعم `Flush`. السبب: سلسلة الالتفاف عبر `metrics.statusWriter` (`metrics.go:84`) و`middleware.NewWrapResponseWriter` لا يطبّقان `http.Flusher` في كل الحلقات، فيفشل الفحص في `StreamNotifications`.

- الأخطاء تكررت في دفعات (03:20، 03:21، 04:09، 04:13، 04:18) أي أن الواجهة تحاول إعادة الاتصال باستمرار وتفشل.

**حالة الإصلاح (في شجرة العمل غير المكرّسة):**

- `activity/handler.go` أصبح يستخدم `http.NewResponseController` (يمشي على سلسلة `Unwrap`) مع fallback لـ `http.Flusher`.
- `metrics.go` أضاف `Flush()` إلى `statusWriter` يعرّف واجهة `http.Flusher`.
- يوجد اختبار `metrics_flush_test.go` و `stream_handler_test.go`.

**الإجراء المتبقي:** نشر النسخة وإعادة فحص `error.log` لرصد عدم عودة 500.

---

### 2.2 أخطاء 500 على `/api/v1/crm/stages` (4 مرات) — **سبب محتمل NULL في `requirements`، مُعالَج في العمل الجاري**

**الموقع:** `internal/adapters/storage/crm/postgres_repo.go:334` (`ListStages`).

- حجم الجسم = **85 بايت** = مطابقة لـ `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"failed to scan stage"}}` — الخطأ عند `rows.Scan` (سطر 355) وليس عند تنفيذ الاستعلام.
- العمود `requirements TEXT` في `crm_stages` **Nullable** بلا قيمة افتراضية (مخطط `000008`)، والاستعلام القديم (قبل الإصلاح) قرأه إلى `string` **دون** `COALESCE`. أي صف بقيمة `NULL` ينتج خطأ pgx القياسي: `converting NULL to string is unsupported`.

**حالة الإصلاح (في شجرة العمل):** تمت إضافة `COALESCE(fold, false)` و`COALESCE(requirements, '')` في كل استعلامات المراحل المفعّلة (`ListStages`, `GetStageByID`, `GetWonStage`, `GetInitialStage`, ...).

> ملاحظة جانبية: ترحيل `000050_localize_all_entities` حوّل `crm_stages.name` إلى JSONB، و`i18n.TranslationString.Scan` أصبح يقبل `[]byte` و`string` (تم تقويته في `i18n.go` ضمن العمل الجاري) — أي أن `name` ليس مصدر الفشل.

---

### 2.3 فشل فحص الجاهزية `/readyz` (378 مرة — 503) — **مُعالَج (تقليل الضجيج)، لا يحتاج فحص**

**الموقع:** `internal/infrastructure/runtime/health/health.go:77` + `router.go:289`.

- كل الأخطاء حدثت في **2026-09-14 05:32:48–05:32:53** أثناء `drain` مؤقت (5 ثوانٍ) أعقبته إعادة بدء عند 05:32:58 — حجم الجسم 69 بايت (`not_ready`).
- كثافة 378 طلباً خلال ~5 ثوانٍ تعني وجود مُصوّر (poller) ضيق يسحب `/readyz`؛ الحالة طبيعية لكنها تملأ `error.log`.

**حالة الإصلاح (في شجرة العمل):** `router.go` يخفض 503-probe إلى مستوى `Debug` ويصفّي التتبع المنفصل للحالة «not ready» حتى لا تختلط بـ ERROR؛ كما يُسجَّل الخطأ الفعلي للأكواد ≥500.

---

### 2.4 أخطاء 500 على `/api/v1/leads/` (4 مرات) — **مفتوحة (أعلى أولوية حالياً)**

**الموقع:** `internal/adapters/storage/crm/postgres_repo.go:194` (`ListLeads`).

- حدثت في **2026-09-16 05:48:39–05:48:43** (تشغيل 05:16)، وحجم الجسم = **84 بايت** = مطابقة لـ `{"success":false,"error":{"code":"INTERNAL_ERROR","message":"failed to scan lead"}}`.
- أي أن الخطأ وقع في `rows.Scan` عند سطر 248.

**السبب الجذري المرجّح (بنمط مطابق تماماً لـ 2.2):**

الاستعلام يقرأ أعمدة **Nullable** من `crm_leads` إلى حقول `string` في نفس النموذج **دون `COALESCE`**:

| العمود | النوع | الحقل | nullable |
| ------ | ------ | ------ | -------- |
| `partner_name` | VARCHAR | `PartnerName` | نعم |
| `contact_name` | VARCHAR | `ContactName` | نعم |
| `email_from` | CITEXT | `EmailFrom` | نعم |
| `phone` | VARCHAR | `Phone` | نعم |
| `source` | VARCHAR | `Source` | نعم |
| `lost_feedback` | TEXT | `LostFeedback` | نعم |
| `notes` | TEXT | `Notes` | نعم |

أي صف تحمل `NULL` في أيٍّ من هذه الأعمدة → pgx: `converting NULL to string is unsupported` → 500. (استعلام `crm_leads` لم يشمل إصلاح COALESCE في العمل الجاري كما حصل مع المراحل.)

**الإصلاح المقترح:**

- إحاطة الأعمدة nullable بـ `COALESCE(..., '')` في استعلام `ListLeads` (والاستعلامات المشابهة `GetLead`, `ListLeadsStat`, ...)، **أو** مسحها إلى `*string`, ثم تقليص قيمة فارغة إلى `""`.
- إضافة صف guarantee في الاختبارات لدالة `ListLeads` بكل قيم NULL.

---

### 2.5 أخطاء `400` على `/api/v1/account.budget` (مرتان — 05:52) — **مفتوحة/غير مؤكدة**

**الموقع:** لا يوجد Route يطابق هذا المسار في الكود الحالي.

- حجم الجسم = **88 بايت**. بتحليل الأطوال مع تنسيق `response.Error`، التطابق الأمثل هو غلاف Structured:
  - `{"success":false,"error":{"code":"VALIDATION_ERROR","message":"<~21 حرفاً>"}}` (مثل `invalid filter terms` / `invalid query string`)، أو
  - `{"success":false,"error":{"code":"BAD_REQUEST","message":"<26 حرفاً>"}}`.
- أي أن الطلب **التُقط فعلًا من Handler** يعيد 400 Validation (وليس 404 من chi)، لكن:

  - في شجرة العمل الحالية **لاRoute** يطابق `GET /api/v1/account.budget` (بحث شامل في `internal/adapters/http/**/routes.go`).
  - لا يوجد تنفيذ لميزة Budget إطلاقاً في المشروع (الاسم هو نموذج Odoo «account.budget» ولا يظهر إلا في ملفات i18n).
  - قبلها بساعات (04:11، تشغيل سابق) نفس الطرف عاود نفس المسار مع شرطة مائلة زائدة فحصل على **404** عادياً.

  **القراءة الأرجح:** الطلب صدر من واجهة تتوقع ميزة Budget بنمط Odoo، وتم الرد من **نسخة build مختلفة عن الكود الحالي** (أو من نقطة نهاية عامة كانت موجودة سابقاً ثم أُزيلت)، وينتج عنها 400 بدل أن يكون 404. لا يمكن تأكيد الرسالة الفعلية من السجلات لأنها لا تحوي Details.

**الإجراء المقترح:**

- إعادة الإنتاج على النسخة الجارية الحالية (بعد النشر) مع تفعيل تسجيل تفاصيل الخطأ (القسم 3) لقراءة الرسالة الحرفية.
- القرار: إما تنفيذ ميزة الميزانيات فعليًا (مثلاً `GET /api/v1/accounting/budgets`)، أو إزالة الاستدعاء من الواجهة نهائياً.
- إن كانت الواجهة تبني المسارات ديناميكياً من أسماء نماذج Odoo، فلا بد من إضافة قائمة النماذج المدعومة لديها.

---

### 2.6 رفض الصلاحيات 403 (4 مرات) — **جزئية**

| الوقت | الطلب | السبب |
| ------- | ------ | ------- |
| 02:53:54 / 02:53:55 | `POST /api/v1/moves/3/cancel` | `authorization.denied` على `account.move` (action=write) |
| 04:10:25 / 04:10:28 | `GET /api/v1/bank-statements/` | `authorization.denied` على `account.bank.statement` (action=read) |

- **حالة:** نموذج ban statements مُعالَج جزئياً — أُضيف ترحيل `000070_seed_bank_statement_acl` (غير مكرّس) يزرع صلاحيات قراءة/كتابة للنموذج.
- **متبقٍ:** `account.move` (cancel) لا يزال يرفض الكتابة للمستخدم `user_id=1`؛ تحقق من مطابقة اسم النموذج في قواعد الأذونات لـ `account.move`, وتأكد من منح إجراء `write` لمستخدم الإدارة.
- ملاحظة: قبل الترحيل 000070 يجب نشر الترحيلات `goose up` في البيئة قبل إعادة الفحص.

---

### 2.7 مسارات غير موجودة 404 (4 مرات)

| الوقت | الطلب | السبب |
| ------- | ------ | ------- |
| 04:11:08 / 04:11:12 | `GET /api/v1/account.budget/` | لا توجد ميزة Budget إطلاقاً |
| 04:11:21 | `GET /api/v1/partners/suppliers/` | شرطة مائلة زائدة — المسار الصحيح بلا `/` |
| 05:31:30 | `GET /nonexistent-endpoint` | اختبار/فحص قصير |

**حالة الإصلاح (في شجرة العمل):** `router.go` أضاف `middleware.StripSlashes` فتُقبَل المسارات المائلة الزائدة (`suppliers/` → `suppliers`). يبقى `account.budget` قضية مستقلة (انظر 2.5).

---

### 2.8 غير مصرّح 401 (مرة واحدة)

- `GET /api/v1/reports/dashboard` في 02:42:55 بدون توكن صالح/منتهي — سجل WARN طبيعي ولا يحتاج إجراء.

---

## 3) فجوة المراقبة (Observability) — **مُعالَجة في العمل الجاري**

في النسخة التي أنتجت الأحداث: `structuredLogger` (`router.go:280`) كان يسجّل فقط `method, path, status, bytes, duration_ms, request_id` **بلا رسالة الخطأ الداخلية ولا Details**، لذلك استنتجنا Causes عبر مطابقة عدد البايتات بجسم الخطأ.

**حالة الإصلاح (في شجرة العمل):**

- `response.go` أضاف `trackError/TakeError`: عند استجابة ≥500 يلتقط جذر الخطأ (message + wrapped error + details) ويربطه بنفس `request_id` في سجل الـ access.
- `router.go` يضيف حقل `error` إلى سجل الـ500، مع سجل منفصل `internal server error response`.
- يوجد اختبار `response_log_test.go` يتحقق من ظهور `request_id`, `INTERNAL_ERROR`, السبب الجذري معاً.

هذا الإصلاح سيجعل أي تحليل مستقبلي مشابه فورياً دون استنتاج بالأحجام.

**ملاحظة ثانوية:** رسالة السجل `authorization.allowed` تظهر أيضاً للأحداث المرفوضة (نفس الاسم بمستوى INFO مع `allowed:false`) — يُفضَّل فصلها إلى `authorization.denied` بمستوى WARN/ERROR لتسهيل التنبيه على محاولات الوصول المرفوضة (لم يُعالَج بعد).

---

## 4) الجدول الزمني للتشغيلات (3 عملية)

| العملية (request_id prefix) | الفترة | الأحداث |
| ------ | ------ | ------ |
| `8F2XGEHU6S` | 2026-09-14 05:31–05:33 | 404 اختباري، 378× 503 على `/readyz` أثناء drain |
| `FAWds8FucL` | 2026-09-16 02:41–04:44 | 401، 403×2 (moves)، 500×8 (stream)، 500×4 (stages)، 403×2 (bank-statements)، 404×3 |
| `4fx0G3dx4L` | 2026-09-16 05:16– | 500×4 (`/leads/`)، 400×2 (`account.budget`) |

---

## 5) ملخص الأولويات

| # | الإجراء | الحالة | الأولوية |
| --- | --------- | ------ | ---------- |
| 1 | إصلاح `ListLeads` (وما يشابهه) بحماية الأعمدة nullable بـ `COALESCE` أو `*string` | **مفتوحة** | عالية |
| 2 | التحقق من `400 /account.budget` بعد النشر + تفعيل تسجيل التفاصيل؛ تنفيذ الميزة أو إزالة النداء من الواجهة | **مفتوحة/غير مؤكدة** | متوسطة |
| 3 | نشر إصلاح SSE (`http.NewResponseController` + `metrics.Flush`) وإعادة فحص توقف 500 على `/notifications/stream` | منجَز WIP — يحتاج نشراً | عالية |
| 4 | نشر تسجيل تفاصيل الـ 500 (message/details + request_id) | منجَز WIP — يحتاج نشراً | عالية |
| 5 | إكمال RBAC: `account.move.write` (cancel)، ونشر 000070 لأذونات bank-statement | جزئية (000070 في WIP) | متوسطة |
| 6 | خفض ضجيج `503 /readyz` إلى Debug وتقنين الـ poller | منجَز WIP | منخفضة |
| 7 | نشر `StripSlashes` للمسارات المائلة الزائدة | منجَز WIP | منخفضة |

---

*ملاحظة:* الأرقام مبنية على آخر البيانات المتاحة (حتى 2026-09-16 05:52). السجلات لا تظهر بعدُ سوى سجل الـ access لكل طلب. يُنصح بإعادة الفحص بعد نشر نسخة العمل الجاري للتحقق الفوري من إصلاح (3, 4) ومن استمرار (1, 2, 5).
