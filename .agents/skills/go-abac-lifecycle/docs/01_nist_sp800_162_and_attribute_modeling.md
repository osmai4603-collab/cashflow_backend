# معيار NIST SP 800-162 ونمذجة السمات في Go

توضح هذه الوثيقة النمذجة الرياضية الرسمية للتحكم بالوصول القائم على السمات (ABAC) وفقاً لمعيار **NIST SP 800-162** (*دليل تعريف واعتبارات التحكم بالوصول القائم على السمات*)، وكيفية ترجمتها إلى هياكل برمجية اصطلاحية وعالية الأداء في لغة Go.

---

## 1. نموذج ABAC الرسمي (معيار NIST SP 800-162)

يعرّف معيار NIST SP 800-162 نموذج ABAC بأنه منهجية للتحكم بالوصول تُمنح فيها امتيازات التفويض للمستخدمين عبر تقييم السمات المسندة إلى الفاعلين، والموارد، والإجراءات، والبيئة المحيطة.

رياضياً، يُعرّف فضاء التقييم كحاصل الضرب الديكارتي لأربعة نطاقات سمات:

$$\mathcal{C} = \mathcal{S} \times \mathcal{R} \times \mathcal{A} \times \mathcal{E}$$

حيث:

- $\mathcal{S}$ هي مجموعة **سمات الفاعل (Subject Attributes)**.
- $\mathcal{R}$ هي مجموعة **سمات المورد (Resource Attributes)**.
- $\mathcal{A}$ هي مجموعة **سمات الإجراء (Action Attributes)**.
- $\mathcal{E}$ هي مجموعة **سمات البيئة (Environment Attributes)**.

قاعدة السياسة $R_i$ هي دالة بوليانية تسقط سياق التقييم $c \in \mathcal{C}$ إلى قرار قاطع:

$$R_i: \mathcal{C} \to \{\text{Permit}, \text{Deny}, \text{NotApplicable}\}$$

```text
       ┌────────────────────────┐
       │     سمات الفاعل        │
       │ (المعرف، الأدوار، الشركة)│
       └───────────┬────────────┘
                   │
                   ▼
       ┌────────────────────────┐         ┌────────────────────────┐
       │      سمات المورد       │────────▶│      سياق التقييم      │
       │(النوع، المالك، الحالة) │         │       (الرباعية)       │
       └────────────────────────┘         └───────────┬────────────┘
                   ▲                                  │
                   │                                  ▼
       ┌───────────┴────────────┐         ┌────────────────────────┐
       │      سمات الإجراء      │         │     قواعد السياسات     │
       │ (النوع: قراءة/تعديل)   │         │    R_i(S, R, A, E)     │
       └────────────────────────┘         └───────────┬────────────┘
                   ▲                                  │
                   │                                  ▼
       ┌───────────┴────────────┐         ┌────────────────────────┐
       │       سمات البيئة      │         │    قرار المحرك النهائي │
       │ (الوقت، الـ IP، الأمان)│         │     (سماح أم منع)      │
       └────────────────────────┘         └────────────────────────┘
```

---

## 2. النمط محكم الأنواع مقابل القابلية للتوسع الديناميكي في Go

أحد المآزق المعمارية الشائعة عند بناء أنظمة ABAC في Go هو الاختيار بين:

1. **الأنواع الصلبة المحكمة (`struct`)**: أمان تام وقت التصريف، صفر استهلاك للذاكرة على الـ Heap، وسرعة وصول فائقة للحقول، ولكن يعيبها صعوبة التوسع بالسمات المتنوعة عبر الخدمات المصغرة.
2. **الخرائط العامة غير محكمة الأنواع (`map[string]any`)**: مرونة غير محدودة، ولكن يعيبها استهلاك مكثف للذاكرة وفقدان التحقق الصارم وقت التصريف ومخاطر حدوث Runtime Panic عند تحويل الأنواع الخاطئ.

يحقق **النمط الاصطلاحي في Go** التوازن الأمثل عبر الجمع بين:

- **حقول رئيسية موحدة ومحكمة الأنواع**: للسمات المشتركة عالمياً في كافة فحوصات الصلاحيات (`ID`, `TenantID`, `Type`, `OwnerID`, `RequestTime`).
- **خريطة تمديد فرعية ديناميكية (`map[string]any`)**: للسمات التخصصية التي تختلف من كيان لآخر أو بين الأنظمة الفرعية.

### النمذجة القياسية في Go

