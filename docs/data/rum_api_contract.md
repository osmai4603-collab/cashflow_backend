# P7 — عقد API المراقبة الحقيقية (RUM)

مرجع: [`observability_security_execution_plan.md`](observability_security_execution_plan.md) — المرحلة P7.

يستقبل الخادم مؤشرات تجربة المستخدم الفعلية (Real User Monitoring) من متصفحات وصفحات المكتب والجوال، ويصدّرها كـ Prometheus histograms على منفذ الإدارة.

---

## 1. النقطة النهائية

```
POST /rum
```

- تُقدَّم حصرياً على **خادم الإدارة** (المنفذ 8066 افتراضياً) — ليست متاحة أبداً على المنفذ العام `8070`.
- تخضع لنفس سياسة المصادقة الإدارية: عندما `MANAGEMENT_REQUIRE_AUTH=true` يُشترط `Authorization: Bearer <token>` على كل طلب (نفس توكن P4).
- عند إيقاف `MANAGEMENT_METRICS_ENABLED` تصبح النقطة 404.
- حد حجم الطلب: **16 KiB**. حد المحتوى: `application/json`.
- المستقبل عادةً: `fetch` أو `navigator.sendBeacon` من كود العميل، وليس على المسار الساخن للعرض.

## 2. تنسيق الطلب

```json
{
  "client_type": "web",
  "ttfb_ms": 142,
  "lcp_ms": 810,
  "inp_ms": 16,
  "cls": 0.012,
  "dom_interactive_ms": 405,
  "app_version": "v1.4.2"
}
```

| الحقل | نوعه | إلزامي | القاعدة |
|:---|:---|:---|:---|
| `client_type` | string | **نعم** | من القائمة المغلقة `web` / `desktop` / `mobile` (غير حساس لحالة الأحرف) |
| `ttfb_ms` | number | لا | `>= 0`، متناهٍ، يُخزن كـ ثوان (÷1000) |
| `lcp_ms` | number | لا | `>= 0`، متناهٍ، ثوانٍ |
| `inp_ms` | number | لا | `>= 0`، متناهٍ، ثوانٍ |
| `cls` | number | لا | `>= 0`، متناهٍ، **بدون وحدة** (تُخزن كما هي) |
| `dom_interactive_ms` | number | لا | `>= 0`، متناهٍ، ثوانٍ |

القواعد:
- يجب وجود `client_type` صحيح **و** حقل مقياس واحد على الأقل.
- أي حقل بقيمة سالبة أو `1e999`/NaN/∞ = رفض 400 (يرسل الاسم كليه).
- `cls` قيمة بلا بُعد (درجة الإزاحة التراكمية)، والباقي بالإطارات الزمنية بوحدة المللي ثانية.
- **الحقول الإضافية غير المعروفة تجاهلها** — أبداً لا تُحوَّل لـ labels (توافق تقدمي ووضع أوامر في أمان الحقول).
- لا يُقبل محتوى زائد بعد كتلة JSON واحدة (تسلسلان متتاليان = 400).

## 3. الردود

| الحالة | المعنى |
|:---|:---|
| `204 No Content` | قُبلت العينة (جسم فارغ) |
| `400` | حمولة غير صالحة؛ الجسم `{"error":"..."}` يحدد السبب |
| `401` | صمام المصادقة الإدارية (عند `require_auth`) |
| `413` | تجاوز حد 16 KiB |
| `415` | `Content-Type` ليس `application/json` |

## 4. المقاييس المُصدَّرة (Prometheus text على `/metrics`)

كل عائلة histogram تُقسَّم بوسم `client_type` الجَّامد (3 قيم فقط):

| العائلة | المعنى | الوحدة |
|:---|:---|:---|
| `cashflow_rum_ttfb` | زمن أول بايت من بداية الملاحة | ثوانٍ |
| `cashflow_rum_lcp` | أكبر رسم للمحتوى (Largest Contentful Paint) | ثوانٍ |
| `cashflow_rum_inp` | زمن التفاعل إلى الرسم التالي (INP) | ثوانٍ |
| `cashflow_rum_cls` | درجة الإزاحة التراكمية | بدون وحدة |
| `cashflow_rum_dom_interactive` | لحظة DOM interactive | ثوانٍ |

- دلاء الوقت: `LatencyBuckets` ({0.005 … 10}s)؛ دلاء CLS: {0.001…1}.
- لكل عائلة `_bucket`/`_sum`/`_count` — صالحة للـ `histogram_quantile` في Grafana والتنبيهات مباشرة.

## 5. مثال عميل (ويب)

```js
const samples = { client_type: "web" };

const nav = performance.getEntriesByType("navigation")[0];
if (nav) {
  samples.ttfb_ms = nav.responseStart;
  samples.dom_interactive_ms = nav.domInteractive;
}

new PerformanceObserver((list) => {
  for (const e of list.getEntries()) {
    if (e.entryType === "largest-contentful-paint") samples.lcp_ms = e.startTime;
  }
}).observe({ type: "largest-contentful-paint", buffered: true });

new PerformanceObserver((list) => {
  for (const e of list.getEntries()) {
    if (e.entryType === "layout-shift" && !e.hadRecentInput) samples.cls += e.value;
  }
}).observe({ type: "layout-shift", buffered: true });

navigator.sendBeacon(API_URL + "/rum",
  new Blob([JSON.stringify(samples)], { type: "application/json" }));
```

> عند تفعيل `require_auth` يجب على العميل إرسال الرأس `Authorization: Bearer <RUM_TOKEN>` (مثل `fetch` العادي)، أو يُفضَّل لاحقاً نقطة `/rum` عامة بلا تصويب للمصادقة — تُعدّ ميزة مستقبلية خارج P7.

## 6. ملاحظة النشر

النقطة على منفذ الإدارة المعزول. لإرسال العينات من أجهزة العملاء يجب أن يكون المنفذ **قابلاً للوصول** لهؤلاء العملاء (واجهة `0.0.0.0`) مع `MANAGEMENT_REQUIRE_AUTH=true` (الحيلولة تمنع الواجهة الواسعة بدون مصادقة). في الطوبولوجيا المعزولة بالكامل يمكن إرسال RUM إلى منفذ الإدارة المحلي فقط بواسطة وكلاء داخليين.

## 7. الحماية من تسمم الـ labels

- `client_type` مغلق بثلاث قيم فقط (مفروضة في `registry.go` وفي المعالج — حاجزان).
- القيم المرفوضة تُردّ بخطأ **ولا تُسجّل**؛ لا يُكتب أي زوج label غير معروف.
- الحقول غير المعروفة تُهمل ولا تصبح labels.
- عدد سلاسل الـ timeseries الهرمية محصور: 5 عائلات × 3 قيم = 15 سلسلة + المحاور الثابتة.