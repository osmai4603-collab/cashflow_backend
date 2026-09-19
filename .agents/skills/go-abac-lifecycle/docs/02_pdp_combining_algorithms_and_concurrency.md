# خوارزميات دمج القرارات في PDP والتقييم عالي التزامن في Go

توضح هذه الوثيقة المعمارية الهندسية والتطبيقية لـ **نقطة اتخاذ القرار (Policy Decision Point - PDP)** في لغة Go، مع التركيز على خوارزميات دمج السياسات، أمان التزامن (Thread Safety)، التقييم بدون أقفال (Lock-Free)، والتحديث اللحظي للسياسات دون انقطاع الخدمة (Zero-Downtime Hot-Reloading).

---

## 1. دور نقطة اتخاذ القرار (PDP)

نقطة اتخاذ القرار (PDP) هي المحرك الرياضي لنظام ABAC. بناءً على `EvaluationContext` الرباعي $(S, R, A, E)$ ومجموعة قواعد السياسات النشطة $\{R_1, R_2, \dots, R_n\}$، تقوم الـ PDP بتقييم القواعد السارية ودمج مخرجاتها في قرار نهائي حاسم: **سماح (Permit)** أو **منع (Deny)**.

```text
               ┌────────────────────────────────────────┐
               │         سياق التقييم القادم            │
               │        (Subject, Resource, ...)        │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │          قواعد السياسات النشطة          │
               │   Rule 1 ───▶ Rule 2 ───▶ Rule 3 ...   │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │           خوارزمية دمج القرارات        │
               │   (Deny-Overrides / Permit-Overrides)  │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │              القرار النهائي            │
               │         سماح (Permit) أم منع (Deny)    │
               └────────────────────────────────────────┘
```

---

## 2. خوارزميات دمج السياسات (Policy Combining Algorithms)

عندما تنطبق أكثر من قاعدة على نفس الطلب، قد تنشأ تعارضات (مثال: القاعدة أ تسمح، بينما القاعدة ب تمنع). يحل محرك PDP هذه التعارضات باستخدام **خوارزمية دمج حتمية**.

### 2.1 خوارزمية تغليب المنع (Deny-Overrides - المعيار المالي والأمني الصارم)

- **القاعدة**: إذا انتهت **أي** قاعدة سارية إلى المنع `Deny`، فإن القرار الإجمالي يكون فورياً وقطعياً هو **المنع (Deny)**.
- **شرط السماح**: يُمنح الوصول فقط وفقط إذا انتهت قاعدة سارية واحدة على الأقل إلى السماح `Permit`، ولم تنته **أي** قاعدة سارية أخرى إلى المنع.
- **الخيار الافتراضي**: في حال عدم وجود أي قواعد سارية، فالقرار هو **المنع الافتراضي (Default Deny)**.
- **حالات الاستخدام**: الدفاتر المالية، العمليات المصرفية، حماية تسريب البيانات الحساسة.

```go
func evaluateDenyOverrides(rules []PolicyRule, ctx EvaluationContext) (Decision, string) {
    hasPermit := false
    permitReason := ""

    for _, rule := range rules {
        if !rule.Target(ctx) {
            continue
        }
        dec, reason := rule.Evaluate(ctx)
        if dec == DecisionDeny {
            return DecisionDeny, fmt.Sprintf("denied by rule %q: %s", rule.ID(), reason)
        }
        if dec == DecisionPermit {
            hasPermit = true
            permitReason = reason
        }
    }

    if hasPermit {
        return DecisionPermit, permitReason
    }
    return DecisionDeny, "default deny: no applicable rule permitted the action"
}
```

### 2.2 خوارزمية تغليب السماح (Permit-Overrides)

- **القاعدة**: إذا انتهت **أي** قاعدة سارية إلى السماح `Permit`، فالقرار النهائي هو **السماح**، حتى لو انتهت قواعد أخرى إلى المنع.
- **شرط المنع**: يُرفض الوصول إذا انتهت كافة القواعد السارية إلى المنع، أو لم تنطبق أي قاعدة.
- **حالات الاستخدام**: مسارات القراءة العامة، استثناءات الطوارئ، وبوابات التعاون المفتوحة.

### 2.3 خوارزمية القاعدة الأولى المنطبقة (First-Applicable)

