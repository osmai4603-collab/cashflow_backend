# خطة تنفيذ المرحلة 17: إدارة المشاريع

## 1. الهدف

تنفيذ إدارة المشاريع والمهام في `cashflow_backend` مع مراحل Kanban، المهام الفرعية، التبعيات، التعيين، المعالم، الوسوم، العزل متعدد الشركات، والصلاحيات، مع الاستفادة من سلوك Odoo 19 دون نسخ المكونات الخارجة عن نطاق الإصدار الأول.

المصدر المرجعي المحلي:

`/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/project`

## 2. النطاق المعتمد

### داخل المرحلة

- المشاريع ومراحل المشاريع.
- مراحل المهام وKanban.
- المهام والمهام الفرعية.
- تعيين مستخدم أو أكثر للمهمة.
- تبعيات المهام داخل المشروع.
- حالات المهام والانتقال بينها.
- المعالم وتحقيقها.
- وسوم المهام.
- ربط المشروع بالشريك والحساب التحليلي والشركة.
- ACL وrecord rules والعزل حسب الشركة.
- واجهات HTTP واختبارات domain/storage/usecase.

### خارج المرحلة

- البريد والمحادثات والتقييمات.
- Portal والمشاركة الخارجية و`privacy_visibility` المتقدم.
- المهام المتكررة والقوالب والأدوار.
- Personal stages.
- Project updates وburndown reports.
- Planning وTimesheets.
- `actual_hours`؛ يدعم الإصدار الأول `allocated_hours` فقط.

## 3. المطابقة مع Odoo 19

| مفهوم الخطة | نموذج/حقل Odoo | قرار التنفيذ |
|---|---|---|
| المشروع | `project.project` | كيان تشغيل مستقل مرتبط بالحساب التحليلي |
| مدير المشروع | `user_id` | يُعرض في Go باسم `ManagerID` عند الحاجة |
| الحساب التحليلي | `account_id` | مرجع إلى `account_analytic_account`، وليس كيانًا جديدًا |
| مرحلة المشروع | `project.project.stage` | منفصلة عن مرحلة المهمة |
| مرحلة المهمة | `project.task.type` | تدعم `sequence`, `fold`, `color`, `active` |
| مسؤولو المهمة | `user_ids` | جدول ربط many-to-many |
| المهمة الفرعية | `parent_id`/`child_ids` | يجب منع الدورات والتحقق من نفس المشروع |
| التبعية | `depend_on_ids`/`dependent_ids` | يجب منع الدورات والتبعيات عبر المشاريع |
| الوقت | `allocated_hours` | لا تُضاف `actual_hours` في هذه المرحلة |
| المعلم | `project.milestone` | لا يتحقق قبل إغلاق المهام المرتبطة |

## 4. نموذج الحالات

القيم الداخلية الثابتة:

- `in_progress`
- `changes_requested`
- `approved`
- `waiting`
- `done`
- `cancelled`

تُصبح المهمة `waiting` عند وجود تبعية مفتوحة، وتعود إلى حالة مفتوحة بعد إغلاق التبعيات وفق قاعدة domain/usecase. لا تُترك هذه القاعدة للـ HTTP handler.

## 5. خطة التنفيذ

### المرحلة 17.1: العقد والنموذج

- إنشاء `internal/domain/project/`.
- تعريف `Project`, `ProjectStage`, `TaskStage`, `Task`, `Milestone`, `TaskTag`.
- تعريف enums وقواعد التحقق.
- منع self-reference والدورات منطقيًا.
- إضافة اختبارات domain للحالات والعلاقات.

### المرحلة 17.2: قاعدة البيانات

- إضافة migration جديدة برقم `000019` بعد حجز `000018` لمخطط stock-account وإزالة النسخة المكررة من `000012`.
- إنشاء جداول المشاريع والمراحل والمهام والربط والتبعيات والمعالم والوسوم.
- إضافة المفاتيح الخارجية والفهارس وقيود uniqueness.
- استخدام `RESTRICT` للمراحل المستخدمة، و`CASCADE` للعناصر التابعة، و`SET NULL` للعلاقات الاختيارية.
- إضافة down migration قابلة للتراجع.

### المرحلة 17.3: التخزين

- إنشاء memory repository أولًا.
- إنشاء PostgreSQL repository.
- دعم CRUD والبحث حسب الشركة والمشروع والحالة والمسؤول.
- دعم Kanban وsubtasks وdependencies وmilestones وtags.
- جعل تعديل حواف التبعيات ذريًا في PostgreSQL.

### المرحلة 17.4: حالات الاستخدام

- تنفيذ use cases للمشاريع والمراحل والمهام والمعالم والوسوم.
- إضافة عمليات صريحة لنقل المهمة، التعيين، إدارة المهام الفرعية، إدارة التبعيات، وتحقيق المعلم.
- التحقق من أعلام المشروع:
  - `AllowSubtasks`
  - `AllowDependencies`
  - `AllowMilestones`
- إعادة حساب حالة المهمة بعد تغييرات التبعيات.

### المرحلة 17.5: HTTP API

- إنشاء `internal/adapters/http/project/`.
- تسجيل:
  - `CRUD /api/v1/projects`
  - `CRUD /api/v1/project-stages`
  - `CRUD /api/v1/task-stages`
  - `CRUD /api/v1/tasks`
  - `GET /api/v1/projects/{id}/tasks`
  - `GET /api/v1/projects/{id}/tasks/kanban`
  - `GET /api/v1/tasks/my`
  - `PUT /api/v1/tasks/{id}/stage`
  - `PUT /api/v1/tasks/{id}/assign`
  - `GET /api/v1/tasks/{id}/subtasks`
  - `CRUD /api/v1/projects/{id}/milestones`
  - `POST /api/v1/milestones/{id}/reach`
  - `CRUD /api/v1/task-tags`
