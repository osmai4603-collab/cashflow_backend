# وسائط PEP ومغلفات أخطاء RFC 7807 في Go

توضح هذه الوثيقة تصميم وتطبيق **نقطة فرض السياسة (Policy Enforcement Point - PEP)** في خدمات Go لبروتوكولي HTTP و gRPC، مع التركيز على حماية السياق من التصادم، وحراس الفرض الصارم للمنع الافتراضي (Fail-Closed)، واستجابات الأخطاء المعيارية وفق **RFC 7807 Problem Details**.

---

## 1. دور نقطة فرض السياسة (PEP)

تحمي نقطة الفرض (PEP) حدود التطبيق عبر:

1. **اعتراض** الطلبات القادمة قبل وصولها إلى معالجات المجال والمنطق التجاري.
2. **استخراج وتمرير** هوية الفاعل والسياق البيئي عبر السياق.
3. **التنسيق** مع محرك اتخاذ القرار (PDP) ومستودعات البيانات (PIP) لجلب سمات الموارد.
4. **فرض** القرار الحاسم: إما السماح باستكمال المعالجة (`200 OK`) أو قطع الطلب فوراً باستجابة آمنة برمز `403 Forbidden`.

```text
طلب شبكي قادم (HTTP / gRPC)
           │
           ▼
┌──────────────────────────────────────────────────────────┐
│ وسيط المصادقة (PEP AuthN Middleware)                     │
│ 1. التحقق من التوكن / الشهادات / الجلسة                  │
│ 2. تكوين كائن الفاعل (Subject) والبيئة (Environment)     │
│ 3. حقن كائن الهوية في السياق بنوع مفتاح غير مصدّر        │
└──────────────────────────┬───────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────┐
│ حارس التفويض (PEP Authorization Guard)                   │
│ 1. استرجاع الفاعل والبيئة من السياق                      │
│ 2. استدعاء PIP: جلب سمات المورد المستهدف                 │
│ 3. استدعاء PDP: تقييم القواعد بخوارزمية الدمج الحتمية    │
└──────────────────────────┬───────────────────────────────┘
                           │
             ┌─────────────┴─────────────┐
             │                           │
       [قرار بالسماح Permit]       [قرار بالمنع Deny]
             │                           │
             ▼                           ▼
┌─────────────────────────┐ ┌──────────────────────────────┐
│ استدعاء next.ServeHTTP()│ │ إرجاع مشكلة RFC 7807 (403)   │
│ (تنفيذ إجراءات المجال)  │ │ (إنهاء المعالجة فورياً وبأمان)│
└─────────────────────────┘ └──────────────────────────────┘
```

---

## 2. تمرير الهوية عبر السياق بأمان ومنع التصادم (Context Safety)

في Go، تقبل دالة `context.WithValue` النوع `any` للمفتاح والقيمة. استخدام مفاتيح نصية بسيطة (مثل `"subject"`) يؤدي حتماً إلى تصادمات بين الحزم ووسطاء الطرف الثالث.

### المعيار الاصطلاحي المعتمد في Go

تعريف **نوع بنية خاصة غير مصدّرة** مخصصة حصرياً لمفاتيح السياق:

```go
package abac

import "context"

// contextKey نوع غير مصدّر يمنع رياضياً أي تصادم خارج هذه الحزمة
type contextKey struct{}

var (
    subjectKey     = contextKey{}
    environmentKey = contextKey{}
)

// WithSubject يحقن الفاعل الموثق بأمان في السياق
func WithSubject(ctx context.Context, sub Subject) context.Context {
    return context.WithValue(ctx, subjectKey, sub)
}

// GetSubject يسترجع الفاعل الموثق من السياق مع تأكيد النوع
func GetSubject(ctx context.Context) (Subject, bool) {
    sub, ok := ctx.Value(subjectKey).(Subject)
    return sub, ok
}

// WithEnvironment يحقن السياق البيئي في السياق
func WithEnvironment(ctx context.Context, env Environment) context.Context {
    return context.WithValue(ctx, environmentKey, env)
}

// GetEnvironment يسترجع السياق البيئي من السياق مع تأكيد النوع
func GetEnvironment(ctx context.Context) (Environment, bool) {
    env, ok := ctx.Value(environmentKey).(Environment)
    return env, ok
}
```

---

## 3. مغلفات أخطاء المنع القياسية وفق RFC 7807

