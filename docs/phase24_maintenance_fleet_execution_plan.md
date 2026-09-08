# خطة تنفيذية — المرحلة 24: الصيانة وإدارة الأسطول (Maintenance & Fleet)

## 1. الخلاصة التنفيذية

تنفيذ نظام **الصيانة (Maintenance)** وإدارة **أسطول المركبات (Fleet)** على غرار موديولي
`addons/maintenance/` و `addons/fleet/` في Odoo 19.0، مع الالتزام بمعمارية Clean Architecture
المستخدمة في المشروع والتكامل مع:
- **المرحلة 9 (HR)**: ربط المعدات وطلبات الصيانة بالموظفين (`hr_employees`) والأقسام (`hr_departments`).
- **المرحلة 6 (المخزون)**: ربط المعدات بمواقع المخزون (`stock_locations`) والأرقام التسلسلية.
- **المرحلة 15 (الأنشطة)**: إنشاء أنشطة/إشعارات للفنيين عند اقتراب الصيانة الوقائية ولتجديد العقود.

**الوضع الحالي**: ✅ **مكتملة** — تم بناء طبقات Ports / Storage (postgres + memory) /
Usecase / HTTP / Wiring / Worker / الاختبارات فوق السكيلد القائم
(`internal/domain/{maintenance,fleet}`, migration `000026`) خلال هذه الجلسة.

**الخلاصة**: اكتمل البناء على النمط المعتمد في موديول `project` (المرحلة 17)
مع Migration تكميلي `000038` لمواءمة الفجوات مقابل Odoo 19.0 و `000039` لبذور ACL،
وعاملين خلفيين (`maintenance` الوقائية + انتهاء عقود الأسطول).

---

## 2. حالة التنفيذ الحالية (Audit)

### 2.1 الموجود (بدأ تنفيذه — تمت المراجعة في 2026-09-08)

| الطبقة | الملف | الحالة |
|---|---|---|
| Domain | `internal/domain/maintenance/maintenance.go` | ✅ موجود — كيانات + `Validate()` |
| Domain | `internal/domain/fleet/fleet.go` | ✅ موجود — كيانات + `Validate()` |
| Migration | `migrations/000026_create_maintenance_fleet_schema.{up,down}.sql` | ✅ منشور — 12 جدولاً |
| Scaffold | `internal/usecase/{maintenance,fleet}/` | ⬜ مجلدات فارغة |
| Scaffold | `internal/adapters/http/{maintenance,fleet}/` | ⬜ مجلدات فارغة |
| Scaffold | `internal/adapters/storage/{maintenance,fleet}/` | ⬜ مجلدات فارغة |

### 2.2 الناقص (يتم بناؤه في هذه الخطة)

| الطبقة | الملف |
|---|---|
| Domain | `ports.go` لكلا الحزمتين + توسعة الكيانات لمواءمة Odoo 19.0 |
| Storage | `postgres_repo.go` + `memory_repo.go` لكل موديول |
| Usecase | `maintenance_usecase.go` + `fleet_usecase.go` (+ اختبارات) |
| HTTP | `handler.go` + `dto.go` + `routes.go` لكل موديول (+ اختبارات) |
| Wiring | `repositories.go` / `usercases.go` / `handlers.go` / `router.go` |
| Migration | `000038_align_maintenance_fleet_odoo19` + `000039_seed_maintenance_fleet_acl` |
| Worker | `internal/infrastructure/worker/maintenance_fleet_worker.go` |
| Tests | اختبارات domain/usecase/http/storage |

> [!NOTE]
> موديول `expense` في نفس الحالة الجزئية (Domain + migration فقط) كمرجع موازٍ —
> لكن هذه الخطة تُكمل `maintenance` و `fleet` بالكامل.

---

## 3. مراجعة الفجوات مقابل Odoo 19.0 (Gap Analysis)

المصدر: `/home/osm/Downloads/odoo-19.0/odoo-19.0/addons/{maintenance,fleet,hr_maintenance,stock_maintenance,fleet_vehicle}/models/`

