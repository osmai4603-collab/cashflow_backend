# P0 — جرد خط الأساس لمنظومة المراقبة (Observability Baseline Inventory)

مرجع: [`observability_security_execution_plan.md`](observability_security_execution_plan.md) — المرحلة P0.

> ملاحظة: هذه الوثيقة تُعِدّ **نسخة مخزنة** من السلوك الحالي قبل أي تغيير معماري. أي تعديل لاحق (P1–P8) يجب أن يحافظ على التوافقيات المعلنة هنا أو يوثّق كسرها صراحة.

---

## 1. مسارات المراقبة (بعد عزل P4)

### 1.1 المنفذ العام (`8070`) — بعد التنفيذ

| المسار | الـ Handler | الحالة |
| :--- | :--- | :--- |
| `GET /livez` | `health.HandleLiveness` | ✅ عام (Kubernetes)، ويُتكرر على منفذ الإدارة |
| `GET /readyz` | `health.HandleReadiness` | ✅ عام، مرتبط بالاعتماديات ودورة الـ drain |
| `GET /metrics` | ~~`metrics.Handler`~~ | 🔒 **أُزيل — 404 حالياً** (معزول إلى `8066`) |
| `/debug` (pprof) | ~~`chi.middleware.Profiler()`~~ | 🔒 **أُزيل — 404 حالياً** (معزول إلى `8066`) |

### 1.2 منفذ الإدارة (`8066`) — جديد في P4

نظام مسارات جديد في [`management_router.go`](../internal/adapters/http/management_router.go) يُشغَّل في خادم HTTP مستقل (مرتبط بصورة افتراضية بـ `127.0.0.1:8066` ضمن دورة حياة `server.Run`):

| المسار | الوصف | الحالة الأمنية |
| :--- | :--- | :--- |
| `GET /` | صفحة index تفاعلية بلا assets خارجية | مشروطة بالـ auth (إن فُعّل) |
| `GET /metrics` | نص `text/plain; version=0.0.4` (OpenMetrics) | مشروط بالـ auth |
| `GET /metrics/json` | JSON التوافقي لـ `tool/monitor.sh` | مشروط بالـ auth |
| `GET /livez` / `GET /readyz` | نفس handlers الصحة، خارج حركة الأعمال | مشروط بالـ auth |
| `/debug/pprof/*` | pprof suite — **مفعل فقط عند `management.pprof_enabled=true`** | إغلاق افتراضي |

**قرار المسار المعتمد (منفّذ في P4):**

- العقد النصي النهائي: `GET /metrics` على منفذ الإدارة المستقل `8066` بصيغة OpenMetrics/Prometheus.
- JSON التوافقي: `GET /metrics/json` على نفس منفذ الإدارة لـ `tool/monitor.sh` والتشخيص المحلي.
- pprof: معزول على منفذ الإدارة فقط، ومغلق افتراضياً (`management.pprof_enabled=false`).
- الأمان: `management.interface=0.0.0.0/::/""` مرفوض بدون `management.require_auth=true`؛ و`require_auth=true` يتطلب `auth_token` غير فارغ. التحقق من ShowMe المسارات: `/metrics` يُرجع 200 وتُفحص `Content-Type`، بينما `/metrics` على المنفذ العام 404.

### 1.3 عقد دورة الحياة (P5 منفّذ)

- كل من listener خادم العام والإدارة يُربَط **قبل** الإقلاع وبشكل متزامن في الـ main thread؛ فشل bind في أي خادم = خطأ fatal بلا إعلان readiness (إصلاح الفجوة 3.3) مع إغلاق الـ listener العام إذا سبق ربطه.
- عند الإيقاف: `MarkNotReady` → انتظار مدة الـ drain → `Shutdown` لكلا الخادمين بتايموت موحد، والإغلاق يشمَل خادم الإدارة في كل المسارات (حتى لو فشل العام أو تلقى الإشارة).
- آلية lifecycle واحدة مع `WaitGroup` تمنع double-close/race وتنتظر إنهاء goroutines الـ Serve حفاظاً على "لا goroutines معلّقة".
- سجلات structured: `server_role` (public/management)، `addr`، `shutdown_reason` (signal/server_failure)، `duration`. `SetListener`/`SetManagementListener` مدعومتان للمنافذ الديناميكية في الاختبارات.