- اتباع نمط CRM في DTOs والاستجابات والأخطاء والتصفية.

### المرحلة 17.6: الدمج والصلاحيات

- إضافة wiring في `cmd/server/main.go`.
- تمرير handler إلى `internal/adapters/http/router.go`.
- إضافة ACL وrecord rules للنماذج الجديدة.
- منع الوصول إلى سجلات شركة أخرى.
- جعل مدير المشروع مسؤولًا عن CRUD للمشاريع والمراحل والمعالم والوسوم، ومستخدم المشروع قادرًا على إدارة المهام المسموح بها.

### المرحلة 17.7: الاختبارات والتحقق

- Domain: الدورات، التوافق، الحالات، وأعلام المشروع.
- Repository: CRUD، العزل، Kanban، والعلاقات.
- Usecase: التبعيات، التعيين، المعالم، والصلاحيات.
- HTTP: CRUD وKanban والعمليات المتخصصة.
- PostgreSQL: migration up/down والقيود.
- Auth: ACL وrecord rules وmulticompany.
- تشغيل `go test ./...` و`go vet ./...`، مع فصل أخطاء baseline عن أخطاء المرحلة.

## 6. معايير القبول

- لا يمكن إنشاء تبعية ذاتية أو دورة تبعيات.
- لا يمكن ربط مهمة أو معلم بمشروع مختلف.
- لا يمكن ربط سجلات من شركة أخرى.
- التبعية المفتوحة تجعل المهمة في حالة `waiting`.
- إغلاق التبعيات يعيد المهمة إلى الحالة المفتوحة المناسبة.
- لا يمكن تحقيق معلم يحتوي مهامًا مفتوحة.
- لا يمكن استخدام milestone أو dependency عندما يكون الخيار الخاص بها معطلًا في المشروع.
- تعمل migrations صعودًا وتراجعًا.
- تمر اختبارات الحزمة الجديدة، مع توثيق أي فشل سابق في المشروع.

## 7. المخاطر والقرارات

- migration Project أصبحت `000019` وتم اختبارها حيًا مع أداة الترحيل.
- يجب مطابقة أسماء جداول المستخدمين والشركات والشركاء قبل إضافة المفاتيح الخارجية.
- Odoo مرجع سلوكي، وليس عقد API حرفيًا.
- دعم `portal`, mail, planning, recurrence, timesheets والتقارير يؤجل إلى مراحل مستقلة.

## 8. حالة التنفيذ

تم البدء فعليًا بالشرائح التالية:

- `internal/domain/project/`: نماذج المشاريع والمراحل والمهام والمعالم والوسوم، الحالات، والتحقق الأساسي.
- `internal/domain/project/ports.go`: عقد repository ومرشحات المهام وبيانات Kanban.
- `internal/adapters/storage/project/memory_repo.go`: تنفيذ memory للعزل حسب الشركة، CRUD، Kanban، التعيين، الوسوم، التبعيات، والمعالم.
- `internal/adapters/storage/project/postgres_repo.go`: تنفيذ PostgreSQL للمشاريع والمهام والمراحل والمعالم والوسوم وKanban والتبعيات والتعيين.
- `internal/adapters/storage/project/memory_repo_test.go`: اختبار حدود الشركات والمشاريع ومنع دورات التبعيات.
- `internal/usecase/project/project_usecase.go`: قواعد أعلام المشروع، دورة حالة التبعية، وإنجاز المعالم.
- `internal/adapters/http/project/`: DTOs وhandler وroutes أولية للمشاريع والمهام والتبعيات.
- `internal/usecase/project/project_usecase_test.go` و`internal/adapters/http/project/handler_test.go`: اختبارات دورة التبعية واستخراج الشركة من JWT.
- `migrations/000019_create_project_schema.up.sql` و`000019_create_project_schema.down.sql`: مخطط قاعدة البيانات والـ rollback.

التحقق المنجز:

- اختبارات domain: ناجحة.
- اختبار memory repository: ناجح.
- اختبارات Use Case وHTTP: ناجحة.
- فحص أخطاء ملفات domain وmemory repository: بلا أخطاء.

تم ربط `projectHandler` و`projectRepo` في composition root والـ router، وإضافة ACL للموديلات في migration `000019`. عمليات PostgreSQL وHTTP الأساسية مكتملة، وتم اختبار migration up/down حيًا بنجاح.

المتبقي بالترتيب: استكمال PostgreSQL CRUD لمراحل المشاريع ومراحل المهام والمعالم والوسوم، ثم توسيع HTTP إلى CRUD الكامل واختبارات PostgreSQL التكاملية.

ملاحظة: محرك record rules الحالي يعتمد على Go AST (`internal/platform/auth`) وليس على JSON مخزن في migrations. عزل Project حسب الشركة مطبق داخل repositories وUse Cases وHTTP، بينما لا توجد بعد واجهة عامة في `Authorizer` لفحص سجل منفرد قبل العملية.

## 9. نتيجة التحقق الأخيرة

- اختبارات project المركزة: ناجحة، وتشمل domain وmemory repository وUse Case وHTTP.
- `go test ./...`: ناجح بالكامل.
- تم الحفاظ على توافق مستدعي `NewRouter` القديم، وتمرير project handler كخيار اختياري.
- اكتملت عمليات update/delete HTTP للمشاريع والمهام والمراحل والمعالم والوسوم، مع endpoint تحقيق المعلم.
- تم التحقق حيًا من `migrate up` حتى النسخة `000019`، ثم `migrate down` وإعادة `migrate up` بنجاح.