| # | الفجوة | الوضع في المشروع | القرار |
|---|---|---|---|
| 1 | `hr_maintenance`: `employee_id`/`department_id` + `assign_to` على المعدات والطلبات | ✅ موجود في `000026` | الاحتفاظ |
| 2 | `stock_maintenance`: `location_id` + `serial_no` | ✅ موجود في `000026` | الاحتفاظ |
| 3 | `fleet_vehicle_assignation_log` مع تحقق التداخل | جدول موجود — **بلا تحقق تداخل** | إضافة قاعدة أعمال (+ انحراف مقصود عن Odoo لأنه مطلوب في خطة التحقق) |
| 4 | `fleet_vehicle_tag` + `fleet_vehicle_model_category` | ✅ جداول موجودة | إضافة M2M `fleet_vehicle_vehicle_tag_rel` |
| 5 | `maintenance.team` (فرق الصيانة + الأعضاء) | ⬜ غير موجود | إضافة في `000038` |
| 6 | `maintenance.stage` بذور (New/In Progress/Repaired/Scrap) | جدول موجود — بلا بذور | بذر في `000038` |
| 7 | الحقول المتكررة للصيانة الوقائية `repeat_*`/`kanban_state`/`schedule_end` (Odoo 19.0 حذف `maintenance.plan`) | ⬜ غير موجود | إضافة في `000038` + التفرع التلقائي عند الإغلاق |
| 8 | `fleet.service.type` (تصنيف الخدمات/العقود) | ⬜ غير موجود | إضافة في `000038` |
| 9 | حالة المركبة كسجلات `fleet.vehicle.state` (`state_id` FK) | ⬜ `VARCHAR` حالياً | **حسب Odoo بصرامة**: جدول `fleet_vehicle_states` + `state_id` (إبقاء `state` للتوافق المؤقت) |
| 10 | نسخ خصائص الموديل إلى المركبة (fuel_type, seats, ...) | ⬜ | منطق `_load_fields_from_model` في الـ Usecase |
| 11 | عداد تنازلي محظور + إنشاء سجل odometer تلقائياً عند الضبط | ⬜ | قاعدة في الـ Usecase |
| 12 | تقرير التكلفة الشهرية لكل مركبة (`fleet.vehicle.cost.report`) | ⬜ | حساب Usecase (خدمات + عقود شهرية/سنوية/يومية) |
| 13 | حالة العقد التلقائية + التذكير (`days_left`، تذكير تجديد) | ⬜ | حساب عند القراءة + Worker |
| 14 | `vehicle.state` (تعريف موديل متقدم) | ⬜ | ثوابت + اختزانه حسب الـ selection |

> [!IMPORTANT]
> التحقق رقم 3 تم إجراؤه على ملفات الـ migrations كاملة (حتى `000037`): لا يوجد أي من
> `fleet_vehicle_states`, `maintenance_teams`, `fleet_service_types`, `fleet_vehicle_vehicle_tag_rel`,
> `repeat_*`, `kanban_state`, `schedule_end` — وبالتالي لا يوجد تكرار.

---

## 4. القرارات التصميمية

| القرار | الاختيار | المبرر |
|---|---|---|
| النقود | `float64` في Go + `NUMERIC(20,4)` في SQL | توافق مع قاعدة المشروع (لا إضافة مكتبة decimal) |
| حالة المركبة | سجلات `fleet_vehicle_states` + `state_id` FK مع إبقاء `state` للتوافق | التزام صارم بسلوك Odoo (طلب المستخدم) |
| أولوية الطلب | Selection `"0".."3"` (varchar + ثوابت) | مثل `maintenance.request.priority` في Odoo |
| مراحل الصيانة | جدول `maintenance_stages` كسجلات Kanban (بتسبة `done`) | مثل Odoo (لا State Selection) |
| الصيانة الوقائية | حقول `repeat_*` على الطلب + نسخة تلقائية عند الإغلاق + نشاط تذكير | سلوك Odoo 19.0 (حذف `maintenance.plan`) |
| تحقق تداخل السائقين | رفض فتح تعيين متزامن لنفس المركبة أو نفس السائق | مطلوب في خطة التحقق (انحراف مقصود عن Odoo) |
| العمال الخلفيون | `maintenance_fleet_worker` يقلّد `reorder_worker` | تكرار الوقائية + انتهاء العقود |
| الترقيم | `000038` ثم `000039` | أحدث migration هو `000037` |

---

## 5. نطاق العمل (Work Items)

### 5.1 هيكل الملفات المستهدف

