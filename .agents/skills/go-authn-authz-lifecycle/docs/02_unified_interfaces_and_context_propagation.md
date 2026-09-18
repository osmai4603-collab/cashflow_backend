# الواجهات الموحدة وإدارة السياق بأمان في Go (Unified Interfaces & Context Propagation)

في المعمارية القياسية للغة Go، يجب أن تكون التجريدات الأمنية خفيفة الوزن وقابلة للتركيب (Composable)، ومحايدة تماماً لأي إطار عمل ويب (مثل Chi أو Gin أو Echo أو Fiber) أو بيئة استدعاء إجرائي عن بُعد (`google.golang.org/grpc`).

---

## 1. نمط ناقل الترويسات الموحد (`HeaderCarrier`)

لفصل استخراج البراهين والرموز التشفيرية عن البروتوكول الشبكي المحدد، نعرّف واجهة `HeaderCarrier`. تمكّن هذه الواجهة كلاً من طلبات HTTP وبيانات gRPC الوصفية من تلبية نفس العقد البرمجي بمرونة:

```go
package security

import (
 "context"
 "net/http"
 "strings"
)

// HeaderCarrier واجهة تجريدية لقراءة الترويسات بغض النظر عن البروتوكول الشبكي
type HeaderCarrier interface {
 Get(key string) string
}

// HTTPHeaderCarrier محوّل من *http.Request لتلبية واجهة HeaderCarrier
type HTTPHeaderCarrier struct {
 req *http.Request
}

func NewHTTPHeaderCarrier(r *http.Request) HTTPHeaderCarrier {
 return HTTPHeaderCarrier{req: r}
}

func (h HTTPHeaderCarrier) Get(key string) string {
 return h.req.Header.Get(key)
}

// MapHeaderCarrier ناقل تخزين بالذاكرة (مثالي للبيانات الوصفية في gRPC أو الاختبارات)
type MapHeaderCarrier map[string]string

func (m MapHeaderCarrier) Get(key string) string {
 if val, ok := m[strings.ToLower(key)]; ok {
  return val
 }
 return m[key]
}
```

---

## 2. نموذج هوية الفاعل النقي (`Principal`)

يمثل كائن `Principal` الهوية النهائية الموثقة والمطهرة بعد اكتمال مرحلة المصادقة. يتميز بتصميمه المنيع (غير القابل للتعديل) واستخدامه لخرائط Go مع هياكل فارغة (`map[string]struct{}`) لتحقيق التحقق الفوري في زمن ثابت $O(1)$:

```go
package security

import "time"

// Principal يمثل كائن الهوية الموثق للمتصل
type Principal struct {
 ID         string              `json:"id"`
 TenantID   string              `json:"tenant_id"`
 Email      string              `json:"email,omitempty"`
 Roles      map[string]struct{} `json:"roles,omitempty"`
 Scopes     map[string]struct{} `json:"scopes,omitempty"`
 Attributes map[string]any      `json:"attributes,omitempty"`
 IssuedAt   time.Time           `json:"issued_at"`
 ExpiresAt  time.Time           `json:"expires_at"`
}

// HasRole يتحقق من انتساب الفاعل لدور معين في زمن O(1)
func (p *Principal) HasRole(role string) bool {
 if p == nil {
  return false
 }
 _, ok := p.Roles[role]
 return ok
}

// HasScope يتحقق من امتلاك الرمز لنطاق تفويض OAuth معين في زمن O(1)
func (p *Principal) HasScope(scope string) bool {
 if p == nil {
  return false
 }
 _, ok := p.Scopes[scope]
 return ok
}
```

---

## 3. حماية السياق (`context.Context`) عبر المفاتيح غير المصدّرة

ينص التوثيق الرسمي لحزمة السياق في Go (`pkg.go.dev/context`) على:
> "يجب أن يكون المفتاح قابلاً للمقارنة، ولا يجوز أن يكون من نوع string أو أي نوع مدمج آخر لتفادي التصادم بين الحزم المختلفة التي تستخدم السياق. يجب على مستخدمي WithValue تعريف أنواع خاصة بهم للمفاتيح."

### النمط المضاد (عُرضة للتصادم البرمجي)

```go
// نمط خاطئ: المفاتيح النصية تؤدي إلى تصادم وتجاوزات غير مقصودة مع مكتبات أخرى
ctx = context.WithValue(ctx, "user", principal)
```

### النمط الصحيح والمعتمد (حصانة تامة من التصادم)

```go
package security

import "context"

// نوع بنية فارغة غير مصدّر بحجم صفر بايت، يستحيل إنشاؤه أو الوصول إليه من خارج هذه الحزمة
type contextKey struct{}

var principalKey = contextKey{}

// InjectPrincipal يحقن كائن الهوية Principal داخل السياق
func InjectPrincipal(ctx context.Context, p *Principal) context.Context {
 return context.WithValue(ctx, principalKey, p)
}

// ExtractPrincipal يسترجع الهوية من السياق مع فحص وتأكيد النوع بأمان
func ExtractPrincipal(ctx context.Context) (*Principal, bool) {
 p, ok := ctx.Value(principalKey).(*Principal)
 return p, ok && p != nil
}

// MustExtractPrincipal يسترجع الهوية أو يرجع خطأ عدم المصادقة للمسارات المحمية إجبارياً
func MustExtractPrincipal(ctx context.Context) (*Principal, error) {
 p, ok := ExtractPrincipal(ctx)
 if !ok {
  return nil, ErrUnauthenticated
 }
 return p, nil
}
```

---

## 4. تجريد نقطة اتخاذ القرار (Policy Decision Point - PDP)

تفصل واجهة `Authorizer` حالات الاستخدام في طبقة التطبيق عن محركات السياسات المحددة (مثل Casbin أو OPA أو Zanzibar أو شروط Go المخصصة):

```go
package security

import "context"

// Action يحدد نوع العملية المطلوبة على المورد
type Action string

const (
 ActionCreate  Action = "create"
 ActionRead    Action = "read"
 ActionUpdate  Action = "update"
 ActionDelete  Action = "delete"
 ActionApprove Action = "approve"
)

// Resource يمثل المورد الخاضع لتقييم السياسة
type Resource struct {
 Type       string         `json:"type"`
 ID         string         `json:"id"`
 TenantID   string         `json:"tenant_id"`
 Attributes map[string]any `json:"attributes,omitempty"`
}

// Authorizer واجهة تقييم السياسات واتخاذ القرار القاطع (PDP)
type Authorizer interface {
 Authorize(ctx context.Context, sub *Principal, act Action, res Resource) (bool, error)
}
```
