# مسارات التدقيق الأمني والإثراء الديناميكي عبر PIP في Go

توضح هذه الوثيقة المعمارية الهندسية والتطبيقية لـ **نقطة جمع المعلومات (Policy Information Point - PIP)** و**مسارات التدقيق الأمني غير القابلة للتلاعب (Tamper-Evident Audit Trails)** في أنظمة ABAC بلغة Go.

---

## 1. الإثراء الديناميكي للسمات (Dynamic Attribute Enrichment - PIP)

في أنظمة التحكم بالوصول القائم على السمات النقية، لا تحمل رموز العميل (مثل JWT) سوى سمات **الفاعل (Subject)** (مثل `user_id`, `roles`, `tenant_id`, `clearance`). وهي لا تحتوي على السمات الحالية لـ **المورد (Resource)** (مثل `owner_id`, `status`, `sensitivity_level`, `amount`) لسببين جوهريين:

1. الموارد متغيرة ومتقلبة الحالة باستمرار؛ وتضمين حالتها داخل رموز العميل يؤدي إلى قرارات تفويض قديمة وفاسدة (Stale Decisions).
2. حشو بيانات الموارد داخل رموز JWT يؤدي إلى تضخم حجم الرموز ويعرض تفاصيل الكيانات للتسريب الأمني.

لذا، تتولى **نقطة جمع المعلومات (PIP)** إثراء سياق التقييم ديناميكياً عبر جلب السمات المحدثة للكيان من قواعد البيانات أو مخازن الذاكرة المؤقتة.

```text
طلب HTTP قادم (Subject & Resource ID)
                 │
                 ▼
┌─────────────────────────────────┐
│ وسيط فرض السياسة (PEP)          │
└────────────────┬────────────────┘
                 │ جلب سمات المورد المستهدف
                 ▼
┌─────────────────────────────────┐
│ نقطة جمع المعلومات (PIP)        │
│ (PostgreSQL / Redis / Memory)   │
└────────────────┬────────────────┘
                 │ إعادة هيكل المورد مكتمل السمات
                 ▼
┌─────────────────────────────────┐
│ نقطة اتخاذ القرار (PDP)         │
│ (تقييم الرباعية S, R, A, E)     │
└────────────────┬────────────────┘
                 │ تصدير حدث التدقيق الأمني
                 ▼
┌─────────────────────────────────┐
│ مسجل التدقيق المهيكل (slog)     │
│ + مقاييس Prometheus OpenMetrics │
└─────────────────────────────────┘
```

---

## 2. واجهة مستكشف PIP القابلة للتركيب في Go

تحدد المعمارية النظيفة واجهة منفذ (Port Interface) لعملية استرجاع بيانات PIP:

```go
package abac

import "context"

// PIPResolver يحدد العقد البرمجي لجلب سمات المورد ديناميكياً
type PIPResolver interface {
    ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error)
}
```

### تطبيق عملي مع قاعدة البيانات

```go
type DatabasePIPResolver struct {
    db DatabaseClient // مستودع الكيان أو pgxpool
}

func (r *DatabasePIPResolver) ResolveResource(ctx context.Context, resourceType, resourceID string) (Resource, error) {
    // 1. جلب الكيان الخام من قاعدة البيانات
    row, err := r.db.FindEntity(ctx, resourceType, resourceID)
    if err != nil {
        return Resource{}, fmt.Errorf("فشل استرجاع المورد %s:%s: %w", resourceType, resourceID, err)
    }

    // 2. تحويل كائن المجال إلى هيكل المورد العام Resource
    return Resource{
        ID:          row.ID,
        Type:        resourceType,
        OwnerID:     row.OwnerID,
        TenantID:    row.TenantID,
        Department:  row.Department,
        Sensitivity: row.SensitivityLevel,
        Status:      row.Status,
        Attributes: map[string]any{
            "amount": row.Amount,
        },
    }, nil
}
```

---

## 3. سجلات التدقيق الأمني المهيكلة عبر `log/slog`

يجب توثيق كل قرار تفويض أمني للامتثال للمعايير واللوائح الدولية (SOC2, ISO 27001, PCI-DSS, HIPAA).

### مخطط حدث التدقيق الأمني

```go
package abac

import (
    "log/slog"
    "time"
)

type AuditEvent struct {
    Timestamp    time.Time
    SubjectID    string
    TenantID     string
    ResourceType string
    ResourceID   string
    Action       string
    Decision     Decision
    RuleID       string
    Reason       string
    ClientIP     string
    Latency      time.Duration
}

func EmitAuditEvent(logger *slog.Logger, ev AuditEvent) {
    level := slog.LevelInfo
    if ev.Decision == DecisionDeny {
        level = slog.LevelWarn
    }

    decisionStr := "permit"
    if ev.Decision == DecisionDeny {
        decisionStr = "deny"
    }

    logger.Log(nil, level, "ABAC_AUDIT_DECISION",
        slog.String("event_type", "authorization_audit"),
        slog.String("decision", decisionStr),
        slog.String("rule_id", ev.RuleID),
        slog.String("reason", ev.Reason),
        slog.String("subject_id", ev.SubjectID),
        slog.String("tenant_id", ev.TenantID),
        slog.String("resource_type", ev.ResourceType),
        slog.String("resource_id", ev.ResourceID),
        slog.String("action", ev.Action),
        slog.String("client_ip", ev.ClientIP),
        slog.Duration("latency_ns", ev.Latency),
    )
}
```

---

## 4. القياس والمراقبة عبر Prometheus OpenMetrics

تتيح المراقبة اللحظية لزمن استجابة التفويض ومعدلات الرفض كشف محاولات التخمين والهجمات السيبرانية اللحظية:

```go
package abac

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    AuthzEvaluationsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "authz_evaluations_total",
            Help: "إجمالي عدد تقييمات تفويض ABAC مقسمة حسب القرار ونوع المورد والعملية.",
        },
        []string{"decision", "resource_type", "action"},
    )

    AuthzEvaluationDurationSeconds = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "authz_evaluation_duration_seconds",
            Help:    "مدرج تكراري لزمن استجابة تقييم سياسات ABAC بالثواني.",
            Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005}, // من 10 ميكروثانية إلى 5 ملي ثانية
        },
        []string{"resource_type"},
    )
)
```

بمراقبة `authz_evaluations_total{decision="deny"}`، يمكن لمنظومة التنبيهات إخطار فريق العمليات الأمنية (SOC) فوراً عند حدوث قفزة غير طبيعية في طلبات الوصول المرفوضة لعنوان IP معين أو مستأجر بعينه.