```
internal/domain/maintenance/
├── maintenance.go         # توسعة: Equipment + MaintenanceRequest (TeamID, Repeat*, state)
├── team.go                # NEW: MaintenanceTeam
├── ports.go               # NEW: Repository + Filters
internal/usecase/maintenance/
├── maintenance_usecase.go # NEW
├── maintenance_usecase_test.go
internal/adapters/http/maintenance/
├── handler.go / dto.go / routes.go
├── handler_test.go
internal/adapters/storage/maintenance/
├── postgres_repo.go
├── memory_repo.go
│
internal/domain/fleet/
├── fleet.go               # توسعة: Vehicle (TagIDs, StateID/State), ServiceLogService, Contract
├── service_type.go        # NEW: ServiceType
├── ports.go               # NEW
internal/usecase/fleet/
├── fleet_usecase.go
├── fleet_usecase_test.go
internal/adapters/http/fleet/
├── handler.go / dto.go / routes.go
├── handler_test.go
internal/adapters/storage/fleet/
├── postgres_repo.go
├── memory_repo.go

migrations/
├── 000038_align_maintenance_fleet_odoo19.{up,down}.sql
├── 000039_seed_maintenance_fleet_acl.{up,down}.sql

internal/infrastructure/worker/
├── maintenance_fleet_worker.go  # NEW
```

### 5.2 توسعة Domain — Maintenance

```go
type MaintenanceTeam struct { ID int64; Name string; Color int; Active bool; CompanyID int64; MemberIDs []int64; Audit audit.Fields }

// Equipment — إضافات Odoo 19.0
TeamID, PartnerID (*int64), PartnerRef, Cost, Note, AssignDate (*time.Time), ScrapDate (*time.Time)

// MaintenanceRequest — إضافات Odoo 19.0
TeamID (*int64), KanbanState (normal|blocked|done), ScheduleEnd (*time.Time),
Recurring bool, RepeatInterval int, RepeatUnit (day|week|month|year),
RepeatType (forever|until), RepeatUntil (*time.Time)
```

### 5.3 توسعة Domain — Fleet

```go
type ServiceType struct { ID int64; Name string; Category string /* contract|service */; CreatedAt time.Time }

// Vehicle — إضافات
ManagerID (*int64), TagIDs []int64, StateID (*int64)  // + ثوابت الحالة الجديدة (new_request, active, inactive)

// VehicleLogService — إضافات
ServiceTypeID (*int64), InvRef string, State string // new|running|done|cancelled

// VehicleLogContract — إضافات
UserID (*int64), Date (*time.Time), Name string, DaysLeft int (computed), ExpiresToday bool (computed)
```

### 5.4 Ports — ملخص الواجهات

`maintenance.Repository`: CRUD لـ EquipmentCategory, EquipmentStage, MaintenanceTeam, Equipment, MaintenanceRequest
+ `ListMaintenanceRequests(companyID, filter)` مع فلاتر (EquipmentID, StageID, Type, Priority) + `CountByStage` للـ dashboard.

`fleet.Repository`: CRUD لـ VehicleBrand, VehicleModelCategory, VehicleModel, VehicleTag, ServiceType, Vehicle,
+ `CreateOdometerReading`, `GetOdometerMax`, `ListAssignations`, `CreateAssignation`, `CountOpenAssignations`,
CRUD خدمات/عقود، + `GetVehicleWithModel`, `CostReport(companyID, vehicleID, from, to)`.

### 5.5 Usecase — منطق الأعمال

**Maintenance**
- `CreateEquipment`: تعيين `NextActionDate` من `EffectiveDate + Period` عند الإنشاء.
- `CreateMaintenanceRequest`: المرحلة الافتراضية = أول مرحلة Sequence؛ `RequestDate=now`؛ إذا كانت وقائية بمجدول → إنشاء نشاط للفني (المرحلة 15).
- `CloseMaintenanceRequest`: تعيين `CloseDate`، الانتقال لمرحلة `Repaired` (done=true)؛
  إذا كانت **وقائية ومتكررة** → إنشاء نسخة جديدة بـ `ScheduleDate + RepeatInterval` (حسب `RepeatUnit`) إن لم يتجاوز `RepeatUntil`.
- `Dashboard`: توزيع الطلبات حسب المرحلة/النوع/الأولوية + عدد المتأخرة.

**Fleet**
- `CreateVehicle`: توليد `Name` (`brand/model/plate`) ونسخ خصائص الموديل (fuel_type, seats, doors, ...).
- `UpdateVehicle` / Driver change: إغلاق التعيين المفتوح وإنشاء سجل `AssignationLog`
  **مع تحقق التداخل** (لا تعيينان متزامنان لنفس المركبة، ولا تعيين للسائق بينما له تعيين آخر مفتوح).
