---
name: go-transport-layer
description: "معمارية طبقة النقل (Transport Layer Architecture) الإنتاجية والمحايدة للبروتوكولات لخدمات Go الخلفية، لتوحيد بروتوكولي HTTP (REST/JSON) و gRPC (Protobuf) قبل معالجة الطلبات. تغطي حدود المحولات الأولية (Hexagonal primary adapter)، تجريد سياق النقل TransportContext، مفاتيح السياق الآمنة غير المصدرة، خط أنابيب المعالجة المسبقة المكون من 6 مراحل، التحديد الصريح للبروتوكول، البرمجيات الوسيطة والمترضات المزدوجة، مضاmultiplexing المنافذ (cmux و h2c)، والربط ثنائي الاتجاه لأخطاء النطاق."
---

# معمارية طبقة النقل في Go: توحيد المعالجة المسبقة لبروتوكولي HTTP و gRPC

تحدد هذه المهارة **معمارية طبقة النقل (Transport Layer Architecture)** الإنتاجية والمجربة والمحايدة للبروتوكولات لخدمات Go الخلفية. توفر هذه المعمارية حدود دخول (Ingress Boundary) موحدة ونظيفة للتطبيقات التي تدعم في آن واحد **HTTP/REST (JSON)** و **gRPC (Protobuf عبر HTTP/2)**، مما يضمن وسم الطلبات وتطبيعها والمصادقة عليها والتحقق منها وإثرائها **قبل** دخولها إلى حالات الاستخدام (Use Cases) أو معالجات النطاق الأساسية.

تجمع المعمارية بين المعايير الاصطلاحية لـ Go من حزمة `net/http`، ومشروع `google.golang.org/grpc` الرسمي من Google، والمعمارية السداسية / النظيفة (Hexagonal / Clean Architecture - Ports & Adapters)، وتصميم النقل في Go kit، وأنماط توحيد استدعاء الإجراءات عن بُعد الحديثة (ConnectRPC، gRPC-Gateway).

---

## المبادئ المعمارية الإنتاجية

1. **التموضع الصارم كمحول قيادة / محول أولي (Primary / Driving Adapter)**:
   تعيش طبقة النقل في الحافة الخارجية للنظام (`internal/transport/` أو `adapter/transport/`). وهي تعمل حصرياً كـ **محول قيادة (Driving Adapter)**؛ تترجم بايتات الشبكة الخارجية (TCP/HTTP/gRPC) إلى استدعاءات لنطاق العمل، وتترجم نتائج النطاق إلى استجابات شبكية.

2. **انعدام تسريب البروتوكولات إلى معالجات الأعمال (Zero Protocol Leakage)**:
   يُحظر تماماً على خدمات النطاق وحالات الاستخدام والكيانات استيراد `net/http` أو `google.golang.org/grpc` أو هياكل Protobuf المولدة تلقائياً. لا يجوز لأي `*http.Request` أو `http.ResponseWriter` أو `grpc.ServerStream` عبور الحدود إلى قلب التطبيق. تمرير أدوات النقل إلى حالات الاستخدام يولد اقتراناً شبكياً وثيقاً، ويدمر قابلية الاختبار الأحادي، وينتهك مبدأ عكس التبعية (DIP).

3. **تمرير الهوية والبروتوكول بأمان نوعي محصور في السياق (Type-Safe Context Propagation)**:
   يجب تمرير جميع البيانات الوصفية المستخرجة عند حدود النقل (البروتوكول، معرف الطلب، عنوان IP للعميل، الهوية الأمنية المعتمدة) داخل `context.Context` القياسي في Go باستخدام **أنواع مفاتيح خاصة وغير مصدرة** (`type contextKey int`). هذا يلغي برمجياً ورياضياً تصادم المفاتيح بين المكتبات.

4. **التحديد الصريح للبروتوكول (Self-Aware Ingress)**:
   يمكن لقلب التطبيق وخطوط الأنابيب العرضية دائماً الاستعلام عن *البروتوكول* الذي سلم الطلب عبر تعداد قوي الأنواع `TransportProtocol` (`HTTP` مقابل `GRPC`). يتيح ذلك القياس عن بُعد المخصص لكل بروتوكول، أو التخزين المؤقت الانتقائي، أو تشكيل الاستجابات المخصصة دون كسر التجريد المعماري.

