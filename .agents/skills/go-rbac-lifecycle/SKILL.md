---
name: go-rbac-lifecycle
description: "معمارية إنتاجية محايدة للأطر لإدارة دورة حياة التحكم بالوصول القائم على الأدوار (RBAC) في تطبيقات Go. تغطي معايير NIST/ANSI RBAC (الأساسي، الهرمي، والقيود SSD/DSD)، نقل الهوية المحمي عبر السياق، التخزين المؤقت للسياسات بدون أقفال وعبر RWMutex، التحديث اللحظي للسياسات، حراس PEP بنمط المنع الافتراضي، تفاصيل مشاكل RFC 7807، وسجلات التدقيق الأمني غير القابلة للتلاعب."
---

# مهارة معمارية دورة حياة التحكم بالوصول القائم على الأدوار (RBAC) في Go

تحدد هذه المهارة المعمارية الهندسية والإنتاجية الشاملة والمحايدة تماماً لأطر العمل لإدارة **دورة حياة التحكم بالوصول القائم على الأدوار (Role-Based Access Control - RBAC)** في خدمات وتطبيقات **Go**. المعمارية مفصولة كلياً عن أي مجال تجاري محدد، مما يجعلها قابلة للتطبيق المباشر على الخدمات المصغرة، والأنظمة الأحادية، وواجهات REST API، وخوادم gRPC، والأنظمة الموزعة السحابية.

تجمع هذه المعمارية بين:

- معايير Go الأمنية الرسمية من [go.dev/doc/security](https://go.dev/doc/security) و [go.dev/security/best-practices](https://go.dev/security/best-practices).
- المعايير الدولية الرسمية لإدارة الوصول من **NIST SP 800-21d** و **ANSI/INCITS 359-2012** (النماذج: الأساسي، الهرمي، قيود SSD و DSD، والنموذج المتناظر).
- أنماط فصل التفويض المعتمدة في **Kubernetes** (`k8s.io/apiserver`).

---

## المبادئ الأمنية للبيئات الإنتاجية (Production Principles)

1. **المنع الافتراضي الصارم (Fail-Closed / Default Deny)**:
   الوصول ممنوع افتراضياً. لا يُسمح بأي إجراء إلا إذا وجد تصريح صريح ومطابق من خلال الأدوار النشطة أو الموروثة. في حال فشل البحث عن الدور، أو تشوه التوكن، أو حدوث خطأ داخلي، يجب أن تنتهي النتيجة فوراً إلى **المنع (Deny)**.

2. **الفصل المعماري لمسؤوليات الأمان**:
   - **الهوية والمصادقة (AuthN)**: التحقق من *مَن* هو المتصل (`Subject`).
   - **التفويض (AuthZ / PDP)**: تقييم *ما الذي* يُسمح للفاعل بتنفيذه بناءً على الأدوار والصلاحيات الفعالة الموروثة.
   - **الفرض (PEP)**: اعتراض الطلب وتمريره أو منعه عند حدود البروتوكول (HTTP/gRPC/CLI).
   - **طبقة المجال والتخزين**: تطبيق القيود الدقيقة على مستوى الكائن وعزل المستأجر (Tenant Isolation).

3. **حماية السياق عبر أنواع غير مصدّرة (Unexported Context Keys)**:
   يجب تمرير بيانات الفاعل والأدوار النشطة عبر `context.Context` حصراً باستخدام أنواع بنى خاصة غير مصدّرة (`type contextKey struct{}`) لمنع أي تصادم أو تلاعب بين الحزم البرمجية.

4. **التخزين المؤقت في الذاكرة عالي الإنتاجية والتزامن**:
   يتم تقييم الصلاحيات مع كل طلب قادم ويجب أن يُنفّذ في زمن يقل عن الميكروثانية. يعيش مخزن السياسات وشجرة وراثة الأدوار في الذاكرة المحمية بـ `sync.RWMutex` أو `atomic.Pointer` لدعم التحديث اللحظي بدون توقف (Hot-Reloading).

5. **فرض الفصل بين الواجبات (Separation of Duties - SSD & DSD)**:
   - **الفصل الاستاتيكي (SSD)**: منع التوليفات الخطرة عند إسناد الأدوار (مثال: لا يجوز للمستخدم الجمع بين دور `منشئ الفواتير` ودور `معتمد الفواتير`).
   - **الفصل الديناميكي (DSD)**: منع تفعيل أدوار متعارضة معاً في نفس الجلسة أو العملية التشغيلية.

6. **منع تسريب تفاصيل البنية عند الرفض (Information Disclosure Prevention)**:
   استجابات الرفض تتبع معيار **RFC 7807 (Problem Details)** برمز الحالة `403 Forbidden`، دون تسريب أسماء الصلاحيات الناقصة أو مخططات قاعدة البيانات.

7. **سجلات تدقيق غير قابلة للتلاعب ومراقبة مستمرة**:
   كل قرار تفويض (وخاصة حالات الرفض) يجب أن يولد سجلاً مهيكلاً عبر `log/slog` ويصدر عدادات لحظية عبر Prometheus.

---

## المخطط المعماري: دورة حياة RBAC السبعية

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 1: النمذجة (Modeling)        تعريف الصلاحيات (resource:action)، والأدوار، ورسم DAG للوراثة │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 2: التزويد (Provisioning)    التخزين الدائم (SQL/KV) + التحميل للذاكرة الحية (RWMutex)    │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 3: الإسناد (Assignment)      ربط الفاعل بالأدوار مع تقييد المستأجر وفحص تعارضات SSD      │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 4: الاعتراض والسياق (PEP)    فحص التوكن القادم، وتجسيد الفاعل، وحقنه بأمان في السياق     │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 5: التقييم (PDP Engine)      اجتياز رسم الوراثة DAG، حساب الصلاحيات الفعالة، والتحقق من DSD│
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 6: الفرض (PEP Enforcement)   مسموح: التمرير للمعالج (200) | ممنوع: إرجاع مشكلة RFC 7807   │
└──────────────────────────────────────────────┬───────────────────────────────────────────────────┘
                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ المرحلة 7: التدقيق والإلغاء (Audit)   تصدير سجلات slog ومقاييس OpenMetrics، وإدارة إبطال الجلسات   │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## الدليل الهندسي للمراحل (Phase-by-Phase Engineering Guide)

### المرحلة 1: النمذجة المحايدة للمجال

تمثيل الصلاحيات باستخدام ثوابت مهيكلة تتبع الاصطلاح المعياري `resource:action`:

```go
package rbac

type Permission string

type Role string

type Subject struct {
    ID       string            `json:"id"`
    TenantID string            `json:"tenant_id,omitempty"` // للأنظمة متعددة المستأجرين
    Roles    []Role            `json:"roles"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

### المرحلة 2: محرك الذاكرة الحية مع الرسم البياني الهرمي الموجه (DAG)

يحسب محرك RBAC الصلاحيات الفعالة عبر اجتياز رسم بياني موجه غير دائري (DAG) يمثل وراثة الأدوار ($r_{senior} \succeq r_{junior}$):

```go
package rbac

import (
    "fmt"
    "sync"
)

type Engine struct {
    mu           sync.RWMutex
    rolePerms    map[Role]map[Permission]bool
    inheritance  map[Role][]Role
    ssdConflicts map[Role][]Role
}

func NewEngine() *Engine {
    return &Engine{
        rolePerms:    make(map[Role]map[Permission]bool),
        inheritance:  make(map[Role][]Role),
        ssdConflicts: make(map[Role][]Role),
    }
}

// DefineRole يربط الصلاحيات المباشرة بالدور
func (e *Engine) DefineRole(role Role, perms ...Permission) {
    e.mu.Lock()
    defer e.mu.Unlock()

    if e.rolePerms[role] == nil {
        e.rolePerms[role] = make(map[Permission]bool)
    }
    for _, p := range perms {
        e.rolePerms[role][p] = true
    }
}

// AddInheritance يحدد أن seniorRole يرث كافة صلاحيات juniorRole
func (e *Engine) AddInheritance(senior Role, junior Role) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.inheritance[senior] = append(e.inheritance[senior], junior)
}

// ResolveEffectivePermissions يجتاز رسم الوراثة لحساب كافة الصلاحيات الممنوحة
func (e *Engine) ResolveEffectivePermissions(roles []Role) map[Permission]bool {
    e.mu.RLock()
    defer e.mu.RUnlock()

    effective := make(map[Permission]bool)
    visited := make(map[Role]bool)

    var walk func(r Role)
    walk = func(r Role) {
        if visited[r] {
            return
        }
        visited[r] = true

        for p := range e.rolePerms[r] {
            effective[p] = true
        }
        for _, parent := range e.inheritance[r] {
            walk(parent)
        }
    }

    for _, r := range roles {
        walk(r)
    }
    return effective
}

// HasPermission يفحص امتلاك أدوار الفاعل للصلاحية المطلوبة (منع افتراضي)
func (e *Engine) HasPermission(roles []Role, required Permission) bool {
    perms := e.ResolveEffectivePermissions(roles)
    return perms[required]
}
```

### المرحلة 3: فرض الفصل الاستاتيكي بين الواجبات (SSD)

التحقق من إسناد الأدوار لمنع التوليفات المتعارضة:

```go
func (e *Engine) RegisterSSDConflict(roleA, roleB Role) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.ssdConflicts[roleA] = append(e.ssdConflicts[roleA], roleB)
    e.ssdConflicts[roleB] = append(e.ssdConflicts[roleB], roleA)
}

func (e *Engine) ValidateAssignment(roles []Role) error {
    e.mu.RLock()
    defer e.mu.RUnlock()

    assigned := make(map[Role]bool, len(roles))
    for _, r := range roles {
        assigned[r] = true
    }

    for _, r := range roles {
        for _, conflict := range e.ssdConflicts[r] {
            if assigned[conflict] {
                return fmt.Errorf("مخالفة لقيد SSD: لا يمكن الجمع بين الدورين %q و %q", r, conflict)
            }
        }
    }
    return nil
}
```

### المرحلة 4: نمط تمرير السياق المحمي (Context Propagation)

تغليف مفاتيح السياق داخل الحزمة بنوع غير مصدّر:

```go
package rbac

import "context"

type contextKey struct{}

var subjectKey = contextKey{}

func WithSubject(ctx context.Context, sub Subject) context.Context {
    return context.WithValue(ctx, subjectKey, sub)
}

func SubjectFromContext(ctx context.Context) (Subject, bool) {
    sub, ok := ctx.Value(subjectKey).(Subject)
    return sub, ok
}
```

### المرحلتان 5 و 6: وسيط الفرض PEP واستجابات RFC 7807

فرض الصلاحيات عند مستوى المسار الشبكي:

```go
package rbac

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "time"
)

type ProblemDetails struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail"`
    Instance string `json:"instance"`
}

