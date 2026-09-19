# مسارات التدقيق الأمني، المراقبة، واستراتيجيات إلغاء الصلاحيات في Go

التحكم بالوصول ليس فحصاً استاتيكياً منفرداً، بل يتطلب مسؤولة ومتابعة دورية مستمرة. توضح هذه الوثيقة سجلات التدقيق الأمني المهيكلة، ومقاييس المراقبة اللحظية، وآليات إبطال الجلسات والصلاحيات في لغة Go.

---

## 1. سجلات التدقيق الأمني غير القابلة للتلاعب عبر `log/slog`

يجب أن ينتج عن كل قرار تفويض مسار تدقيق غير قابل للتعديل وجاهز للابتلاع المباشر في أنظمة SIEM (مثل Elastic, Splunk, CloudWatch).

### سمات حدث التدقيق الأمني (Audit Event Attributes)

- `timestamp`: الطابع الزمني بصيغة RFC 3339 nano.
- `event_type`: نوع الحدث `authz.decision`.
- `subject_id`: المعرف الفريد للفاعل الموثق.
- `tenant_id`: معرف المنظمة أو الشركة المستأجرة.
- `roles`: لقطة للأدوار المقيمة لحظة القرار.
- `permission`: الصلاحية المطلوبة للعملية.
- `decision`: القرار النهائي `permit` (سماح) أو `deny` (منع).
- `reason`: كود سبب مقروء آلياً (مثل `role_unmatched`, `ssd_conflict`, `token_expired`).
- `client_ip`: عنوان IP للمتصل.
- `latency_us`: زمن تقييم المحرك بالميكروثانية.

```go
func LogAudit(logger *slog.Logger, subject Subject, perm Permission, decision string, reason string, latency time.Duration, ip, path string) {
 level := slog.LevelInfo
 if decision == "deny" {
  level = slog.LevelWarn
 }

 logger.LogAttrs(context.Background(), level, "authz_decision",
  slog.String("event_type", "authz.decision"),
  slog.String("subject_id", subject.ID),
  slog.String("tenant_id", subject.TenantID),
  slog.Any("roles", subject.Roles),
  slog.String("required_permission", string(perm)),
  slog.String("decision", decision),
  slog.String("reason", reason),
  slog.String("client_ip", ip),
  slog.String("path", path),
  slog.Int64("latency_us", latency.Microseconds()),
 )
}
```

---

## 2. المراقبة اللحظية ومقاييس OpenMetrics

تساعد مراقبة عمليات التفويض على كشف هجمات التخمين ومحاولات تصعيد الامتيازات (Privilege Escalation):

### المقاييس الجوهرية

1. `authz_evaluations_total{decision="permit|deny", permission="...", tenant="..."}`: عداد يتتبع قرارات الوصول مقسمة بالقرار والمستأجر والصلاحية.
2. `authz_evaluation_duration_seconds`: مدرج تكراري لزمن اتخاذ القرار في PDP (مستوى الخدمة المستهدف SLO: p99 < 500µs).
3. `authz_active_roles_gauge`: إجمالي عدد الأدوار النشطة المسجلة في محرك الذاكرة الحية.

---

## 3. استراتيجيات إلغاء الصلاحيات وإبطال الجلسات

أحد التحديات المعمارية في الأنظمة الموزعة هو **التأخر الزمني لتطبيق إلغاء الامتيازات (Privilege Latency)**: عندما يُلغى دور مستخدم في قاعدة البيانات، متى يتوقف رمزه الحالي عن العمل؟

### الاستراتيجية 1: رموز وصول قصيرة الأجل (موصى بها)

- إصدار رموز JWT عديمة الحالة بفترة صلاحية قصيرة (من 5 إلى 15 دقيقة).
- عند تغيير الأدوار، يمتد تأثير الدور القديم لـ 15 دقيقة على الأكثر.
- تقوم رموز التجديد (Stateful Refresh Tokens) بالاستعلام من قاعدة البيانات لجلب أحدث الأدوار عند كل عملية تجديد.

### الاستراتيجية 2: إصدار التوكن للفاعل (Subject Token Versioning)

- حفظ حقل `token_version` (رقم صحيح) في سجل المستخدم في قاعدة البيانات.
- تضمين `token_version: 3` في ادعاءات رمز JWT.
- الاحتفاظ بخريطة سريعة في Redis أو الذاكرة: `user_id -> current_token_version`.
- عندما يقوم المدير بإلغاء دور، يتم زيادة `token_version` في قاعدة البيانات وتحديث الكاش.
- تقارن نقطة الفرض PEP إصدار التوكن مع القيمة في الكاش؛ وإذا لم تتطابق، ترفض الطلب فوراً بـ `401 Unauthorized`.

```go
type ClaimsWithVersion struct {
 UserID       string `json:"sub"`
 Roles        []Role `json:"roles"`
 TokenVersion int    `json:"v"`
}

func ValidateTokenVersion(cachedVersion int, claims ClaimsWithVersion) bool {
 return claims.TokenVersion == cachedVersion
}
```

---

## 4. مكافحة تراكم الامتيازات (Periodic Access Reviews)

مع مرور الوقت، ينتقل الموظفون بين الأقسام وتتراكم لديهم الأدوار القديمة دون إلغاء:

- الاحتفاظ بجدول تدقيق: `role_assignments(user_id, role, granted_at, expires_at, granted_by)`.
- فرض مدة صلاحية زمنية إلزامية لكل دور (مثال: 90 يوماً قابلة للتجديد بموافقة).
- تقديم تقارير دورية آلية للحسابات الخاملة التي تمتلك أدواراً عالية الامتيازات لمراجعتها وإلغائها.
