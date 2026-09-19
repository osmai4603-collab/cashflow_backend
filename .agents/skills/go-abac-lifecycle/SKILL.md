---
name: go-abac-lifecycle
description: "معمارية إنتاجية محايدة للأطر لإدارة دورة حياة التحكم بالوصول القائم على السمات (ABAC) في تطبيقات Go. تغطي معايير NIST SP 800-162، نمذجة السمات رباعية الأبعاد (Subject, Resource, Action, Environment)، الإثراء الديناميكي عبر PIP، خوارزميات دمج القرارات في PDP (مثل Deny-Overrides و Permit-Overrides)، نقل الهوية المحمي عبر السياق، حراس وسائط PEP، مغلفات مشاكل RFC 7807، وسجلات التدقيق غير القابلة للتلاعب."
---

# مهارة معمارية دورة حياة التحكم بالوصول القائم على السمات (ABAC) في Go

تحدد هذه المهارة المعمارية الهندسية والإنتاجية المحايدة تماماً لأطر العمل لإدارة **دورة حياة التحكم بالوصول القائم على السمات (Attribute-Based Access Control - ABAC)** في خدمات وتطبيقات **Go**. المعمارية مفصولة كلياً عن أي مجال تجاري محدد، مما يجعلها قابلة للتطبيق المباشر على الخدمات المصغرة، والأنظمة الأحادية، وواجهات REST API، وخوادم gRPC، والأنظمة الموزعة السحابية.

تجمع هذه المعمارية بين:

- معايير Go الأمنية الرسمية من [go.dev/doc/security](https://go.dev/doc/security) و [go.dev/security/best-practices](https://go.dev/security/best-practices).
- المعايير الدولية الرسمية لإدارة الوصول من **NIST SP 800-162** (*دليل تعريف واعتبارات التحكم بالوصول القائم على السمات*) ومواصفات **OASIS XACML 3.0**.
- أنماط فصل التفويض المعتمدة في **Kubernetes** (`k8s.io/apiserver`) ومحرك **Google CEL** (`google/cel-go`).

---

## المبادئ الأمنية للبيئات الإنتاجية (Production Principles)

1. **المنع الافتراضي الصارم (Fail-Closed / Default Deny)**:
   الوصول ممنوع افتراضياً. لا يُسمح بأي إجراء إلا إذا وجد تصريح صريح ومطابق من السياسات. عند فقدان أي سمة، أو فشل استعلام PIP، أو عدم وجود مصادقة، يجب أن تنتهي النتيجة فوراً إلى **المنع (Deny)**.

2. **رباعية السمات المتزامنة (The 4-Dimensional Attribute Quadruple)**:
   يتم تقييم أربعة أبعاد لحظية في كل قرار تفويض:
   - **الفاعل ($S$ - Subject)**: سمات المتصل الموثق (ID, TenantID, Roles, Clearance, Department, Attributes map).
   - **المورد ($R$ - Resource)**: سمات الكيان المستهدف (ID, Type, OwnerID, TenantID, Sensitivity, Status, Attributes map).
   - **الإجراء ($A$ - Action)**: سمات العملية المطلوبة (Verb, Method, TargetScope).
   - **البيئة ($E$ - Environment)**: السياق المحيط لحظة الطلب (RequestTime, ClientIP, NetworkZone, TLSVersion, SecurityPosture).

3. **الفصل المعماري لمسؤوليات XACML / NIST**:
   - **نقطة إدارة واسترجاع السياسات (PAP / PRP)**: تصريح واختبار وتخزين السياسات (كدوال Go نقية أو قواعد CEL أو JSON).
   - **نقطة جمع المعلومات (PIP)**: استرجاع سمات المورد والبيئة ديناميكياً من قواعد البيانات ومخازن الذاكرة المؤقتة.
   - **نقطة اتخاذ القرار (PDP)**: تقييم السمات مقابل السياسات المعتمدة باستخدام خوارزميات دمج حتمية (`Deny-Overrides`, `Permit-Overrides`).
   - **نقطة فرض السياسة (PEP)**: اعتراض الطلبات، وحقن السياق بأمان، وتطبيق قرار PDP عند حدود البروتوكول (HTTP/gRPC).

4. **حماية السياق عبر أنواع غير مصدّرة (Unexported Context Keys)**:
   يجب تمرير بيانات الفاعل والسمات الموثقة عبر `context.Context` حصراً باستخدام أنواع بنى خاصة غير مصدّرة (`type contextKey struct{}`) لمنع أي تصادم بين الحزم.

5. **تقييم فائق السرعة متوافق مع التزامن (Thread-Safe Evaluation)**:
   يقع تقييم السياسات في المسار الحرج للطلبات. يجب أن يعمل محرك PDP بدون أي تنازع أقفال في مسارات القراءة (باستخدام `sync.RWMutex` أو `atomic.Pointer`) وبأقل استهلاك ممكن للذاكرة.

6. **منع تسريب تفاصيل البنية عند الرفض (Information Disclosure Prevention)**:
   استجابات المنع يجب أن تتبع معيار **RFC 7807 (Problem Details)** برمز الحالة `403 Forbidden`، دون تسريب أسماء القواعد أو حقول الجداول الداخلية للعميل غير الموثوق.

7. **سجلات تدقيق غير قابلة للتلاعب ومراقبة مستمرة**:
   كل قرار تفويض (سواء بالسماح أو المنع) يجب أن يولد سجلاً مهيكلاً (`log/slog`) ويصدر عدادات آنية (Prometheus Metrics).

---

## المخطط المعماري: دورة حياة ABAC السداسية/السباعية

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 1: نمذجة السمات (Modeling)       تحديد مخططات الفاعل والمورد والإجراء والبيئة             │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 2: كتابة السياسات (PAP)          تعريف القواعد والشروط وخوارزميات الدمج الحتمية           │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 3: اعتراض الطلب والسياق (PEP)     التحقق من الهوية، وحقن Subject والبيئة في السياق       │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 4: الإثراء الديناميكي (PIP)       استرجاع بيانات وسمات المورد من قاعدة البيانات أو الكاش   │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 5: محرك التقييم (PDP Engine)     تنفيذ خوارزمية الدمج (Deny-Overrides / Permit-Overrides)│
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 6: الفرض والرفض الآمن (PEP)      مسموح: تمرير المعالجة (200) | ممنوع: مشكلة RFC 7807 (403)│
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 7: التدقيق والمراقبة             سجلات مهيكلة عبر slog، وتصدير مقاييس Prometheus و SIEM  │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## الدليل الهندسي للمراحل (Phase-by-Phase Engineering Guide)

### المرحلة 1: نمذجة السمات بشكل محايد للمجال

تمثيل الأبعاد الأربعة باستخدام هياكل Go محكمة الأنواع مع خرائط سمات ديناميكية:

```go
package abac

import "time"

// Subject يمثل الفاعل الموثق الذي يطلب الوصول
type Subject struct {
    ID         string            `json:"id"`
    TenantID   string            `json:"tenant_id,omitempty"`
    Roles      []string          `json:"roles,omitempty"`
    Department string            `json:"department,omitempty"`
    Clearance  int               `json:"clearance,omitempty"`
    Attributes map[string]any    `json:"attributes,omitempty"`
}

// Resource يمثل الكيان المستهدف المطلوب الوصول إليه أو تعديله
type Resource struct {
    ID          string            `json:"id"`
    Type        string            `json:"type"` // مثال: "document", "transaction", "invoice"
    OwnerID     string            `json:"owner_id,omitempty"`
    TenantID    string            `json:"tenant_id,omitempty"`
    Department  string            `json:"department,omitempty"`
    Sensitivity int               `json:"sensitivity,omitempty"`
    Status      string            `json:"status,omitempty"`
    Attributes  map[string]any    `json:"attributes,omitempty"`
}

// Action يمثل العملية المطلوب تنفيذها على المورد
type Action struct {
    Verb   string `json:"verb"`             // مثال: "read", "create", "update", "delete", "approve"
    Method string `json:"method,omitempty"` // أسلوب HTTP أو اسم دالة RPC
}

// Environment يمثل السياق البيئي والزمني المحيط بلحظة التقييم
type Environment struct {
    RequestTime time.Time         `json:"request_time"`
    ClientIP    string            `json:"client_ip,omitempty"`
    NetworkZone string            `json:"network_zone,omitempty"`
    Attributes  map[string]any    `json:"attributes,omitempty"`
}

// EvaluationContext سياق التقييم الجامع للرباعية الكاملة
type EvaluationContext struct {
    Subject     Subject
    Resource    Resource
    Action      Action
    Environment Environment
}
```

### المرحلة 2: كتابة السياسات وخوارزميات الدمج (PAP/PDP)

تحقق كل قاعدة سياسة واجهة `PolicyRule`، ويقوم محرك PDP بدمج القواعد وفق خوارزمية محددة:

```go
package abac

type Decision int

const (
    DecisionNotApplicable Decision = iota
    DecisionPermit
    DecisionDeny
)

type PolicyRule interface {
    ID() string
    Description() string
    Target(ctx EvaluationContext) bool
    Evaluate(ctx EvaluationContext) (Decision, string)
}

type CombiningAlgorithm int

const (
    DenyOverrides CombiningAlgorithm = iota
    PermitOverrides
    FirstApplicable
)
```

### المرحلة 3: اعتراض الطلب وحقن السياق بأمان (PEP)

يُحظر استخدام السلاسل النصية كمفاتيح للسياق؛ يجب دائماً تعريف نوع خاص غير مصدّر:

```go
package abac

import "context"

type contextKey struct{}

var subjectContextKey = contextKey{}

func WithSubject(ctx context.Context, sub Subject) context.Context {
    return context.WithValue(ctx, subjectContextKey, sub)
}

func GetSubject(ctx context.Context) (Subject, bool) {
    sub, ok := ctx.Value(subjectContextKey).(Subject)
    return sub, ok
}
```

### المرحلة 4: الإثراء الديناميكي لنقطة المعلومات (PIP)

في التطبيقات الحقيقية، لا يجوز الوثوق بسمات المورد القادمة من العميل؛ بل يجب جلبها من التخزين الدائم:

```go
type PIPResolver interface {
    ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error)
}
```

### المرحلة 5: محرك التقييم المتزامن عالي الأداء (PDP Engine)

يدير المحرك قواعد السياسات بأمان تحت القراءات المتزامنة المكثفة:

```go
type Engine struct {
    mu         sync.RWMutex
    algorithm  CombiningAlgorithm
    rules      []PolicyRule
    logger     *slog.Logger
}

func (e *Engine) Evaluate(ctx EvaluationContext) (Decision, string) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    switch e.algorithm {
    case DenyOverrides:
        // إذا رفضت أي قاعدة سارية، فالقرار الفوري هو المنع Deny
        // ويجب أن توجد قاعدة سارية واحدة على الأقل تسمح، وإلا فالمنع الافتراضي
    }
}
```

### المرحلة 6: الفرض (PEP) ومغلفات أخطاء RFC 7807

عند فشل فحص الوصول، يتم إرجاع استجابة قياسية وفق RFC 7807:

```go
type ProblemDetails struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail"`
    Instance string `json:"instance"`
}

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