```go
package abac

import (
    "fmt"
    "time"
)

// Subject يمثل هوية المتصل وبياناته الأمنية
type Subject struct {
    ID          string         `json:"id"`
    TenantID    string         `json:"tenant_id,omitempty"`
    Roles       []string       `json:"roles,omitempty"`
    Department  string         `json:"department,omitempty"`
    Clearance   int            `json:"clearance,omitempty"`
    Attributes  map[string]any `json:"attributes,omitempty"`
}

// Resource يمثل الكيان المستهدف المطلوب التعامل معه
type Resource struct {
    ID          string         `json:"id"`
    Type        string         `json:"type"` // مثال: "document", "invoice", "account"
    OwnerID     string         `json:"owner_id,omitempty"`
    TenantID    string         `json:"tenant_id,omitempty"`
    Department  string         `json:"department,omitempty"`
    Sensitivity int            `json:"sensitivity,omitempty"`
    Status      string         `json:"status,omitempty"`
    Attributes  map[string]any `json:"attributes,omitempty"`
}

// Action يمثل العملية المطلوب تنفيذها
type Action struct {
    Verb   string `json:"verb"`             // مثال: "read", "create", "update", "delete", "approve"
    Method string `json:"method,omitempty"` // أسلوب HTTP أو اسم دالة RPC
}

// Environment يمثل السياق البيئي المحيط لحظة الطلب
type Environment struct {
    RequestTime time.Time      `json:"request_time"`
    ClientIP    string         `json:"client_ip,omitempty"`
    NetworkZone string         `json:"network_zone,omitempty"`
    Attributes  map[string]any `json:"attributes,omitempty"`
}

// EvaluationContext يجمع أبعاد التقييم الأربعة معاً
type EvaluationContext struct {
    Subject     Subject     `json:"subject"`
    Resource    Resource    `json:"resource"`
    Action      Action      `json:"action"`
    Environment Environment `json:"environment"`
}
```

---

## 3. دوال الوصول الآمن للسمات (Safe Attribute Accessors)

لمنع حدوث Runtime Panic عند قراءة السمات الديناميكية من الخرائط، يجب استخدام دوال وصول مساعدة ومحمية بقيم افتراضية:

```go
// GetStringAttr يستخرج قيمة نصية بأمان من خريطة السمات مع قيمة افتراضية
func GetStringAttr(attrs map[string]any, key string, defaultVal string) string {
    if attrs == nil {
        return defaultVal
    }
    val, ok := attrs[key]
    if !ok {
        return defaultVal
    }
    strVal, ok := val.(string)
    if !ok {
        return defaultVal
    }
    return strVal
}

// GetFloatAttr يستخرج قيمة رقمية عشرية بأمان مع معالجة كافة أنواع الأرقام
func GetFloatAttr(attrs map[string]any, key string, defaultVal float64) float64 {
    if attrs == nil {
        return defaultVal
    }
    val, ok := attrs[key]
    if !ok {
        return defaultVal
    }
    switch v := val.(type) {
    case float64:
        return v
    case float32:
        return float64(v)
    case int:
        return float64(v)
    case int64:
        return float64(v)
    default:
        return defaultVal
    }
}

// GetBoolAttr يستخرج قيمة بوليانية بأمان مع قيمة افتراضية
func GetBoolAttr(attrs map[string]any, key string, defaultVal bool) bool {
    if attrs == nil {
        return defaultVal
    }
    val, ok := attrs[key]
    if !ok {
        return defaultVal
    }
    boolVal, ok := val.(bool)
    if !ok {
        return defaultVal
    }
    return boolVal
}
```

---

## 4. عزل المستأجرين وحماية ثغرات BOLA/IDOR

في أي نظام متعدد المستأجرين (Multi-Tenant)، يجب فرض حدود منطقية وتشفيرية صارمة بين المستأجرين. يعالج نموذج ABAC وفق NIST SP 800-162 عزل المستأجرين أصلياً عند طبقة السمات:

```go
// InvariantTenantIsolation يتحقق من تطابق المستأجر بين الفاعل والمورد لمنع BOLA/IDOR
func InvariantTenantIsolation(ctx EvaluationContext) error {
    if ctx.Subject.TenantID == "" || ctx.Resource.TenantID == "" {
        return fmt.Errorf("tenant isolation violation: missing tenant ID")
    }
    if ctx.Subject.TenantID != ctx.Resource.TenantID {
        return fmt.Errorf("tenant isolation violation: subject tenant %q does not match resource tenant %q",
            ctx.Subject.TenantID, ctx.Resource.TenantID)
    }
    return nil
}
```

بفرض هذا القيد الحتمي كأول خطوة في كل قاعدة سياسة أو تقييم في محرك PDP، يتم القضاء هندسياً على ثغرات **تجاوز الصلاحيات على مستوى الكائن (Broken Object Level Authorization - BOLA)** وثغرات **المراجع المباشرة غير الآمنة (IDOR)** (المرتبة الأولى في تصنيف OWASP Top 10 لأمان الواجهات البرمجية).