- **القاعدة**: يتم تقييم القواعد بالتسلسل وفق ترتيب تسجيلها. أول قاعدة ينطبق شرطها `Target(ctx)` وتنتج سماحاً أو منعاً يتم تبني قرارها فوراً وقطع مسار التقييم.
- **حالات الاستخدام**: جدران الحماية وقوائم التصفية التتابعية التي تتطلب قطعاً سريعاً (Short-Circuit) لتوفير المعالجة.

---

## 3. التخزين المؤقت في الذاكرة والتحديث الحي المتزامن

نظراً لوقوع فحص الصلاحيات مباشرة في المسار الحرج لكل طلب، فإن الاستعلام من الشبكة أو القرص لقراءة السياسات يسبب تأخيراً غير مقبول. يجب أن تعيش السياسات دائماً في الذاكرة الحية (In-Memory).

### 3.1 النمط أ: القراءة المتزامنة الكثيفة باستخدام `sync.RWMutex`

للأنظمة التي تُحدّث سياساتها دورياً أو عبر واجهات الإدارة:

```go
type Engine struct {
    mu        sync.RWMutex
    algorithm CombiningAlgorithm
    rules     []PolicyRule
    logger    *slog.Logger
}

func (e *Engine) Evaluate(ctx EvaluationContext) (Decision, string) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    switch e.algorithm {
    case DenyOverrides:
        return evaluateDenyOverrides(e.rules, ctx)
    case PermitOverrides:
        return evaluatePermitOverrides(e.rules, ctx)
    case FirstApplicable:
        return evaluateFirstApplicable(e.rules, ctx)
    default:
        return DecisionDeny, "unknown combining algorithm"
    }
}

// ReloadRules يستبدل القواعد النشطة ذرياً تحت قفل الكتابة
func (e *Engine) ReloadRules(newRules []PolicyRule) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.rules = make([]PolicyRule, len(newRules))
    copy(e.rules, newRules)
}
```

### 3.2 النمط ب: الاستبدال الذري الخالي من الأقفال عبر `sync/atomic`

للبيئات ذات الإنتاجية الفائقة (مئات آلاف الطلبات في الثانية)، قد تسبب أقفال القراءة تنازعاً في الذاكرة المؤقتة للمعالج (CPU Cache Bouncing). يوفر استخدام `atomic.Pointer` صفراً من عبء المزامنة في مسار القراءة:

```go
import "sync/atomic"

type PolicySnapshot struct {
    algorithm CombiningAlgorithm
    rules     []PolicyRule
}

type AtomicEngine struct {
    snapshot atomic.Pointer[PolicySnapshot]
}

func NewAtomicEngine(alg CombiningAlgorithm, rules []PolicyRule) *AtomicEngine {
    eng := &AtomicEngine{}
    eng.snapshot.Store(&PolicySnapshot{
        algorithm: alg,
        rules:     rules,
    })
    return eng
}

func (e *AtomicEngine) Evaluate(ctx EvaluationContext) (Decision, string) {
    snap := e.snapshot.Load() // قراءة ذرية بدون أقفال نهائياً
    switch snap.algorithm {
    case DenyOverrides:
        return evaluateDenyOverrides(snap.rules, ctx)
    }
    return DecisionDeny, "unknown algorithm"
}

func (e *AtomicEngine) SwapPolicies(newAlg CombiningAlgorithm, newRules []PolicyRule) {
    e.snapshot.Store(&PolicySnapshot{
        algorithm: newAlg,
        rules:     newRules,
    })
}
```

---

## 4. إرشادات الأداء: تقييم في زمن يقل عن الميكروثانية

1. **تجنب حجز الذاكرة في الـ Heap أثناء التقييم**: لا تقم بإنشاء شرائح (Slices) أو خرائط جديدة داخل دالة `rule.Evaluate()`.
2. **الترشيح المسبق للأهداف**: صمم دالة `rule.Target(ctx) bool` كفحص أولي سريع ورخيص (مثل التحقق من `ctx.Resource.Type == "invoice"`) قبل خوض المقارنات المعقدة للسمات.
3. **التحقق المستمر بـ `-race`**: شغّل دائماً الاختبارات عبر `go test -race` لضمان عدم وجود أي سباق بيانات بين القراءة المتزامنة وتحديث السياسات.