5. **خط أنابيب حتمي للمعالجة المسبقة من 6 مراحل (Deterministic 6-Stage Pre-Handling Pipeline)**:
   يجب على كل طلب وارد (سواء دخل عبر موجه HTTP أو مستمع gRPC) اجتياز خط أنابيب معالجة مسبقة متطابق ومرتب قبل الوصول إلى المعالج:
   - **المرحلة 1: وسم البروتوكول وتسجيل الدخول (Protocol Stamping & Ingress Logging)**
   - **المرحلة 2: معرف الارتباط والتتبع الموزع (Correlation ID & Distributed Tracing)** (`X-Request-ID` و W3C TraceContext)
   - **المرحلة 3: تطبيع البيانات الوصفية (Metadata Normalization)** (توحيد الترويسات وحل عنوان IP الحقيقي)
   - **المرحلة 4: المصادقة وحل الهوية الأمنية (Authentication & SecurityPrincipal)** (رمز Bearer JWT أو مفتاح API)
   - **المرحلة 5: التحقق النحوي من الحمولة (Syntactic Payload Validation)** (فحص المخططات والشكل الخارجي بمعزل عن منطق الأعمال)
   - **المرحلة 6: المهل الزمنية وتحديد المعدل (Deadlines & Rate Limiting)** (`context.WithDeadline` ودلاء الرموز)

6. **مطابقة الأخطاء ورموز الحالة ثنائية الاتجاه (Bidirectional Error Mapping)**:
   تُرجع معالجات النطاق أخطاء Go نقية خاصة بالنطاق (مثل `ErrNotFound` و `ErrConflict` و `ErrUnauthorized`). يقوم محول النقل بمطابقة هذه الأخطاء بشكل متماثل مع **رموز حالة HTTP** (مع تفاصيل المشكلات RFC 7807) و **رموز حالة gRPC** (`codes.NotFound` و `codes.AlreadyExists` و `codes.Unauthenticated`).

7. **التعافي الآمن من الانهيارات (Fail-Safe Panic Recovery)**:
   أي هلع (Panic) يقع أثناء فك تشفير النقل أو المعالجة المسبقة أو تنفيذ الأعمال يتم التقاطه عند حدود النقل. يتم تسجيل الهلع مع تتبع المكدس (Stack Trace) في أنظمة القياس الداخلية، وتُعاد للعميل استجابة آمنة ومطهرة `500 Internal Server Error` (أو رمز gRPC `codes.Internal`) دون تسريب مؤشرات الذاكرة أو تفاصيل البنية التحتية.

---

## الهيكلية المعمارية: من الدخول المزدوج إلى المعالجة النقية

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                 حركة مرور الشبكة الواردة                         │
│         العميل (HTTP/REST JSON)                     الخدمة المصغرة (gRPC Protobuf)│
└──────────────────────────┬───────────────────────────────────────┬───────────────┘
                           │                                       │
                           ▼                                       ▼
┌───────────────────────────────────────┐   ┌──────────────────────────────────────┐
│       موجه HTTP / Chi / المكتبة القياسية│   │          محرك خادم gRPC               │
│            (http.Handler)             │   │       (UnaryServerInterceptor)       │
└──────────────────┬────────────────────┘   └──────────────────────┬───────────────┘
                   │                                               │
                   ▼                                               ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                   طبقة محولات النقل الموحدة (حدود الدخول Ingress Boundary)        │
│                                                                                  │
│   HTTPTransportContext                                  GRPCTransportContext     │
│   ├── يغلف *http.Request                                 ├── يغلف metadata.MD    │
│   └── يحقق TransportContext                             └── يحقق TC              │
│                                                                                  │
│   ┌──────────────────────────────────────────────────────────────────────────┐   │
│   │                      خط أنابيب المعالجة المسبقة (6 مراحل)                │   │
│   │                                                                          │   │
│   │  [1] وسم البروتوكول وتسجيل الدخول (HTTP مقابل GRPC)                       │   │
│   │  [2] الربط والتتبع الموزع (X-Request-ID و W3C traceparent)               │   │
│   │  [3] تطبيع البيانات الوصفية (IP العميل، الترويسات، وكيل المستخدم)         │   │
│   │  [4] المصادقة واستخراج الهوية الأمنية SecurityPrincipal                   │   │
│   │  [5] التحقق النحوي من الحمولة وفصل المخطط                                │   │
│   │  [6] المهل الزمنية للسياق وحدود معدل الطلبات                             │   │
│   └─────────────────────────────────────┬────────────────────────────────────┘   │
└─────────────────────────────────────────┼────────────────────────────────────────┘
                                          │
                                          ▼ سياق Context مثرى + كائن DTO نقي