### 1.4 المراقبة الحقيقية RUM (P7 منفّذ)

- نقطة `POST /rum` على **منفذ الإدارة** فقط (معطّلة مع `metrics_enabled=false` → 404)، وتخضع لـ `require_auth`؛ الردود: `204`/`400`/`401`/`413`/`415` مع حد جسم `16 KiB`. GET على المسار = `405`.
- خمس عائلات histogram تُسجَّل عبر `metrics.ObserveRUM`: `cashflow_rum_ttfb` و`cashflow_rum_lcp` و`cashflow_rum_inp` (بالثواني — المصدر `ms`) و`cashflow_rum_cls` (بلا وحدة) و`cashflow_rum_dom_interactive`. دلاء الزمن `LatencyBuckets` والـ CLS `CLSBuckets`.
- خصائص أمنية: مجموعة `client_type` مغلقة `web`/`desktop`/`mobile` (معرَّبة)، حقول علوية غير معروفة تُهمل ولا تصبح labels، لا trailing JSON، قيم غير منتهية/سالبة مرفوضة 400. العقد الكامل: [`rum_api_contract.md`](rum_api_contract.md).
- ملاحظة النشر: المُصدر الحقيقي للاختبار الفعلي يُعدّ **ميزة مستقبلية** (نقطة `/rum` عامة بلا مصادقة أو إتاحة منفذ الإدارة للعملاء مع `require_auth`).

### 1.5 مكدس المراقبة والمعايرة (P6 + الخطوات التشغيلية منفّذة)

- تشغيل: `docker-compose.monitoring.yml` (Prometheus + Grafana + **Alertmanager** loopback) على شبكة `cashflow-monitoring-net` المشتركة، scrape لـ `server:8066` عبر Bearer — التفاصيل في [`monitoring_operations.md`](monitoring_operations.md).
- إشعارات: `alertmanager.yml` يوجه إلى webhook `ALERTMANAGER_WEBHOOK_URL`، و`prometheus.yml` فيه `alerting:` نحو `alertmanager:9093`.
- جَمع baseline: `make monitoring-baseline` عبر `tool/calibrate_baseline.sh`، يدعم `STORAGE=memory` و`STORAGE=postgres` (+`BUSINESS_URL`) → `deploy/monitoring/baseline/summary.json`. جلستان بتاريخ 2026-09-12: in-memory (p95=4.8ms، 5xx=0، heap=−2.4MiB) و**postgres حقيقي** (pool peak ratio=0.84، waits≈1950/s).
- التنبيهات مُعالَرة من القياس (`prometheus-alerts.yml`، وسوم `CALIBRATED`/`SLO-DERIVED`): 5xx=0.5%، p95=400ms، heap=128MiB، pool=0.8، waits=50/s.
- فحص صياغة قبل التشغيل: `make monitoring-check` (compose الاثنان + YAML/JSON) دون سحب صور.

---

## 2. جرد مسارات الأعمال (Route Inventory)

### 2.1 الوحدات المثبتة فعلياً في الـ Router (30 وحدة)

المجموع: **507 route handler** تحت `Group("api/v1")` محمية بـ `auth.Middleware(secret)`.

