# تقرير مقارنة أدوات المراقبة والمقاييس (Monitoring & Metrics Comparison Report)

## خادم Mattermost Server مقابل خادم Cashflow Backend

---

## الفهرس

1. [الملخص التنفيذي](#الملخص-التنفيذي)
2. [أولاً: منظومة المراقبة والمقاييس في Mattermost Server](#أولا-منظومة-المراقبة-والمقاييس-في-mattermost-server)
    - [1.1 بنية المقاييس ونواة النظام (Prometheus & OpenMetrics)](#11-بنية-المقاييس-ونواة-النظام-prometheus--openmetrics)
    - [1.2 عزل خادم المراقبة (Dedicated Metrics Server)](#12-عزل-خادم-المراقبة-dedicated-metrics-server)
    - [1.3 تغطية الأنظمة الفرعية (Subsystems Coverage)](#13-تغطية-الأنظمة-الفرعية-subsystems-coverage)
    - [1.4 مراقبة تجربة المستخدم الحقيقية (Real User Monitoring - RUM)](#14-مراقبة-تجربة-المستخدم-الحقيقية-real-user-monitoring---rum)
    - [1.5 تجميع مقاييس الإضافات (Plugin Metrics Aggregation)](#15-تجميع-مقاييس-الإضافات-plugin-metrics-aggregation)
    - [1.6 تحليل الأداء وتفريغ الذاكرة (pprof Profiling Suite)](#16-تحليل-الأداء-وتفريغ-الذاكرة-pprof-profiling-suite)
    - [1.7 لوحات Grafana والقياس عن بعد (Dashboards & Telemetry)](#17-لوحات-grafana-والقياس-عن-بعد-dashboards--telemetry)
    - [1.8 فحص السلامة (Health Check)](#18-فحص-السلامة-health-check)
3. [ثانياً: منظومة المراقبة والمقاييس في Cashflow Backend](#ثانيا-منظومة-المراقبة-والمقاييس-في-cashflow-backend)
    - [2.1 محرك المقاييس المخصص الخفيف (In-House JSON Metrics Engine)](#21-محرك-المقاييس-المخصص-الخفيف-in-house-json-metrics-engine)
    - [2.2 فحوصات السلامة المتوافقة مع Kubernetes (/livez و /readyz)](#22-فحوصات-السلامة-المتوافقة-مع-kubernetes-livez-و-readyz)
    - [2.3 وسائط التتبع والتسجيل المهيكل (Middlewares & Structured Logging)](#23-وسائط-التتبع-والتسجيل-المهيكل-middlewares--structured-logging)
    - [2.4 ملف تعريف الأداء (pprof Profiler)](#24-ملف-تعريف-الأداء-pprof-profiler)
4. [ثالثاً: جدول المقارنة الشامل (Detailed Comparison Matrix)](#ثالثا-جدول-المقارنة-الشامل-detailed-comparison-matrix)
5. [رابعاً: التحليل الهندسي والتوصيات لمشروع Cashflow Backend](#رابعا-التحليل-الهندسي-والتوصيات-لمشروع-cashflow-backend)

---

## الملخص التنفيذي

يقدم هذا التقرير تحليلاً مقارناً معمقاً لتقنيات وأدوات المراقبة والمقاييس (Observability & Monitoring) بين مشروعي:

1. **Mattermost Server (`mattermost/server`):** منصة اتصالات مؤسسية عالية التوزيع (Enterprise Distributed System) مجهزة ببنية مراقبة شاملة تعتمد معايير الصناعة السحابية (Prometheus & OpenMetrics).
2. **Cashflow Backend (`cashflow_backend`):** نظام تخطيط موارد مؤسسات (Modular ERP) مبني بلغة Go، يتبنى نهجاً بسيطاً خالياً من التبعيات الخارجية المفرطة (Zero-External Dependencies) مع تطبيق صارم لمعايير الفحص السحابي الحديث (Cloud-Native Health Probes).

---

## أولاً: منظومة المراقبة والمقاييس في Mattermost Server

### 1.1 بنية المقاييس ونواة النظام (Prometheus & OpenMetrics)

- **المكتبات المعتمدة:**
  - [`github.com/prometheus/client_golang`](file:///home/osm/mattermost/server/go.mod#L61)
  - `github.com/prometheus/client_model`
  - `github.com/prometheus/common/expfmt`

- **المعمارية والواجهة الموحدة:**
  - يعرّف النظام واجهة موحدة [`MetricsInterface`](file:///home/osm/mattermost/server/einterfaces/metrics.go#L13) تضم ما يزيد عن 80 دالة رصد مختلفة.
  - التطبيق الفعلي يقع في حزمة [`enterprise/metrics/metrics.go`](file:///home/osm/mattermost/server/enterprise/metrics/metrics.go) متضمناً أنواع المقاييس الكاملة: `Counters`, `Gauges`, `Histograms`, و `GaugeFunc`.

### 1.2 عزل خادم المراقبة (Dedicated Metrics Server)

- **المسار:** [`channels/app/platform/metrics.go`](file:///home/osm/mattermost/server/channels/app/platform/metrics.go#L110-L142)

- **الخصائص:**
  - يعمل خادم المراقبة على منفذ شبكي مستقل (الافتراضي `:8067`) عبر بروتوكول HTTP منفصل تماماً عن منفذ الخدمة العام (`:8065`).
  - يحقق هذا العزل حماية أمنية للبيانات الحساسة وأدوات الـ Profiling، ويمنع تأثير سحب المقاييس (Scrape Load) على زمن استجابة المستخدمين العاديين.
  - يوفر صفحة رئيسية تفاعلية تعرض روابط للمقاييس ومختلف مسارات الـ Profiling.

### 1.3 تغطية الأنظمة الفرعية (Subsystems Coverage)

يتم تصنيف المقاييس تحت البادئة `mattermost` وتشمل أكثر من 20 نظاماً فرعياً:

- **قواعد البيانات (`db`):**
  - مراقبة اتصالات الـ Master، والـ Read Replicas، والـ Search Connections.
  - قياس تأخر النسخ المتطابق لحظياً بالحجم والزمن (`DbReplicaLagGaugeAbs`, `DbReplicaLagGaugeTime`).
  - قياس أزمنة استعلامات كل مخزن بيانات وكل عملية CRUD عبر `ObserveStoreMethodDuration`.
- **الشبكة والواجهات (`http`, `api`):**
  - حساب أعداد الطلبات الكلية، ومعدلات الأخطاء، وأزمنة المعالجة لكل مسار ولكل كود استجابة.
  - إحصائيات اتصالات WebSocket الحية مصنفة حسب نوع العميل.
- **الذاكرة المخبأة (`cache`):**
  - نسب الإصابة والخطأ والإبطال (Hit / Miss / Invalidation) لذاكرة L1 المحلية و Redis.
- **التكتل والتزامن (`cluster`):**
  - تتبع أزمنة طلبات التزامن بين العقد، وتوزيع أحجام الرسائل الاحتياطية الموثوقة.
- **المهام الخلفية والإشعارات (`jobs`, `notifications`):**
  - تتبع المهام النشطة حسب النوع، وإحصائيات تسليم الإشعارات الفورية (Push Notifications) مع تفصيل أسباب الفشل حسب منصات الهواتف (iOS / Android).

### 1.4 مراقبة تجربة المستخدم الحقيقية (Real User Monitoring - RUM)

- **المسار:** [`channels/api4/metrics.go`](file:///home/osm/mattermost/server/channels/api4/metrics.go)

- **الخصائص:**
  - يستقبل الخادم مقاييس أداء تجربة المستخدم مباشرة من تطبيقات الويب، وسطح المكتب، والجوال عبر نقطة نهاية الـ API.
  - تسجيل مؤشرات الويب الأساسية (Core Web Vitals): TTFB, LCP, INP, CLS, DOM Interactive.
  - قياس سرعة التبديل بين القنوات والفِرق وتكلفة تحميل الشريط الجانبي (RHS Load Duration).
  - استهلاك المعالج والذاكرة لتطبيق سطح المكتب، وسرعة واستجابة شبكة تطبيقات الجوال.

### 1.5 تجميع مقاييس الإضافات (Plugin Metrics Aggregation)

- **المسار:** دالة `wrapMetricsHandler` و `addPluginLabelToMetrics` في [`channels/app/platform/metrics.go`](file:///home/osm/mattermost/server/channels/app/platform/metrics.go#L259-L347).

- **الخصائص:**
  - يتيح للملحقات البرمجية (Plugins) تصدير مقاييس مخصصة عبر خطاف (Hook) `ServeMetrics`.
  - يقوم الخادم عند استدعاء `/metrics` بطلب مقاييس جميع الملحقات النشطة وإعادة تشفيرها تلقائياً مع إضافة الوسم `plugin_id="<id>"`، مما يوفر نقطة سحب موحدة لكامل المنظومة.

### 1.6 تحليل الأداء وتفريغ الذاكرة (pprof Profiling Suite)

- **المسار:** [`channels/app/platform/metrics.go`](file:///home/osm/mattermost/server/channels/app/platform/metrics.go#L179-L192)

- **الخصائص:**
  - توفير المسارات القياسية: `/debug/pprof/profile` (CPU)، `/debug/pprof/heap` و `/debug/pprof/allocs` (Memory Allocation).
  - توفير مسارات تنافس الموارد: `/debug/pprof/goroutine`, `/debug/pprof/block`, `/debug/pprof/mutex`, `/debug/pprof/threadcreate`.
  - إمكانية ضبط وتفعيل تتبع حجب الخيوط ديناميكياً عبر إعداد `BlockProfileRate`.

### 1.7 لوحات Grafana والقياس عن بعد (Dashboards & Telemetry)

- **لوحات جاهزة:** يحتوي المجلد [`build/docker/grafana/dashboards/mattermost/`](file:///home/osm/mattermost/server/build/docker/grafana/dashboards/mattermost/) على ملفات JSON جاهزة للاستيراد:
  - لوحة مراقبة الأداء العامة (v2).
  - لوحة مؤشرات الأداء الرئيسية (KPI Metrics).
  - لوحات أداء تطبيقات الويب وسطح المكتب.

- **خدمة القياس عن بعد:** حزمة [`platform/services/telemetry`](file:///home/osm/mattermost/server/platform/services/telemetry/telemetry.go) ترسل دورياً معلومات تشخيصية مجهولة الهوية عن حجم التكتل ومميزات النظام المفعلة لتحسين استقرار المنتج (مع إمكانية تعطيلها).

### 1.8 فحص السلامة (Health Check)

- **المسار:** [`channels/api4/system.go`](file:///home/osm/mattermost/server/channels/api4/system.go#L147)

- نقطة نهاية مركزية `/api/v4/system/ping` تعيد حالة الخادم وإصدارات تطبيقات الجوال المطلوبة، وتدعم معامل `?get_server_status=true` للتحقق من الاتصال الحي بقاعدة البيانات.

---

## ثانياً: منظومة المراقبة والمقاييس في Cashflow Backend

### 2.1 محرك المقاييس المخصص الخفيف (In-House JSON Metrics Engine)

- **المسار:** [`internal/infrastructure/runtime/metrics/metrics.go`](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/infrastructure/runtime/metrics/metrics.go)

- **الخصائص الهندسية:**
  - محرك محلي صُمم بالكامل دون الاعتماد على مكتبات خارجية، مستفيداً من العمليات الذرية السريعة (`sync/atomic`) وبنية ذاكرة Go القياسية (`runtime.MemStats`).
  - يقدم مخرجاته بصيغة `application/json` عبر نقطة النهاية `GET /metrics`.
  - **المقاييس المسجلة:**
    - **مقاييس النظام:** الذاكرة المحجوزة (`memory_alloc_bytes`)، ذاكرة الكومة المستخدمة والخاملة (`heap_inuse_bytes`, `heap_idle_bytes`)، عدد الـ Goroutines، دورات تنظيف الذاكرة (`num_gc`)، ومدة التشغيل الكلية (`uptime_seconds`).
    - **حركة المرور HTTP:** إجمالي الطلبات (`http_requests_total`)، تصنيف الحالات (`http_2xx_total`, `http_4xx_total`, `http_5xx_total`)، ومتوسط زمن الاستجابة العام بالمللي ثانية (`avg_latency_ms`).
    - **قاعدة البيانات:** عبر واجهة `DBStatsProvider` لحوض اتصالات `pgxpool` (الحد الأقصى، الاتصالات النشطة، الخاملة، وعدد الطلبات المنتظرة).

### 2.2 فحوصات السلامة المتوافقة مع Kubernetes (`/livez` و `/readyz`)

- **المسار:** [`internal/infrastructure/runtime/health/health.go`](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/infrastructure/runtime/health/health.go)

- **الخصائص الهندسية:**
  - تم تصميم الفحوصات وفقاً لأدق المعايير السحابية المعتمدة لبيئات Kubernetes:
    - **مسبار الحيوية (`/livez`):** يفحص صحة عملية Go المعالجة داخلياً فقط؛ ولا يفحص التبعيات الخارجية بتاتاً لتجنب مشكلة إعادة تشغيل الـ Pods المتسلسلة (Cascading Pod Restarts) عند حدوث وميض أو تأخير مؤقت في قاعدة البيانات.
    - **مسبار الجاهزية (`/readyz`):** يتحقق من اكتمال الإقلاع وعدم دخول الخادم في مرحلة التصريف (Draining)، ويفحص الاتصال الفعلي بقاعدة البيانات عبر `Pinger.Ping()`. في حال فشله يتم إخراج الـ Pod من موازن الأحمال دون قتله.

### 2.3 وسائط التتبع والتسجيل المهيكل (Middlewares & Structured Logging)

- **المسار:** [`internal/adapters/http/router.go`](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/adapters/http/router.go#L53-L65)

- **الخصائص الهندسية:**
  - **وسيط المقاييس (`metrics.Middleware`):** يحسب زمن كل طلب ويرفع العدادات الذرية، مع استثناء مسارات المراقبة (`/metrics`, `/livez`, `/readyz`) من الحساب لتجنب تشويه البيانات.
  - **وسيط السجلات المهيكلة (`structuredLogger`):** يعتمد على الحزمة القياسية `log/slog` في Go 1.21+؛ مبرمج بذكاء لكتم تسجيل مسارات المراقبة الروتينية إلا إذا أنتجت رمز خطأ (HTTP Status >= 400).
  - **معرف الطلبات (`middleware.RequestID`):** يربط كل استجابة وسجل بمعرف تتبع فريد.

### 2.4 ملف تعريف الأداء (pprof Profiler)

- **المسار:** [`internal/adapters/http/router.go`](file:///home/osm/StudioProjects/cashflow/cashflow_backend/internal/adapters/http/router.go#L64)

- **الخصائص:**
  - مدمج عبر موجه Chi القياسي تحت المسار الموحد `/debug` عبر `middleware.Profiler()`.
  - يوفر وصولاً سريعاً لـ pprof لفحص المعالج والذاكرة وخيوط التشغيل أثناء التطوير والاختبار.

---

## ثالثاً: جدول المقارنة الشامل (Detailed Comparison Matrix)

| البعد التقني | Mattermost Server (`server`) | Cashflow Backend (`cashflow_backend`) | التقييم الهندسي |
| :--- | :--- | :--- | :--- |
| **بروتوكول تصدير المقاييس** | **Prometheus / OpenMetrics** (Text Format) | **JSON API** مخصص عبر HTTP | بروميثيوس معيار صناعي للمراقبة السحابية والتجميع، بينما JSON أبسط للقراءة المباشرة ولكن يصعب ربطه بأنظمة الـ Alerting المركزية. |
| **المكتبات والتبعيات** | `prometheus/client_golang` الرسمية ومكتبات Expfmt المرافقة | مكتبات Go القياسية المدمجة (`sync/atomic`, `runtime`) | تفوق لـ Cashflow في انعدام التبعيات الخارجية، وتفوق لـ Mattermost في قوة الأدوات المتاحة. |
| **عزل المنفذ والشبكة** | **منفذ مستقل ومعزول تماماً** (`:8067`) مع خادم HTTP مخصص | **منفذ مشترك** يخدم الـ API ونقاط المراقبة والتشخيص معاً | عزل المنفذ في Mattermost أكثر أماناً ويمنع كشف مقاييس الأداء أو pprof للمستخدمين الخارجيين. |
| **مستوى تفصيل المقاييس (Granularity)** | **دقيق للغاية:** لكل مسار API، لكل جدول واستعلام، تفصيل أخطاء الإشعارات، مقاييس الـ WebSocket والـ Cache | **عام وتجميعي:** إجمالي طلبات الـ HTTP، ونسب الحالات العامة، ومتوسط زمن استجابة وحيد | Mattermost يتفوق في تشخيص الاختناقات بدقة؛ بينما Cashflow يقدم مؤشرات صحة عامة فقط. |
| **فحوصات السلامة (Health Probes)** | نقطة تقليدية `/api/v4/system/ping` تقبل فحص قاعدة البيانات عبر Query Param | **معيارية سحابية صارمة:** فصل كامل بين الحيوية `/livez` والجاهزية `/readyz` | **تفوق صريح لـ Cashflow** لمطابقته لأفضل ممارسات بيئات Kubernetes وتفادي الانهيارات المتسلسلة. |
| **مراقبة تجربة العميل (RUM)** | مدعومة عبر جمع مؤشرات Core Web Vitals واستهلاك موارد الذاكرة من تطبيقات الويب/الجوال | غير متوفرة (مقتصرة على مقاييس الخادم الخلفي) | Mattermost يربط أداء الخادم بتجربة المستخدم الفعلية في المتصفح والهاتف. |
| **تكامل الملحقات (Plugins)** | يدعم الجمع التلقائي وإعادة وسم مقاييس إضافات النظام | لا ينطبق (نظام وحدات داخلية متكامل) | ميزة متقدمة جداً في Mattermost تناسب المنصات القابلة للتوسيع. |
| **أدوات التشخيص (pprof)** | مدمجة في خادم المراقبة المستقل مع واجهة HTML وضبط لمعدل حجب الخيوط | مدمجة في الموجه الرئيسي تحت `/debug` | كلاهما يدعم pprof، لكن Mattermost يوفر عزلاً وتحكماً برمجياً أعمق. |
| **لوحات المراقبة (Dashboards)** | لوحات **Grafana** احترافية جاهزة ومرفقة بالمشروع | غير متوفرة ضمن المستودع | Mattermost يوفر حل مراقبة متكامل من اليوم الأول (Out-of-the-box). |
| **السجلات المهيكلة (Logging)** | مكتبة مخصصة `mlog` (فوق Uber Zap) متكاملة مع مجمع مقاييس الأخطاء | الحزمة القياسية `log/slog` مع مرشح ذكي لكتم مسارات المراقبة | كلاهما يعتمد السجلات المهيكلة؛ استخدام `log/slog` في Cashflow حديث وقياسي وممتاز. |

---

## رابعاً: التحليل الهندسي والتوصيات لمشروع Cashflow Backend

### 1. نقاط القوة الاستثنائية الحالية في Cashflow

- **نقاء الكود وانعدام التبعيات:** إنجاز تتبع الذاكرة ومؤشرات الـ HTTP وحوض الاتصالات باستخدام مكتبات Go القياسية والعمليات الذرية بدون سحب حزم ضخمة.

- **فحوصات Kubernetes المثالية:** تطبيق مبدأ الفصل بين `/livez` و `/readyz` تم بشكل احترافي يضاهي أرقى المشاريع السحابية.
- **الفلترة الذكية للسجلات:** كتم مسارات الفحص الروتيني في `structuredLogger` يمنع امتلاء أدوات جمع السجلات بآلاف السجلات غير المفيدة.

### 2. توصيات مستوحاة من Mattermost لتطوير Cashflow مستقبلاً

1. **دعم تنسيق بروميثيوس (Prometheus Exposition Format):**
    - بدلاً من (أو بجانب) JSON، يمكن إضافة نقطة تصدير بتنسيق بروميثيوس النصي القياسي لتسهيل ربط النظام مباشرة مع Prometheus و Grafana و VictoriaMetrics دون الحاجة لكتابة Custom Exporters.
2. **توزيع المقاييس على الوحدات (Per-Module / Per-Route Metrics):**
    - تحويل عداد الطلبات ومتوسط الزمن إلى مصفوفة ذات وسوم (Labels) ترصد الأداء لكل وحدة عمل (مثل: `/api/v1/accounting`, `/api/v1/sale`, `/api/v1/stock`) لمعرفة الوحدة المتسببة في بطء النظام بدقة.
3. **تأمين أو عزل مسار `/debug` و `/metrics`:**
    - حالياً مسار `/debug` و `/metrics` متاحان على المنفذ العام؛ في بيئة الإنتاج يجب إما حمايتهما بـ Middleware للمصادقة أو عزلهما على منفذ إداري داخلي خاص (Internal Management Port) كنهج Mattermost لمنع كشف بنية الذاكرة وأداء الخادم لأطراف خارجية.
4. **بناء لوحة Grafana مرافقة للمشروع:**
    - تضمين لوحة Grafana قياسية في مجلد التوثيق أو التطوير تعرض استهلاك الذاكرة، وعدد اتصالات `pgxpool` النشطة والخاملة، ومعدل الطلبات في الثانية (RPS).

### 3. التوصيات المنفَّذة الآن (حالة P8)

> بقيت توصيات القسم السابق (المستوحاة من Mattermost) توصيات مستقبلية في وقت كتابة التقرير. بموجب مراحل [`observability_security_execution_plan.md`](observability_security_execution_plan.md) (P0–P7) **أُنفذت الأربع جميعها** وأصبحت جزءاً من المستودع:

| التوصية | ما نُفِّذ الآن | الملفات/التحقق |
|:---|:---|:---|
| **1. دعم تنسيق بروميثيوس** | مصدِّر نصي OpenMetrics (0.0.4) عبر `prometheus/client_golang` يقرأ من **نفس** الـ Registry الذي يغذي `/metrics/json` — بلا مصدرين متنافسين | `metrics/exporter.go`, `registry.go`؛ يعتمده `expfmt.TextParser`، و`docker-compose.monitoring.yml` يسحبه من `server:8066` |
| **2. مقاييس per-module / per-route** | `cashflow_http_requests_total{duration,size}` بوسوم `method` + `route` (قالب مؤكد بـ `chi.RoutePattern`: مسارات ديناميكية كقالب واحد) + `status_class`؛ المسارات غير المطابقة → `unknown` | `metrics.go` (`Middleware`, `routePatternFor`), `types.go`؛ باختبارات `TestMiddleware_*` |
| **3. تأمين/عزل `/debug` و `/metrics`** | كلاهما انتقل حصرياً إلى **خادم إدارة معزول** على منفذ مستقل (الافتراضي 8066) داخل دورة حياة الخادمين؛ pprof **مغلق افتراضياً** ولا يُفعَّل إلا بـ `management.pprof_enabled=true` (إعداد ثابت)؛ مصادقة `Bearer` اختيارية للمنفذ. المنفذ العام `8070` يرد **404** لكليهما مضموناً باختبارات | `management_router.go`, `server.go` (P4–P5)؛ فحص حي: عام 404، إدارة 200/401 |
| **4. لوحة Grafana مرافقة** | لوحة `cashflow-overview.json` provisioned تلقائياً (traffic, latency p50/p95/p99, Go runtime, DB pool, SRE) + Prometheus scrape config + 6 قواعد تنبيه bootstrap | `deploy/monitoring/` + `docker-compose.monitoring.yml` + `docs/monitoring_operations.md` (منهجية ضبط العتبات) |

الإضافات اللاحقة التي لم تكن ضمن التوصيات الأصلية وأُنجزت معها:
- **RUM** (P7): Endpoint `POST /rum` يستقبل مؤشرات تجربة المستخدم (TTFB/LCP/INP/CLS/DOM-Interactive) بـ histograms مقسومة على `client_type` مغلقة — بما يكافئ 1.4 في Mattermost. العقد في `docs/rum_api_contract.md`.
- **التنبيهات** (P6): قواعد bootstrap قابلة للمعايرة بعد قياس baseline في staging (التفاصيل في `docs/monitoring_operations.md`).
- **عزل** كامل لدورة الحياة (P5): ربط المنفذين مسبقاً قبل readiness، وإيقاف موحد بدون goroutines معلّقة — بما يعادل متانة خادم المراقبة المستقل في Mattermost.