┌──────────────────────────────────────────────────────────────────────────────────┐
│                قلب التطبيق النقي / حالة الاستخدام (خالٍ تماماً من استيرادات الشبكة)│
│                                                                                  │
│   type OrderUseCase interface {                                                  │
│       CreateOrder(ctx context.Context, req CreateOrderDTO) (*OrderResult, error) │
│   }                                                                              │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## الواجهات الأساسية وعقود البيانات (Core Interfaces & Contracts)

### 1. تعريف البروتوكول والهوية الأمنية (Protocol & Security Principal)

```go
package transport

import (
 "context"
 "time"
)

// TransportProtocol يحدد قناة الدخول الشبكية التي سلمت الطلب.
type TransportProtocol string

const (
 ProtocolHTTP TransportProtocol = "HTTP"
 ProtocolGRPC TransportProtocol = "GRPC"
)

func (p TransportProtocol) String() string {
 return string(p)
}

// SecurityPrincipal يمثل هوية المتصل المعتمدة، مفصولة تماماً عن
// بروتوكول النقل أو الترويسات أو ترميز الرموز.
type SecurityPrincipal struct {
 UserID    string    `json:"user_id"`
 TenantID  string    `json:"tenant_id,omitempty"`
 Roles     []string  `json:"roles,omitempty"`
 Scopes    []string  `json:"scopes,omitempty"`
 Subject   string    `json:"subject,omitempty"`
 IsAdmin   bool      `json:"is_admin"`
 IssuedAt  time.Time `json:"issued_at,omitempty"`
 ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// TransportInfo يحتوي على البيانات الوصفية التشخيصية للنقل المحقونة داخل context.Context.
type TransportInfo struct {
 Protocol  TransportProtocol `json:"protocol"`
 RequestID string            `json:"request_id"`
 ClientIP  string            `json:"client_ip"`
 UserAgent string            `json:"user_agent"`
}
```

### 2. واجهة `TransportContext` المجردة

```go
// TransportContext يجرد خصائص الطلب الوارد عبر بروتوكولي HTTP و gRPC،
// مما يتيح بناء معترضات وسجلات وتقييم أمني قابل لإعادة الاستخدام دون استيراد حزم الشبكة.
type TransportContext interface {
 // Protocol يعيد ما إذا كان الطلب قد وصل عبر HTTP أو gRPC.
 Protocol() TransportProtocol

 // Context يعيد سياق Go القياسي المقترن بالطلب.
 Context() context.Context

 // RequestID يعيد معرف التتبع والارتباط الموزع.
 RequestID() string

 // ClientIP يستخرج عنوان IP القياسي للعميل بعد حل ترويسات الوكيل المعتمدة.
 ClientIP() string

 // UserAgent يعيد بصمة برنامج العميل.
 UserAgent() string

 // Header يعيد قيمة البيانات الوصفية بشكل غير حساس لحالة الأحرف.
 Header(key string) string

 // Principal يسترجع الهوية الأمنية التي تم التحقق منها إن وجدت.
 Principal() (*SecurityPrincipal, bool)

 // SetPrincipal يربط هوية أمنية معتمدة بسياق النقل.
 SetPrincipal(principal *SecurityPrincipal)
}
```

### 3. مفاتيح السياق الآمنة من التصادم (Collision-Free Context Keys)

```go
package transport

import "context"

type contextKey int

const (
 transportInfoKey contextKey = iota
 principalKey
)

// WithTransportInfo يحقن بيانات النقل التشخيصية في context.Context.
func WithTransportInfo(ctx context.Context, info TransportInfo) context.Context {
 return context.WithValue(ctx, transportInfoKey, info)
}

// GetTransportInfo يسترجع بيانات النقل التشخيصية من context.Context.
func GetTransportInfo(ctx context.Context) (TransportInfo, bool) {
 info, ok := ctx.Value(transportInfoKey).(TransportInfo)
 return info, ok
}

// WithPrincipal يحقن هوية SecurityPrincipal معتمدة في context.Context.
func WithPrincipal(ctx context.Context, p *SecurityPrincipal) context.Context {
 return context.WithValue(ctx, principalKey, p)
}

// GetPrincipal يسترجع الهوية الأمنية المعتمدة من context.Context.
func GetPrincipal(ctx context.Context) (*SecurityPrincipal, bool) {
 p, ok := ctx.Value(principalKey).(*SecurityPrincipal)
 return p, ok && p != nil
}
```