func writeRFC7807(w http.ResponseWriter, status int, title, detail, instance string) {
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(ProblemDetails{
        Type:     "https://golang.org/errors/forbidden",
        Title:    title,
        Status:   status,
        Detail:   detail,
        Instance: instance,
    })
}

// RequirePermission ينشئ وسيط HTTP PEP لحراسة المسار
func RequirePermission(engine *Engine, logger *slog.Logger, required Permission) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            subject, ok := SubjectFromContext(r.Context())
            if !ok {
                logger.Warn("تم حظر طلب غير مصادق عليه عند نقطة PEP", "path", r.URL.Path)
                writeRFC7807(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
                return
            }

            allowed := engine.HasPermission(subject.Roles, required)
            duration := time.Since(start)

            if !allowed {
                logger.Warn("تم رفض وصول RBAC",
                    "subject_id", subject.ID,
                    "tenant_id", subject.TenantID,
                    "roles", subject.Roles,
                    "required_permission", required,
                    "path", r.URL.Path,
                    "duration_us", duration.Microseconds(),
                )
                writeRFC7807(w, http.StatusForbidden, "Forbidden", "صلاحيات غير كافية", r.URL.Path)
                return
            }

            logger.Info("تم السماح بالوصول",
                "subject_id", subject.ID,
                "permission", required,
                "duration_us", duration.Microseconds(),
            )

            next.ServeHTTP(w, r)
        })
    }
}
```

### المرحلة 7: مسار التدقيق وإلغاء الجلسات (Audit & Revocation)

- **حمولة سجلات التدقيق المهيكلة**: توثيق `subject_id`, `tenant_id`, `roles`, `required_permission`, `client_ip`, و `decision`.
- **إبطال الرموز عند تغيير الأدوار**: عند تعديل أدوار المستخدم، يجب ألا تمنح رموز الوصول القديمة صلاحيات ملغاة:
  1. **رموز وصول قصيرة الأجل** ($\le 15\text{ دقيقة}$) مقترنة برموز تجديد مشفرة.
  2. **فحص إصدار التوكن (Token Versioning)**: تضمين `token_version` في ادعاءات JWT وفحصها مقابل الذاكرة المؤقتة عند العمليات الحساسة.
  3. **إلغاء التخزين المؤقت عبر الأحداث**: بث رسالة عبر ناقل الأحداث لتطهير ذاكرة الصلاحيات المؤقتة للمستخدم فورياً.

---

## الأنماط المضادة الشائعة (Anti-Patterns to Avoid)

| النمط المضاد | الخطر الأمني | المعالجة الهندسية في الإنتاج |
| :--- | :--- | :--- |
| **مفاتيح سياق نصية** (`ctx.Value("user")`) | إمكانية استبدال أو تصادم سياق الأمان من حزم خارجية | استخدام أنواع بنى خاصة غير مصدّرة (`type contextKey struct{}`) |
| **فحص الأدوار في المعالجات** (`if role == "Admin"`) | اقتران هش يعطل التفويض المرن للصلاحيات | فحص الصلاحيات الدقيقة (`HasPermission(perms, "invoices:approve")`) |
| **استعلام قاعدة البيانات في كل طلب** | تشبع لقاعدة البيانات وتأخير شبكي حاد | تخزين السياسات في الذاكرة عبر `sync.RWMutex` أو `atomic.Pointer` |
| **تسريب تفاصيل الصلاحيات في خطأ 403** | مساعدة المهاجمين على استكشاف ثغرات النظام | إرجاع تفاصيل عامة ومقتضبة عبر مغلفات RFC 7807 |
| **تجاهل سباق البيانات** | تلف البيانات أو تجاوز الصلاحيات تحت الضغط المتزامن | تشغيل الاختبارات دائماً بمفعل السباق `go test -race` |
| **السماح الافتراضي للمسارات غير المعرفة** | تصبح المسارات الجديدة عامة ومكشوفة دون قصد | تطبيق المنع الافتراضي الصارم (Fail-Closed) |

---

## قائمة التحقق الأمني للإنتاج (Verification Checklist)

- [ ] **المنع الافتراضي**: المسارات غير المعرفة أو السياقات غير المصادق عليها ترفض فوراً بـ 401 أو 403.
- [ ] **رسم وراثة آمن متزامن (Thread-Safe DAG)**: اجتياز وراثة الأدوار محمي بأقفال قراءة ومحصن ضد الحلقات التكرارية.
- [ ] **مفاتيح سياق غير مصدّرة**: جميع قيم السياق تُحفظ عبر أنواع بنى خاصة.
- [ ] **الفصل الاستاتيكي SSD**: منع الأدوار المتعارضة وقت الإسناد.
- [ ] **مطابقة RFC 7807**: استجابات الرفض تتبع صيغة `application/problem+json`.
- [ ] **منع تسريب المعلومات**: استجابات الأخطاء لا تكشف مفاتيح الصلاحيات الداخلية.
- [ ] **خلو تام من سباق البيانات**: اجتياز `go test -race ./...` بنجاح كامل.
- [ ] **إبطال الجلسات**: وجود آلية لتحديث أو إلغاء الجلسات النشطة فور تعديل الأدوار.
