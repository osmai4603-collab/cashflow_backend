# P6 — تشغيل منظومة المراقبة (Prometheus + Grafana)

مرجع: [`observability_security_execution_plan.md`](observability_security_execution_plan.md) — المرحلة P6.

هذه الوثيقة تغطي: تشغيل المنظومة، الأمان، التنبيهات، و**عملية ضبط العتبات في staging** قبل الاعتماد عليها على-call.

---

## 1. المكوّنات

| المكوّن | الملف | الدور |
| :--- | :--- | :--- |
| Prometheus scrape config | `deploy/monitoring/prometheus.yml` | سحب `/metrics` من `server:8066` مع Bearer token + إرسال التنبيهات إلى Alertmanager |
| قواعد التنبيه (مُعايرة) | `deploy/monitoring/prometheus-alerts.yml` | 6 قواعد بقيم مُعايرة من قياسات حقيقية (وسوم `CALIBRATED`/`SLO-DERIVED`) |
| Alertmanager | `deploy/monitoring/alertmanager/alertmanager.yml` + خدمة compose | توجيه التنبيهات إلى webhook `ALERTMANAGER_WEBHOOK_URL` (Slack/Teams/PagerDuty-style) على `127.0.0.1:9093` |
| Grafana datasource | `deploy/monitoring/grafana/provisioning/datasources/` | مصدر بيانات `prometheus` (uid: `prometheus`) |
| Grafana dashboard provider | `deploy/monitoring/grafana/provisioning/dashboards/` | تحميل `cashflow-overview.json` تلقائياً |
| لوحة الإصدار الأول | `deploy/monitoring/grafana/dashboards/cashflow-overview.json` | RPS، 4xx/5xx، p50/p95/p99، Go runtime، pool، SRE |
| ساحة التطوير | `docker-compose.monitoring.yml` | Prometheus + Grafana للـ dev فقط |
| أوامر Makefile | `monitoring-up`/`down`/`logs`/`baseline` | تشغيل/إيقاف المنظومة + جَمع baseline معايرة |

---

## 2. التشغيل

```bash
# 1) شغّل التطبيق أولاً (ينشئ شبكة cashflow-monitoring-net المشتركة)
docker compose up -d

# 2) شغّل مكدس المراقبة (Prometheus + Grafana)
docker compose -f docker-compose.monitoring.yml up -d
```

أو عبر Makefile:

```bash
make monitoring-up            # لا يقبل أنه يعمل بدون التطبيق — تأكد من الترتيب أعلاه
make monitoring-logs
make monitoring-baseline      # جلسة قياس baseline + تسجيل summary.json (انظر سجل المعايرة)
make monitoring-down
```

النقاط النهائية (مربوطة على الـ loopback فقط):

| الخدمة | العنوان |
| :--- | :--- |
| Prometheus UI | `http://127.0.0.1:9090` |
| Grafana | `http://127.0.0.1:3000` (admin / `GRAFANA_ADMIN_PASSWORD` دعى إفتراضي `admin`) |
| مصدر البيانات | `http://prometheus:9090` داخل الشبكة |
| الهدف المكسور | `http://server:8066/metrics` عبر Bearer `MANAGEMENT_AUTH_TOKEN` |

**ملاحظة ترتيب:** شبكة `cashflow-monitoring-net` تُعرَّف `external` في ملف المراقبة، لذا يجب أن يكون المكدس الرئيسي مشغلاً أولاً أو سيخطئ `docker compose` بخطأ "network not found".

---

## 3. الأمان