- `SetOdometer`: رفض قيمة < أقصى قيمة مسجلة؛ إنشاء سجل odometer.
- `CreateServiceLog`: إنشاء قراءة odometer إن وُجدت القيمة (+ تفعيل `ServiceType`).
- `CloseContract`/حساب الحالة: تحويل تلقائي `futur → open → expired` حسب التاريخ + `DaysLeft/ExpiresToday`.
- `CostReport`: تجميع شهري للخدمات (غير الملغاة) + العقود (تكلفة تفعيل لمرة واحدة + النسبية اليومية/الشهرية/السنوية).

### 5.6 HTTP — Endpoints و ACL Models

**Maintenance** (`access(routes, model, action)` بنمط `project`)

```
POST/GET/GET{id}/PUT{id}/DELETE{id}  /api/v1/equipment                  maintenance.equipment
CRUD                                  /api/v1/equipment-categories       maintenance.equipment.category
CRUD                                  /api/v1/maintenance-stages         maintenance.stage
CRUD                                  /api/v1/maintenance-teams          maintenance.team
CRUD                                  /api/v1/maintenance-requests       maintenance.request
POST /start  /close                   /api/v1/maintenance-requests/{id}  maintenance.request (write)
GET                                   /api/v1/maintenance-requests/dashboard  (read)
```

**Fleet** (`fleet.vehicle.*` و `fleet.service.type`)

```
CRUD  /api/v1/vehicle-brands            fleet.vehicle.model.brand
CRUD  /api/v1/vehicle-model-categories  fleet.vehicle.model.category
CRUD  /api/v1/vehicle-models            fleet.vehicle.model
CRUD  /api/v1/vehicle-tags              fleet.vehicle.tag
CRUD  /api/v1/service-types             fleet.service.type
CRUD  /api/v1/vehicles                  fleet.vehicle
POST  /api/v1/vehicles/{id}/odometer    fleet.vehicle.odometer
CRUD  /api/v1/vehicles/{id}/services    fleet.vehicle.log.services
GET   /api/v1/vehicles/{id}/cost-report fleet.vehicle (read)
POST  /api/v1/vehicles/{id}/assignations
GET   /api/v1/vehicles/{id}/assignations
CRUD  /api/v1/vehicles/{id}/contracts   fleet.vehicle.log.contract
```

### 5.7 Migration التكميلي `000038` (up — ملخص)

1. `CREATE TABLE maintenance_teams` + `maintenance_team_members` (rel).
2. `ALTER TABLE maintenance_equipment ADD team_id, partner_id, partner_ref, cost, notes, assign_date, scrap_date` (FK حيث يلزم).
3. `ALTER TABLE maintenance_requests ADD team_id, kanban_state, schedule_end, recurring_maintenance, repeat_interval, repeat_unit, repeat_type, repeat_until, archived`.
4. `CREATE TABLE fleet_vehicle_states` + بذر السجلات (New Request, To Order, Ordered, Registered, Downgraded, Reserve, Waiting List).
5. `ALTER TABLE fleet_vehicles ADD state_id, manager_id`.
6. `CREATE TABLE fleet_vehicle_vehicle_tag_rel` (M2M) + `ALTER TABLE fleet_vehicles` (لا عمود إضافي — علاقة رابطة).
7. `CREATE TABLE fleet_service_types` + بذر (خدمات: Repair and maintenance؛ عقود: Omnium, Leasing).
8. `ALTER TABLE fleet_vehicle_log_services ADD service_type_id, state, inv_ref`.
9. `ALTER TABLE fleet_vehicle_log_contracts ADD user_id, date, name`.
10. بذر `maintenance_stages` (New Request / In Progress / Repaired / Scrap — مع `done` للمراحل النهائية).

### 5.8 ACL `000039` (نمط `000031_seed_purchase_requisition_acl`)

| model | group 1 (internal) | group 2 (admin) |
|---|---|---|
| `maintenance.equipment`, `maintenance.request`, `maintenance.equipment.category`, `maintenance.stage`, `maintenance.team` | read+create+update | + delete |
| `fleet.vehicle`, `fleet.vehicle.model`, `fleet.vehicle.model.brand`, `fleet.vehicle.model.category`, `fleet.vehicle.tag`, `fleet.service.type`, `fleet.vehicle.odometer`, `fleet.vehicle.log.services`, `fleet.vehicle.log.contract` | read+create+update | + delete |

### 5.9 نقاط التركيب (Wiring)