عندما يفشل فحص الصلاحية، يجب أن تُرجع الواجهة البرمجية استجابة مهيكلة ومعيارية بصيغة JSON متوافقة تماماً مع معيار **RFC 7807** (*Problem Details for HTTP APIs*).

### 3.1 هيكل حمولة تفاصيل المشكلة (Problem Details Schema)

```go
package abac

import (
    "encoding/json"
    "net/http"
)

// ProblemDetails يمثل حمولة استجابة مشكلة RFC 7807
type ProblemDetails struct {
    Type     string `json:"type"`               // رابط URI يحدد نوع المشكلة البرمجية
    Title    string `json:"title"`              // ملخص مقروء وموجز عن الخطأ
    Status   int    `json:"status"`             // رمز حالة HTTP (403)
    Detail   string `json:"detail"`             // شرح عام ومحدد لحدوث هذه الحالة
    Instance string `json:"instance,omitempty"` // مسار الطلب الشبكي الذي وقعت عنده المشكلة
}

// WriteForbiddenProblem يرسل استجابة معيارية برمز 403 Forbidden ونوع محتوى problem+json
func WriteForbiddenProblem(w http.ResponseWriter, r *http.Request, detail string) {
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(http.StatusForbidden)
    _ = json.NewEncoder(w).Encode(ProblemDetails{
        Type:     "https://example.com/errors/forbidden",
        Title:    "Forbidden",
        Status:   http.StatusForbidden,
        Detail:   detail,
        Instance: r.URL.Path,
    })
}
```

### 3.2 منع تسريب معلومات البنية الأمنية (Information Disclosure Prevention)

> [!CAUTION]
> **إياك وتسريب أسماء قواعد السياسات الداخلية**:
> يمكن للمهاجم استكشاف البنية الأمنية إذا كانت رسالة الخطأ توضح:
> `"تم المنع: تصريح الفاعل 2 أقل من حساسية المورد 5، وقسم المحاسبة لا يطابق قسم المالية"`.
>
> في البيئات الإنتاجية:
>
> - **الاستجابة الخارجية للعميل (RFC 7807)**: `"تم رفض الوصول: صلاحيات غير كافية للوصول لهذا المورد"`.
> - **سجل التدقيق الداخلي (`log/slog`)**: سبب المنع التفصيلي متضمناً معرّف القاعدة التي رفضت الطلب، ومستوى التصريح، وتطابق الأقسام، ومعرّف المستأجر.

---

## 4. تطبيق وسيط HTTP PEP العام القابل لإعادة الاستخدام

فيما يلي وسيط HTTP نموذجي يفرض سياسات ABAC بنمط الفشل السريع (Fail-Closed):

```go
type ResourceExtractor func(r *http.Request) (Resource, error)

func RequireABAC(pdp *Engine, verb string, extractResource ResourceExtractor) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. استخراج الفاعل من السياق (المحقون سابقاً عبر وسيط AuthN)
            subject, ok := GetSubject(r.Context())
            if !ok {
                WriteForbiddenProblem(w, r, "تم رفض الوصول: سياق فاعل غير مصادق عليه")
                return
            }

            // 2. استخراج المورد المستهدف
            resource, err := extractResource(r)
            if err != nil {
                WriteForbiddenProblem(w, r, "تم رفض الوصول: تعذر استخراج المورد المستهدف")
                return
            }

            // 3. استخراج أو تكوين السياق البيئي
            env, ok := GetEnvironment(r.Context())
            if !ok {
                env = Environment{
                    RequestTime: time.Now(),
                    ClientIP:    r.RemoteAddr,
                }
            }

            // 4. بناء سياق التقييم الرباعي
            evalCtx := EvaluationContext{
                Subject:     subject,
                Resource:    resource,
                Action:      Action{Verb: verb, Method: r.Method},
                Environment: env,
            }

            // 5. تقييم القرار عبر محرك PDP
            decision, _ := pdp.Evaluate(evalCtx)
            if decision != DecisionPermit {
                // الفشل الآمن: إرجاع 403 فوراً دون تسريب أسماء القواعد الداخلية
                WriteForbiddenProblem(w, r, "تم رفض الوصول: لا تملك الصلاحيات الكافية لتنفيذ هذا الإجراء على المورد")
                return
            }

            // 6. منح الوصول والتسليم للمعالج التالي في خط الأنابيب
            next.ServeHTTP(w, r)
        })
    }
}
```
