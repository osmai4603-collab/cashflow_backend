# سياسة ترويسات الأمان ومشاركة الموارد عبر الأصول (Security Headers & CORS)

يجب على خدمات Go الإنتاجية التي تقدم واجهات برمجة تطبيقات REST إرفاق ترويسات الأمان بجميع استجابات HTTP لحماية العملاء ضد هجمات البرمجة عبر المواقع (XSS)، وخطف النقرات (Clickjacking)، والتخمين الخاطئ لأنواع MIME، وهجمات الوسيط (MITM).

---

## 1. ترويسات أمان HTTP الإلزامية (Mandatory HTTP Security Headers)

| الترويسة (Header) | القيمة الموصى بها في الإنتاج | الغرض الأمني |
| :--- | :--- | :--- |
| `X-Content-Type-Options` | `nosniff` | منع المتصفح من تخمين نوع المحتوى (MIME-type sniffing) |
| `X-Frame-Options` | `DENY` (أو `SAMEORIGIN`) | منع خطف النقرات عبر حظر التضمين داخل إطارات `iframe` |
| `X-XSS-Protection` | `1; mode=block` | تفعيل مرشح الحماية القديم للمتصفحات السابقة |
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains; preload` | فرض بروتوكول HTTPS الصارم لمدة عامين |
| `Content-Security-Policy` | `default-src 'self'; frame-ancestors 'none';` | تقييد مصادر الموارد وحظر التضمين تماماً |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | تجريد المسار والاستعلام من ترويسة الإحالة عبر النطاقات |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=()` | تعطيل واجهات عتاد المتصفح الحساسة |

---

## 2. سياسة مشاركة الموارد عبر الأصول (CORS Policy)

يجب ضبط إعدادات CORS بصرامة استناداً إلى بيئة التشغيل:

### محظور تماماً في البيئة الإنتاجية

- استخدام `Access-Control-Allow-Origin: *` عند السماح ببيانات الاعتماد (`cookies`، أو ترويسة `Authorization`).
- الانعكاس الديناميكي لترويسة `Origin` دون التحقق من القائمة البيضاء المعتمدة (Whitelist).

### التكوين الإنتاجي الموصى به

```go
// CORSConfig يحدد معاملات CORS في البيئة الإنتاجية.
type CORSConfig struct {
 AllowedOrigins   []string
 AllowedMethods   []string
 AllowedHeaders   []string
 ExposedHeaders   []string
 AllowCredentials bool
 MaxAgeSeconds    int
}

// الإعدادات الافتراضية الإنتاجية
var DefaultCORS = CORSConfig{
 AllowedOrigins: []string{
  "https://app.example.com",
  "https://admin.example.com",
 },
 AllowedMethods: []string{
  "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
 },
 AllowedHeaders: []string{
  "Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID",
 },
 ExposedHeaders: []string{
  "X-Request-ID", "Retry-After",
 },
 AllowCredentials: true,
 MaxAgeSeconds:    86400, // تخزين مؤقت للتحقق المسبق (Preflight) لمدة 24 ساعة
}
```