- منفذ إدارة التطبيق يعمل على `8066` معزولاً عن المنفذ العام `8070` (P4)، والمقاييس/pprof **غير متاحة** على المنفذ العام (اختبار 404 مثبت).
- داخل الحاوية يُربط `MANAGEMENT_INTERFACE=0.0.0.0` (لتُتاح الشبكة المشتركة)، ويصبح **المصادقة إلزامية**؛ Prometheus يحمل نفس `MANAGEMENT_AUTH_TOKEN` لكل scrape. غياب التوكن أو اختلافه = رفض 401 (وليس 200 فارغ).
- نشر واجهتَي Prometheus وGrafana على `127.0.0.1` فقط (أبداً `0.0.0.0` في هذا الملف).
- لا توجد أسرار في اللوحة أو القواعد — متغيرات dashboard مقتصرة على `job` و`instance`.
- لا تسجّل أبداً `Authorization` headers أو DSN أو auth_token في سجلات/labels (منفّذ في الـ management router).

### عطل شائع

- **`/targets` يظهر 401:** `MANAGEMENT_AUTH_TOKEN` في ملف المراقبة لا يطابق توكن الخادم. فعلها بمتغير واحد:

  ```bash
  export MANAGEMENT_AUTH_TOKEN=$(openssl rand -hex 24)
  docker compose up -d                       # يعيد إنشاء الخادم بالتوكن الجديد
  docker compose -f docker-compose.monitoring.yml up -d
  ```

---

## 4. قواعد التنبيه وحالة المعايرة

`deploy/monitoring/prometheus-alerts.yml` يحتوي 6 قواعد. لكل قاعدة حالة معايرة (موضحة في الملف بوسوم `CALIBRATED` / `SLO-DERIVED` / `PENDING`):

| القاعدة | التعبير | القيمة | الحالة |
| :--- | :--- | :--- | :--- |
| `High5xxShare` | نسبة 5xx من كامل الحركة | > 0.5% لمدة 10m | SLO-DERIVED (المقاس محلياً = 0) |
| `HighLatencyP95` | `histogram_quantile(0.95, ...)` | > 400ms لمدة 10m | SLO-DERIVED (SLO 500ms − 20%) |
| `CashflowScrapeDown` | `up{job="cashflow"} == 0` | لمدة 2m | STRUCTURAL (لا عتبة) |
| `DBPoolSaturated` | active/max | > 80% لمدة 10m | CALIBRATED (ذروة مقاسة 84%) |
| `DBWaitClimbing` | `rate(wait_count_total[10m])` | > 50 waits/s | CALIBRATED (ذروة مقاسة ~1950/s) |
| `HeapGrowthAbnormal` | نمو heap في 10m | > 128MiB | CALIBRATED (المقاس = −2.4MiB / +4.9MiB) |

## 5. ضبط العتبات في staging (إلزامي قبل الـ on-call)

لا تُختار العتبات عشوائياً داخل الكود. العملية:

1. **اجمع baseline** من staging لمدة 5–7 أيام عمل كاملة (بما فيها عطلات/ذروة نهاية الشهر): الصق المقاييس عبر Prometheus فترة زمنية أطول (لا تعتمد على لوحة "آخر ساعة").
2. **حدد التوزيعات**: لكل مقياس احسب المتوسط، p95، p99، ونسبة الذروة أثناء الـ backups/المطالبة الليلية.
3. **صيغ القاعدة**:
   - نسبة 5xx: `max(0.10 × baseline_share, 1.5 × p95_share)` — مع تجنب الإنذارات الكاذبة عند عمليات إعادة الاستهلاك الليلية / المهجرات.
   - p95 latency: «SLO - هامش الأمان» (مثلاً إذا SLO = 500ms، فالعتبة 400ms).
   - pool: `max_conns × 0.75` متصلة لمدة 10م، وليس ذروة لحظية.
4. **غيّر `for:`** لتقليل الضجيج: قاعدة قد تنطلق لحظياً عند إعادة deploy يجب ألا تُقلقل.
5. **الإشعارات (Alertmanager):** مُعبّأ الآن كخدمة في `docker-compose.monitoring.yml` مع `alertmanager.yml` (webhook عبر `ALERTMANAGER_WEBHOOK_URL`). ضبط القناة الفعلية وتأكيد التسليم هو الخطوة الأخيرة عندما تصبح العتبات معتمدة.
6. **وثّق التغيير** في هذا الملف (تاريخ + baseline مستخدم + rationale).