| الوحدة | عدد الـ handlers | مسار الجذر |
| :--- | :--- | :--- |
| company | 6 | `/api/v1/companies` |
| user | 6 | `/api/v1/users` |
| currency | 8 | `/api/v1/currencies` |
| sequence | 6 | `/api/v1/sequences` |
| attachment | 5 | `/api/v1/attachments` |
| activity | 9 | `/api/v1/activities` |
| project | 27 | `/api/v1/projects` |
| bankstatement | 22 | `/api/v1/bank-statements` |
| calendar | 11 | `/api/v1/calendars` |
| partner | 7 | `/api/v1/partners` |
| product | 27 | `/api/v1/products` |
| accounting | 36 | `/api/v1/accounts`, `/api/v1/journals`, ... |
| analytic | 29 | `/api/v1/analytic` |
| sale | 11 | `/api/v1/sale-orders` |
| purchase | 23 | `/api/v1/purchase-orders` |
| stock | 44 | `/api/v1/warehouses`, `/api/v1/stock-pickings`, ... |
| crm | 22 | `/api/v1/leads`, `/api/v1/crm/...` |
| expense | 10 | `/api/v1/expenses` |
| payment | 14 | `/api/v1/payments` |
| pos | 5 | `/api/v1/pos` |
| recruitment | 6 | `/api/v1/recruitments` |
| timesheet | 6 | `/api/v1/timesheets` |
| resource | 3 | `/api/v1/resources` |
| hr | 34 | `/api/v1/departments`, `/api/v1/employees`, ... |
| mrp (Mount) | 17 | `/api/v1/mrp` |
| loyalty (Mount) | 27 | `/api/v1/loyalty` |
| maintenance | 28 | `/api/v1/maintenance` |
| fleet | 52 | `/api/v1/fleet` |
| delivery | 4 | `/api/v1/delivery` |
| report | 2 | `/api/v1/reports` |

مسارات عامة إضافية خارج `/api/v1`: `GET /` (Base), `POST /api/v1/users/login`, `GET|POST /api/v1/databases`.

### 2.2 وحدات معرّفة لكنها غير مثبتة في الـ Router حالياً

10 وحدات لها `routes.go` لكنها غير مرتبطة في `router.go`: ecommerce(11)، helpdesk(10)، livechat(7)، marketing(19)، portal(8)، quality(5)، repair(5)، subscription(7)، survey(4)، website(3) — **إجمالي ~79 handler غير نشطة**. لا تظهر في مقاييس المراقبة لأنها لا تصل للموجّه.

### 2.3 المعلمات الديناميكية (أهمية لـ P2 — الكاردينالية)

عدد الوسائط الديناميكية في الـ route templates عبر جميع الـ routes:

| المعامل | التكرار |
| :--- | :--- |
| `{id}` | 325 |
| `{code}` | 5 |
| `{slug}` | 3 |
| `{service_id}`, `{orderID}`, `{odometer_id}`, `{log_id}`, `{line_id}`, `{contract_id}` | 2 لكل منها |
| `{token}`, `{session_id}`, `{itemId}`, `{coupon_id}` | 1 لكل منها |

**الاستنتاج لـ P2:** عند استخدام `chi.RouteContext().RoutePattern()` تبقى الـ labels محدودة بعدد الـ templates (507 تقريباً)، وليست بعدد القيم الفعلية — وهذا يبقي cardinality منضبطاً.

**عقد P2 (منفّذ):** middleware يقرأ `RoutePattern()` بعد اكتمال المعالجة مع fallback `unknown` للطلبات غير المطابقة أو خارج chi router. كاردينالية `cashflow_http_requests_total` = (عدد Methods المستخدمة ≤ 6) × (عدد الـ templates ≈ 507 + `unknown`) × (4 فئات حالة) ≈ **~12k series** أقصى حد نظري، وعملياً أقل بكثير لأن كل endpoint يستخدم Methods ثابتة واحدة أو اثنتين. أُضيفت عائلات جديدة موثقة في `registry.go`:

| العائلة | الشكل |
| :--- | :--- |
| `cashflow_http_request_duration_seconds{method,route}` | histogram (11 bucket) |
| `cashflow_http_response_size_bytes{method,route}` | histogram (9 bucket) |
| `cashflow_probe_duration_seconds{probe}` | histogram — probe مغلق على `{livez, readyz}` ويُقاس خارج حركة الأعمال |
| `cashflow_go_gc_seconds_total` | gauge — زمن GC التراكمي منذ بدء العملية من `runtime.MemStats.PauseTotalNs` |

**عقد الشكل النصي (P3 منفّذ):** `text/plain; version=0.0.4; charset=utf-8`، كل العائلات بادئتها `cashflow_` ومعرّفة عبر `expfmt.MetricFamilyToText` بعد ترتيب حسب الفئة (process → go → http → probe → db)، رأس تزييني، ورؤوس `Cache-Control: no-store` و`X-Content-Type-Options: nosniff`. صفحة index تفاعلية (`IndexPage()`) بلا assets خارجية تُثبَّت في P4 على `/`. التحقق: `expfmt.NewTextParser(model.LegacyValidation)` يقرأ كامل النص بلا أخطاء.

---

## 3. شكل مخرجات `/metrics` JSON الحالي (Snapshot)

مصدر البيانات: [`internal/infrastructure/runtime/metrics/metrics.go`](../internal/infrastructure/runtime/metrics/metrics.go) — الناتج عبر `Metrics` struct + `runtime.MemStats` + عدادات ذرية + `DBStatsProvider`.

```json
{
  "server_url": "http://localhost:8070",
  "memory_alloc_bytes": 524288,
  "memory_total_bytes": 10485760,
  "memory_sys_bytes": 8388608,
  "heap_alloc_bytes": 4194304,
  "heap_idle_bytes": 2097152,
  "heap_inuse_bytes": 6291456,
  "num_goroutines": 12,
  "num_gc": 3,
  "uptime_seconds": 3600.5,
  "http_requests_total": 1000,
  "http_2xx_total": 950,
  "http_4xx_total": 30,
  "http_5xx_total": 20,
  "avg_latency_ms": 12.34,
  "db": {
    "max_conns": 25,
    "active_conns": 3,
    "idle_conns": 2,
    "wait_count": 0
  }
}
```

### 3.1 جدول الحقول

| المفتاح | النوع | الوصف | ملاحظات التوافق |
| :--- | :--- | :--- | :--- |
| `server_url` | string | العنوان المعلن (`RegisterServerURL`) | `omitempty` — غائب إن لم يُسجَّل |
| `memory_alloc_bytes` | uint64 | `runtime.MemStats.Alloc` | ثابت لغرض التوافق |
| `memory_total_bytes` | uint64 | `MemStats.TotalAlloc` | ثابت |
| `memory_sys_bytes` | uint64 | `MemStats.Sys` | ثابت |
| `heap_alloc_bytes` | uint64 | `MemStats.HeapAlloc` | ثابت |
| `heap_idle_bytes` | uint64 | `MemStats.HeapIdle` | ثابت |
| `heap_inuse_bytes` | uint64 | `MemStats.HeapInuse` | ثابت |
| `num_goroutines` | int | `runtime.NumGoroutine()` | ثابت |
| `num_gc` | uint32 | `MemStats.NumGC` | ثابت |
| `uptime_seconds` | float64 | مدة الإقلاع منذ `startTime` | ثابت |
| `http_requests_total` | uint64 | عداد الطلبات الكلي (atomic) | ثابت |
| `http_2xx_total` | uint64 | الردود الناجحة | ثابت |
| `http_4xx_total` | uint64 | أخطاء العميل | ثابت |
| `http_5xx_total` | uint64 | أخطاء الخادم | ثابت |
| `avg_latency_ms` | float64 | متوسط زمن الاستجابة | ثابت (سيبقى لكن يُرفق بـ Histogram لاحقاً) |
| `db` | object | صحة حوض الاتصالات `pgxpool` | `omitempty` — غائب مع `storage=memory` أو عند تعطّل المزوّد |