---

## محرك خط أنابيب المعالجة المسبقة المكون من 6 مراحل

```go
package transport

import (
 "context"
 "fmt"
 "log/slog"
 "strings"
 "time"
)

type TokenValidator interface {
 ValidateToken(ctx context.Context, token string) (*SecurityPrincipal, error)
}

type PreHandlingPipeline struct {
 validator TokenValidator
 logger    *slog.Logger
 timeout   time.Duration
}

func NewPreHandlingPipeline(validator TokenValidator, logger *slog.Logger, defaultTimeout time.Duration) *PreHandlingPipeline {
 if defaultTimeout <= 0 {
  defaultTimeout = 10 * time.Second
 }
 return &PreHandlingPipeline{
  validator: validator,
  logger:    logger,
  timeout:   defaultTimeout,
 }
}

// Execute ينسق المراحل الست قبل تنفيذ منطق الأعمال.
func (p *PreHandlingPipeline) Execute(tc TransportContext) (context.Context, error) {
 start := time.Now()
 protocol := tc.Protocol()
 reqID := tc.RequestID()
 clientIP := tc.ClientIP()

 // المرحلتان 1 و 2: وسم البروتوكول وتسجيل الارتباط
 p.logger.Info("Ingress request received at transport boundary",
  slog.String("protocol", protocol.String()),
  slog.String("request_id", reqID),
  slog.String("client_ip", clientIP),
  slog.String("user_agent", tc.UserAgent()),
 )

 // المرحلة 3: تطبيع البيانات الوصفية وحقن سياق النقل
 ctx := tc.Context()
 ctx = WithTransportInfo(ctx, TransportInfo{
  Protocol:  protocol,
  RequestID: reqID,
  ClientIP:  clientIP,
  UserAgent: tc.UserAgent(),
 })

 // المرحلة 4: المصادقة وحل الهوية الأمنية
 authHeader := tc.Header("Authorization")
 if authHeader != "" {
  token := strings.TrimPrefix(authHeader, "Bearer ")
  token = strings.TrimSpace(token)
  if token != "" && p.validator != nil {
   principal, err := p.validator.ValidateToken(ctx, token)
   if err != nil {
    p.logger.Warn("Authentication rejected at transport boundary",
     slog.String("protocol", protocol.String()),
     slog.String("request_id", reqID),
     slog.Any("error", err),
    )
    return nil, fmt.Errorf("unauthenticated: %w", err)
   }
   tc.SetPrincipal(principal)
   ctx = WithPrincipal(ctx, principal)
  }
 }

 // المرحلة 6: فرض المهلة الزمنية للسياق (إذا لم تكن محددة مسبقاً)
 if _, hasDeadline := ctx.Deadline(); !hasDeadline && p.timeout > 0 {
  var cancel context.CancelFunc
  ctx, cancel = context.WithTimeout(ctx, p.timeout)
  _ = cancel // تدار عبر المتصل أو اكتمال غلاف النقل
 }

 p.logger.Debug("Pre-handling pipeline successfully completed",
  slog.String("protocol", protocol.String()),
  slog.Duration("duration_us", time.Since(start)),
 )

 return ctx, nil
}
```

---

## محولات الدخول للبروتوكولين (Dual Protocol Ingress Adapters)

### 1. برمجية HTTP الوسيطة (`net/http`)

```go
package transport

import (
 "crypto/rand"
 "encoding/hex"
 "net/http"
)

func HTTPMiddleware(pipeline *PreHandlingPipeline) func(http.Handler) http.Handler {
 return func(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
   reqID := r.Header.Get("X-Request-ID")
   if reqID == "" {
    reqID = generateID()
   }
   w.Header().Set("X-Request-ID", reqID)

   tc := NewHTTPTransportContext(r, reqID)
   enrichedCtx, err := pipeline.Execute(tc)
   if err != nil {
    http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
    return
   }

   next.ServeHTTP(w, r.WithContext(enrichedCtx))
  })
 }
}

func generateID() string {
 b := make([]byte, 16)
 _, _ = rand.Read(b)
 return hex.EncodeToString(b)
}
```

### 2. معترض gRPC الأحادي (`google.golang.org/grpc`)