## 6. سجل المعايرة (أُنجز)

جلسات baseline نُفذت عبر `make monitoring-baseline` (أداة `tool/calibrate_baseline.sh`)، والآثار مُركَّبة في القواعد أعلاه.

### 6.1 الجلسة الأولى — in-memory (التحقق من مخرجات القياس)

| بند | القيمة |
| :--- | :--- |
| التاريخ | 2026-09-12 |
| البيئة | محلية، in-memory driver، loopback |
| نمط الحمل | مسار رفض الأعمال (`/api/v1/companies` → 401) c=32×30k + `/readyz` c=64×30k + `/livez` c=16×5k (65k طلب) |
| العينات | 23 snapshot لـ `/metrics` كل 0.5 ثانية → `deploy/monitoring/baseline/` |

**المقاس:** `business_p95_min/median/max = 4.7/4.8/4.8ms` · `probe_p95 = 4.8ms` · `share_4xx = 100%` · `share_5xx = 0%` · `heap_delta = −2.4MiB` · goroutines ثابتة عند 10 · ~2.5k rps · الـ p95 مستقر (min≈max) بلا ذيل.

### 6.2 الجلسة الثانية — postgres حقيقي (معايرة الـ pool)

| بند | القيمة |
| :--- | :--- |
| التاريخ | 2026-09-12 |
| البيئة | محلية، `STORAGE=postgres`، postgres 16 (حاوية من الصفحة المحلية، بعد إصلاح سلسلة migrations)، DB فارغ يُهاجر تلقائياً إلى 69 |
| نمط الحمل | `/api/v1/companies` (401) c=32×30k + `/readyz` c=64×30k + `/livez` c=16×5k |
| العينات | 29 snapshot كل 0.5 ثانية → `deploy/monitoring/baseline/` |

**المقاس:** `pool_max_conns=25` · `pool peak ratio = 0.84` (عند استنزاف الحوض بحمل معادٍ) · `pool_waits ≈ 30k (≈1950 waits/s ذروة)` · `business_p95 = 4.8ms` (مسارات الرفض سريعة مستقرة) · `probe_p95 = 5.3ms` (`/readyz` يلمس الـ DB فعلاً) · `heap_delta = +4.9MiB` · goroutines=12.

**قرارات المعايرة (بعد الجلسات الاثنتين):**

| القاعدة | الأساس | القيمة المعتمدة |
| :--- | :--- | :--- |
| `High5xxShare` | SLO توافر 99.5% على نافذة 10m (المقاس 0 — لا 5xx حقيقي محلياً) | 0.5% |
| `HighLatencyP95` | SLO 500ms − هامش 20%؛ المقاس 4.8ms يترك هامشاً ~80× | 400ms |
| `HeapGrowthAbnormal` | المقاس −2.4MiB/+4.9MiB (ضجيج)؛ العتبة = max(المقاس×10، أرضية 128MiB) | 128MiB |
| `DBPoolSaturated` | مقاس real-DB: ذروة 84% فقط تحت حمل استنزاف معادٍ (لحظية) — استمرار 10m فوق 80% حدث تشبع حقيقي | 0.8 نسبة |
| `DBWaitClimbing` | مقاس real-DB: ذروة ~1950/s عند الاستنزاف؛ العتبة 50/s ≈ 2.5% من الذروة — إنذار عند ازدحام مستدام فقط | 50 waits/s |

**الخطوة الباقية قبل الـ on-call (كما في القسم 5):** اعتماد هذه القيم على staging 5–7 أيام (بما فيها الدفعات الليلية) لمراجعة قيمتي SLO والتحقق من عدم الـ alert fatigue، مع **Alertmanager** (مُعبّأ الآن في `docker-compose.monitoring.yml` + `deploy/monitoring/alertmanager/alertmanager.yml`) عبر `ALERTMANAGER_WEBHOOK_URL` كقناة الإشعارات.