### المرحلة 7: التدقيق الأمني والمراقبة (Audit Logging)

تسجيل أحداث التدقيق المهيكلة دون تسريب أي أسرار للمستخدمين:

```go
logger.Warn("ABAC evaluation denied",
    slog.String("rule_id", failedRuleID),
    slog.String("subject_id", ctx.Subject.ID),
    slog.String("tenant_id", ctx.Subject.TenantID),
    slog.String("resource_type", ctx.Resource.Type),
    slog.String("resource_id", ctx.Resource.ID),
    slog.String("action", ctx.Action.Verb),
    slog.String("client_ip", ctx.Environment.ClientIP),
    slog.Duration("eval_latency", elapsed),
)
```

---

## قائمة التحقق الأمني للإنتاج (Production Checklist)

- [ ] **المنع الافتراضي (Fail-Closed)**: أي طلب غير مغطى بقاعدة، أو قائمة سياسات فارغة، أو خطأ تقييم ينتهي حتماً بـ `Deny`.
- [ ] **الحماية من ثغرات BOLA/IDOR**: كافة العمليات المقيدة بالمستأجر تفرض مطابقة `Subject.TenantID == Resource.TenantID`.
- [ ] **تحديث السياسات بأمان متزامن**: تحديث ذاكرة السياسات يستخدم `sync.RWMutex` أو `atomic.Pointer`.
- [ ] **التحقق من عدم وجود سباق بيانات**: يجب أن تجتاز جميع الحزم أمر `go test -race ./...`.
- [ ] **عدم تصادم مفاتيح السياق**: قيم السياق تُحفظ وتُسترجع عبر أنواع بنى خاصة غير مصدّرة (`type contextKey struct{}`).
- [ ] **مطابقة RFC 7807**: استجابات الرفض ترجع برمز `403 Forbidden` ونوع المحتوى `application/problem+json`.
- [ ] **منع تسريب المعلومات**: تفاصيل الأخطاء لا تكشف أسماء القواعد أو حقول الجداول الداخلية للعملاء الخارجيين.
- [ ] **سجل التدقيق والتكامل مع SIEM**: جميع القرارات (السماح والمنع) تنتج سجلات مهيكلة عبر `log/slog`.
