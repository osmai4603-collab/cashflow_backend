# سياق النقل الموحد ومفاتيح السياق الآمنة من التصادم (Unified TransportContext & Context Keys)

توضح هذه الوثيقة المواصفات الهندسية لواجهة `TransportContext`، والتطبيقات الفعلية لبروتوكولي HTTP و gRPC، وآلية تمرير البيانات داخل السياق عبر مفاتيح خاصة غير مصدرة ومحمية من التصادم في Go.

---

## 1. تصميم واجهة `TransportContext`

توفر واجهة `TransportContext` منظوراً موحداً ومحايداً للبروتوكولات لفحص ومعالجة طلب الشبكة الوارد. يتيح ذلك لمدققي الأمان، ومحقني بيانات القياس عن بُعد، ومحددي معدل الطلبات فحص الطلبات الواردة دون الحاجة لاستيراد `net/http` أو `google.golang.org/grpc/metadata`.

```go
package transport

import "context"

type TransportContext interface {
 // Protocol يحدد ما إذا كان الطلب قد وصل عبر HTTP أو gRPC.
 Protocol() TransportProtocol

 // Context يعيد سياق Go القياسي الحامل للمهل وإشارات الإلغاء.
 Context() context.Context

 // RequestID يعيد معرف التتبع والارتباط الموزع.
 RequestID() string

 // ClientIP يعيد عنوان IP القياسي للعميل بعد فحص ترويسات الوكيل المعتمدة.
 ClientIP() string

 // UserAgent يعيد بصمة برنامج العميل.
 UserAgent() string

 // Header يسترجع قيمة البيانات الوصفية بشكل غير حساس لحالة الأحرف.
 Header(key string) string

 // Principal يعيد الهوية الأمنية المعتمدة للمتصل في حال التحقق منها.
 Principal() (*SecurityPrincipal, bool)

 // SetPrincipal يحفظ الهوية الأمنية المعتمدة بعد نجاح التحقق.
 SetPrincipal(principal *SecurityPrincipal)
}
```

---

## 2. التطبيقات الملموسة لحدود الدخول (Concrete Ingress Implementations)

### 2.1 سياق نقل HTTP (`httpTransportContext`)

يغلف تطبيق HTTP كائني `http.ResponseWriter` و `*http.Request` القياسيين:

```go
package transport

import (
 "context"
 "net"
 "net/http"
 "strings"
)

type httpTransportContext struct {
 req       *http.Request
 requestID string
 principal *SecurityPrincipal
}

func NewHTTPTransportContext(req *http.Request, requestID string) TransportContext {
 return &httpTransportContext{
  req:       req,
  requestID: requestID,
 }
}

func (h *httpTransportContext) Protocol() TransportProtocol {
 return ProtocolHTTP
}

func (h *httpTransportContext) Context() context.Context {
 return h.req.Context()
}

func (h *httpTransportContext) RequestID() string {
 return h.requestID
}

func (h *httpTransportContext) ClientIP() string {
 // 1. فحص ترويسات الوكيل القياسية (Forwarded، X-Forwarded-For، X-Real-IP)
 if xff := h.req.Header.Get("X-Forwarded-For"); xff != "" {
  parts := strings.Split(xff, ",")
  return strings.TrimSpace(parts[0])
 }
 if xrip := h.req.Header.Get("X-Real-IP"); xrip != "" {
  return strings.TrimSpace(xrip)
 }

 // 2. الرجوع لعنوان TCP المباشر RemoteAddr في حال غياب الترويسات
 host, _, err := net.SplitHostPort(h.req.RemoteAddr)
 if err != nil {
  return h.req.RemoteAddr
 }
 return host
}

func (h *httpTransportContext) UserAgent() string {
 return h.req.UserAgent()
}

func (h *httpTransportContext) Header(key string) string {
 return h.req.Header.Get(key)
}

func (h *httpTransportContext) Principal() (*SecurityPrincipal, bool) {
 if h.principal != nil {
  return h.principal, true
 }
 return GetPrincipal(h.req.Context())
}

func (h *httpTransportContext) SetPrincipal(principal *SecurityPrincipal) {
 h.principal = principal
 *h.req = *h.req.WithContext(WithPrincipal(h.req.Context(), principal))
}
```

### 2.2 سياق نقل gRPC (`grpcTransportContext`)

يغلف تطبيق gRPC كائنات `context.Context` و `metadata.MD` و `peer.Peer`:

```go
package transport

import (
 "context"
 "net"
 "strings"

 "google.golang.org/grpc/metadata"
 "google.golang.org/grpc/peer"
)

type grpcTransportContext struct {
 ctx       context.Context
 md        metadata.MD
 requestID string
 principal *SecurityPrincipal
}

func NewGRPCTransportContext(ctx context.Context, requestID string) TransportContext {
 md, ok := metadata.FromIncomingContext(ctx)
 if !ok {
  md = metadata.New(nil)
 }
 return &grpcTransportContext{
  ctx:       ctx,
  md:        md,
  requestID: requestID,
 }
}

func (g *grpcTransportContext) Protocol() TransportProtocol {
 return ProtocolGRPC
}

func (g *grpcTransportContext) Context() context.Context {
 return g.ctx
}

func (g *grpcTransportContext) RequestID() string {
 return g.requestID
}

func (g *grpcTransportContext) ClientIP() string {
 // 1. فحص البيانات الوصفية لترويسات تمرير الوكيل
 if vals := g.md.Get("x-forwarded-for"); len(vals) > 0 {
  parts := strings.Split(vals[0], ",")
  return strings.TrimSpace(parts[0])
 }
 if vals := g.md.Get("x-real-ip"); len(vals) > 0 {
  return strings.TrimSpace(vals[0])
 }

 // 2. استخراج النظير الشبكي TCP peer من سياق الاتصال
 if p, ok := peer.FromContext(g.ctx); ok && p.Addr != nil {
  host, _, err := net.SplitHostPort(p.Addr.String())
  if err == nil {
   return host
  }
  return p.Addr.String()
 }
 return "unknown"
}

func (g *grpcTransportContext) UserAgent() string {
 if vals := g.md.Get("user-agent"); len(vals) > 0 {
  return vals[0]
 }
 return "grpc-client"
}

func (g *grpcTransportContext) Header(key string) string {
 // مفاتيح البيانات الوصفية في gRPC تتطلب أحرفاً صغيرة دائماً
 vals := g.md.Get(strings.ToLower(key))
 if len(vals) > 0 {
  return vals[0]
 }
 return ""
}

func (g *grpcTransportContext) Principal() (*SecurityPrincipal, bool) {
 if g.principal != nil {
  return g.principal, true
 }
 return GetPrincipal(g.ctx)
}

func (g *grpcTransportContext) SetPrincipal(principal *SecurityPrincipal) {
 g.principal = principal
 g.ctx = WithPrincipal(g.ctx, principal)
}
```

---

## 3. معمارية مفاتيح السياق الآمنة من التصادم (Collision-Free Context Keying)

في لغة Go، يخزن `context.Context` القيم كارتباط بين مفتاح من نوع `any` وقيمة من نوع `any`. إذا استخدمت حزمتان مختلفتان مفتاحاً نصياً مثل `"user_id"` أو عدداً صحيحاً مصدراً، فإن إحداهما ستستبدل بيانات الأخرى دون سابق إنذار، مما يؤدي إلى أخطاء برمجية خفية أو ثغرات أمنية خطيرة.

### النمط الاصطلاحي للأنواع غير المصدرة (Unexported Type Idiom)

لضمان استحالة التصادم رياضياً وبرمجياً:

1. تعريف نوع مخصص غير مصدر: `type contextKey int` أو `type contextKey struct{}`.
2. الإعلان عن ثوابت أو متغيرات غير مصدرة من ذلك النوع.
3. كشف دوال مساعدة موجهة النوع (Typed Getters/Setters) فقط للقراءة والكتابة.

```go
package transport

import "context"

type contextKey int

const (
 transportInfoKey contextKey = iota
 principalKey
)

func WithTransportInfo(ctx context.Context, info TransportInfo) context.Context {
 return context.WithValue(ctx, transportInfoKey, info)
}

func GetTransportInfo(ctx context.Context) (TransportInfo, bool) {
 info, ok := ctx.Value(transportInfoKey).(TransportInfo)
 return info, ok
}

func WithPrincipal(ctx context.Context, p *SecurityPrincipal) context.Context {
 return context.WithValue(ctx, principalKey, p)
}

func GetPrincipal(ctx context.Context) (*SecurityPrincipal, bool) {
 p, ok := ctx.Value(principalKey).(*SecurityPrincipal)
 return p, ok && p != nil
}
```

### اعتبارات تخصيص الذاكرة والأداء

- `TransportInfo` هو هيكل مدمج من 4 حقول (~64 بايت). يتم نسخه مباشرة داخل `context.WithValue` بتكلفة تخصيص ذاكرة مهملة.
- `SecurityPrincipal` يتم تخزينه كمؤشر `*SecurityPrincipal`. يضمن ذلك انعدام تكلفة النسخ عبر سلاسل البرمجيات الوسيطة ويمنع تضارب حالات الهوية.