```go
package transport

import (
 "context"

 "google.golang.org/grpc"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/metadata"
 "google.golang.org/grpc/status"
)

func GRPCUnaryInterceptor(pipeline *PreHandlingPipeline) grpc.UnaryServerInterceptor {
 return func(
  ctx context.Context,
  req any,
  info *grpc.UnaryServerInfo,
  handler grpc.UnaryHandler,
 ) (any, error) {
  md, _ := metadata.FromIncomingContext(ctx)
  var reqID string
  if vals := md.Get("x-request-id"); len(vals) > 0 {
   reqID = vals[0]
  } else {
   reqID = generateID()
  }

  _ = grpc.SetHeader(ctx, metadata.Pairs("x-request-id", reqID))
  tc := NewGRPCTransportContext(ctx, reqID)

  enrichedCtx, err := pipeline.Execute(tc)
  if err != nil {
   return nil, status.Error(codes.Unauthenticated, err.Error())
  }

  return handler(enrichedCtx, req)
 }
}
```

---

## مصفوفة مطابقة أخطاء النطاق (Domain Error Mapping Matrix)

| خطأ النطاق (Domain Error) | رمز حالة HTTP | عنوان مشكلة RFC 7807 | رمز حالة gRPC | الوصف الهندسي |
| :--- | :--- | :--- | :--- | :--- |
| `ErrNotFound` | `404 Not Found` | Resource Not Found | `codes.NotFound` | الكيان المطلوب غير موجود |
| `ErrUnauthorized` | `401 Unauthorized` | Authentication Required | `codes.Unauthenticated` | بيانات الاعتماد مفقودة أو غير صالحة |
| `ErrForbidden` | `403 Forbidden` | Access Denied | `codes.PermissionDenied` | المتصل موثق ولكنه يفتقر للصلاحية |
| `ErrInvalidInput` | `400 Bad Request` | Invalid Input | `codes.InvalidArgument` | فشل في التحقق النحوي أو تنسيق البيانات |
| `ErrConflict` | `409 Conflict` | Resource Conflict | `codes.AlreadyExists` | انتهاك لقيد فريد أو تضارب في المورد |
| `ErrPreconditionFailed` | `412 Precondition` | Precondition Failed | `codes.FailedPrecondition` | فشل القفل التفاؤلي أو عدم تطابق الحالة |
| `ErrRateLimited` | `429 Too Many Req` | Rate Limit Exceeded | `codes.ResourceExhausted` | تجاوز الحصة أو عتبة معدل الطلبات |
| `ErrDeadlineExceeded` | `504 Timeout` | Gateway Timeout | `codes.DeadlineExceeded` | انقضاء المهلة الزمنية المحددة للسياق |
| `ErrInternal` / Panic | `500 Server Error` | Internal Server Error | `codes.Internal` | خطأ غير متوقع أو انهيار غير معالج |

---

## قائمة التحقق للإنتاجية والجاهزية (Production Checklist)

- [ ] **انعدام استيراد البروتوكولات في حالات الاستخدام**: تحقق عبر `go vet` أو أدوات الفحص المخصصة من أن حزم `application/` و `domain/` لا تستورد `net/http` أو `google.golang.org/grpc`.
- [ ] **مفاتيح السياق غير مصدرة**: تأكد من أن جميع المفاتيح المستخدمة مع `context.WithValue` هي أنواع خاصة غير مصدرة أو أعداد صحيحة لمنع التصادم.
- [ ] **الحل الآمن لعنوان IP للعميل**: جرد ترويسات الوكيل غير الموثوقة (`X-Forwarded-For`) إلا إذا كانت خلف بوابة دخول موثوقة ومحددة بدقة.
- [ ] **التعافي من الانهيار عند الحدود (Panic Recovery)**: تأكد من أن برمجية HTTP الوسيطة ومعترضات gRPC تستخدم `recover()` لمنع توقف العملية بأكملها.
- [ ] **ترويسات غير حساسة لحالة الأحرف**: تأكد من أن مفاتيح `metadata.MD` في gRPC تكون بحروف صغيرة دائماً وأن ترويسات HTTP تستخدم `http.CanonicalHeaderKey`.
- [ ] **القياس المنظم عن بُعد**: تأكد من إرفاق `TransportInfo` (البروتوكول، معرف الطلب، IP العميل) بجميع رسائل `slog` ومسارات OpenTelemetry.