### 3.2 سلوك يجب الحفاظ عليه (Baseline Behaviors)

1. `Content-Type: application/json` مع `200 OK`.
2. طلبات `/metrics`, `/livez`, `/readyz` **لا** تُحصى في عدادات الـ HTTP (استثناء صريح في `metrics.Middleware`).
3. الردود الأقل من 400 على مسارات المراقبة لا تُسجَّل في `structuredLogger`.
4. العدادات تراكمية (cumulative) لا تُصفَّر أثناء التشغيل.
5. المتوسط `avg_latency_ms` يحسب من القسمة على `requestCount` (لا يميّز التطرفات — سيعالجه Histogram في P1/P2).
6. المصنّف الحالي لأكواد الحالة: أي `status >= 200` (بما فيه الـ 3xx) يُحصى داخل حقل `http_2xx_total`؛ و`status >= 400` في `http_4xx_total`، و`>= 500` في `http_5xx_total`. سيُستبدل في P2 بـ `status_class` صريح (2xx/3xx/4xx/5xx).

---

## 4. نقاط الدمج في دورة الحياة (Lifecycle Integration Points)

المصدر: [`internal/platform/app/app.go`](../internal/platform/app/app.go) و[`internal/infrastructure/runtime/server/server.go`](../internal/infrastructure/runtime/server/server.go).

| الموقع | السطر | الدور | نقطة الدمج المخطط لـ P4–P5 |
| :--- | :--- | :--- | :--- |
| `metrics.RegisterDB(&pgxStatsProvider{pool})` | app.go:292 | تسجيل مزوّد إحصاءات الحوض (postgres) | سيتحول إلى instance مُمرر عند بناء الـ Registry في P1 |
| `metrics.RegisterServerURL(cfg.BaseURL())` | app.go:293 / 311 | تسجيل عنوان الخادم | نفسه |
| `server.NewServer(cfg, router, ...)` | app.go:354 | إنشاء خادم HTTP الوحيد | سيضاف إنشاء خادم الإدارة (`management_router.go`) وإنشاء listener مسبق الربط |
| `a.server.Run(sigCtx)` | app.go:379 | تشغيل دورة الحياة الوحيدة | ستضاف إدارة خادمين (P5) |
| `Server.Run` — PHASE 3 (Stratup) | server.go:81-103 | `go func(){ListenAndServe}` ثم `MarkReady()` | **فجوة 3.3:** `MarkReady` قبل ضمان نجاح الربط — تُصلح في P5 |
| `Server.Run` — PHASE 5/6 (Drain/Shutdown) | server.go:116-140 | `MarkNotReady` → sleep → `Shutdown` | ستشمل الخادمين معاً (P5) |
| `Server.Run` — PHASE 7 (Cleanup) | server.go:142-158 | إيقاف العمال ثم إغلاق الموارد | ثابت |

ملاحظة: التشغيل يستخدم `log/slog` مزدوج الوضع (Pretty في الطرفية / JSON في الإنتاج) عبر `app.initLogger` — لا تغيير مخططاً عليه.

---

## 5. معايير إتمام P0 (حالة التحقق)

- [x] قرار مسار النص معتمد: `GET /metrics` على `127.0.0.1:8066` بصيغة OpenMetrics + `/metrics/json` للتوافق.
- [x] شكل JSON الحالي موثق (قسم 3) دون تغيير أسماء الحقول.
- [x] جرد المسارات معتمد (قسم 2): 507 handler نشطة + 79 غير مثبتة + المعلمات الديناميكية.
- [x] نقاط دمج دورة الحياة محددة (قسم 4).
- [x] اختبارات baseline مضافة: `internal/adapters/http/router_baseline_test.go` و`internal/infrastructure/runtime/metrics/metrics_test.go`.
- [x] `go test ./...` خضراء قبل أي تغيير معماري، و`go test -race` نظيفة على الحزمتين المعدلتين.