| الملف | التعديل |
|---|---|
| `internal/adapters/storage/repositories.go` | حقول `Maintenance maintenance.Repository` و `Fleet fleet.Repository` + إنشاء postgres/memory |
| `internal/usecase/usercases.go` | حقول `Maintenance` و `Fleet` + constructor مع `repositories.Activity` (لأنشطة المرحلة 15) |
| `internal/adapters/http/handlers.go` | حقول + `NewHandler` |
| `internal/adapters/http/router.go` | `maintenancehttp.RegisterRoutes(v1, ...)` و `fleethttp.RegisterRoutes(v1, ...)` |
| `cmd/server/main.go` | `wm.Start("maintenance-fleet-watcher", ...)`.Run() |

### 5.10 Worker — `maintenance_fleet_worker.go`

نسخة من `ReorderWorker` بمهامتين بمعدل يومي:
1. **Maintenance pass**: طلبات وقائية نشطة حيث `ScheduleDate <= now` وليست في مرحلة `done` → إنشاء/مزامنة نشاط للفني + إشعار (`activity.Bus.NotifyUser`).
2. **Fleet pass**: عقود مفتوحة تنتهي خلال `delay_alert_contract` (افتراضي 30 يوم) → إنشاء نشاط "تجديد عقد" لمستخدم المركبة؛
   والعقود المنتهية تاريخ `تحديث حالتها` إلى `expired`.

---

## 6. Verification Plan

### 6.1 اختبارات آلية (Unit — تسلسل `make test` و `make lint`)

| # | الاختبار | الموقع |
|---|---|---|
| 1 | إنشاء طلب صيانة لمعدات مرتبطة بموظف (assert `EmployeeID` + المرحلة الافتراضية + نشاط الفني) | `usecase/maintenance/..._test.go` |
| 2 | منع تداخل تواريخ السائقين في `AssignationLog` (سائقان متزامنان لمركبة والسائق بمركبتين) | `usecase/fleet/..._test.go` |
| 3 | حساب تكاليف الخدمات (Service Logs) لكل مركبة في تقرير التكلفة | `usecase/fleet/..._test.go` |
| 4 | عداد تنازلي محظور + قراءة odometer تنشئ سجلاً | `usecase/fleet/..._test.go` |
| 5 | التفرع التلقائي للطلب الوقائي المتكرر عند الإغلاق | `usecase/maintenance/..._test.go` |
| 6 | تحول حالة العقد التلقائي + `days_left/expires_today` | `usecase/fleet/..._test.go` |

اختبارات الحالة: Domain `Validate`, storage `memory_repo_test`, HTTP `handler_test` بنمط
`auth.WithClaims(ctx, &auth.UserClaims{CompanyID: n})`.

### 6.2 تحقق يدوي

- **الربط**: `GET /api/v1/equipment` تُظهر المعدة المرتبطة بموقع مخزون محدد (`location_id` → `stock_locations`).
- **التنبيه**: طلب صيانة وقائية بمجدول قريب → تظهر نشاطاً للفني عبر `GET /api/v1/activities` (المرحلة 15) وعبر الـ Worker.
- **التقرير**: `GET /api/v1/vehicles/{id}/cost-report` يعرض الشهور والتكاليف بعد إضافة خدمات وعقود.
- **التداخل**: محاولة تعيين سائقين متزامنين تُرفض بخطأ 409.

```bash
make migrate-up && make run
make test
make lint
```

---

## 7. تقدير الجهد وجدولة المهام

| الخطوة | الوصف | الجهد |
|---|---|---|
| 1 | الخطة التنفيذية (هذا الملف) | — |
| 2 | Migrations `000038`/`000039` | 0.5 يوم |
| 3 | Domain + ports | 1 يوم |
| 4 | Storage (postgres + memory) | 1 يوم |
| 5 | Usecase + أعمال منطقية | 1.5 يوم |
| 6 | HTTP + DTO + Routes | 1 يوم |
| 7 | Wiring + Worker | 0.5 يوم |
| 8 | اختبارات + lint | 0.5 يوم |
| **الإجمالي** | | **~6 أيام** |

---

## 8. المخاطر والافتراضات

- **توافق schema**: الحفاظ على `000026` كما هو وإضافة `000038` فقط (لا تعديل على migrations ملتزمة).
- **الانحراف المقصود**: تحقق تداخل السائقين (غير موجود في Odoo) بناءً على خطة التحقق.
- **`state` المزدوج**: يرافق `state_id` عمود `state` الحالي للتوافق؛ الـ API الجديد يفضّل `state_id`.
- **التنبيهات**: تعتمد على نظام الأنشطة (المرحلة 15) المكتمل — لا إنشاء نظام إشعارات جديد.
- **لا `maintenance.plan`**: استُبدلت بالحقول المتكررة على الطلب (سلوك Odoo 19.0 الصارم).