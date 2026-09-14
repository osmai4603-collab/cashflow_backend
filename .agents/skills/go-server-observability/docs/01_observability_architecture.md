# معمارية المراقبة والشفافية التشغيلية (Observability Architecture)

تحويل خادم Go من "صندوق أسود" مجهول الحالة إلى "صندوق زجاجي" شفاف (Glass Box) هو الضمان الحقيقي لاستقرار أنظمة الإنتاج وتسهيل التشخيص السريع.

## مبادئ معمارية المراقبة

### 1. مبدأ فصل مسارات الإدارة (Isolated Management Plane)

يجب ألا يتم كشف نقاط المراقبة والتشخيص على المنفذ العام للمستخدمين:

- **المنفذ العام (`:8070`)**: يخدم واجهات الأعمال البرمجية فقط (`/api/v1/*`). أي طلب لمسارات `/metrics` أو `/debug` يعيد خطأ `404 Not Found`.
- **منفذ الإدارة الداخلي (`:8066`)**: يخدم مسارات القياس والمراقبة، ومحمي بشبكة داخلية خاصة أو تصريح Bearer Token عند ربطه على `0.0.0.0`.

### 2. التنسيق المزدوج للمقاييس (Dual-Format Exposition)

- **معيار OpenMetrics / Prometheus (`/metrics`)**:
  - نصي (Text format version 0.0.4).
  - مخصص لأنظمة التجميع المركزية (Prometheus, Grafana, Datadog).
  - يعتمد على العدادات التراكمية (Cumulative Counters) والمدرجات التكرارية (Histograms).
- **لقطة JSON الحظية السريعة (`/metrics/json`)**:
  - حمولة JSON مضغوطة ومنظمة تُنتج بتمريرة واحدة (Single-pass serialization).
  - مخصصة لأدوات الطرفية المحلية (`tool/monitor.sh`) والـ Health checks الذكية.
  - تحتوي على مئينات النافذة المنزلقة (`p50`, `p95`, `p99`) والحالة اللحظية للـ DB Pool وآخر الأخطاء.

### 3. مسار مراقبة تجربة المستخدم الحقيقي (Real User Monitoring - RUM)

- مسار `POST /rum`:
  - يستقبل مؤشرات الأداء الحيوية للواجهات الأمامية (Core Web Vitals):
    - `LCP` (Largest Contentful Paint)
    - `FID` / `INP` (Interaction to Next Paint)
    - `CLS` (Cumulative Layout Shift)
  - يتيح ربط أداء الباك إند بتجربة المستخدم الفعلية في المتصفح.

### 4. أدوات التشخيص المتقدمة (`/debug/pprof/*`)

- تمكين حزمة Go الرسمية `net/http/pprof` على منفذ الإدارة فقط.
- تفعيلها اختياري ومشروط عبر التكوين (`management.pprof_enabled=true`) لأغراض تحليل الذاكرة (Heap Profiling) واستكشاف تسريبات الـ Goroutines.
