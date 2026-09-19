# مضاعفة المنافذ والربط ثنائي الاتجاه للأخطاء (Port Multiplexing & Error Mapping)

تغطي هذه الوثيقة جانبين محوريين في طبقة النقل في Go: تقديم حركة مرور كل من بروتوكولي HTTP و gRPC على منفذ TCP واحد (مضاعفة المنافذ Port Multiplexing)، وترجمة أخطاء النطاق الداخلية إلى رموز الحالة الخاصة بكل بروتوكول (رموز حالة HTTP وتفاصيل المشاكل RFC 7807 مقابل رموز حالة gRPC).

---

## 1. استراتيجيات مضاعفة المنافذ (Port Multiplexing Strategies)

في بيئات الحاويات (Kubernetes و AWS ECS و Cloud Run) وشبكات الخدمات المصغرة، يساهم الاستماع على منفذ شبكي واحد في تبسيط توجيه الدخول، وإعدادات DNS، وموزعات الأحمال (Load Balancers).

### الاستراتيجية الأولى: مضاعفة الاتصالات عبر `cmux`

تفحص مكتبة `cmux` البايتات الأولى من اتصال TCP الوارد لتحديد البروتوكول قبل توجيه المقبس (Socket) إلى خادم gRPC أو خادم HTTP.

```go
package main

import (
 "log"
 "net"
 "net/http"

 "github.com/soheilhy/cmux"
 "google.golang.org/grpc"
)

func RunMultiplexedServer(addr string, grpcSrv *grpc.Server, httpHandler http.Handler) error {
 listener, err := net.Listen("tcp", addr)
 if err != nil {
  return err
 }

 m := cmux.New(listener)

 // 1. يستخدم gRPC بروتوكول HTTP/2 مع Content-Type: application/grpc
 grpcL := m.MatchWithWriters(
  cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
 )

 // 2. حركة مرور HTTP/1.1 تطابق أساليب HTTP القياسية
 httpL := m.Match(cmux.HTTP1Fast())

 go func() {
  if err := grpcSrv.Serve(grpcL); err != nil {
   log.Printf("خطأ في خادم gRPC: %v", err)
  }
 }()

 httpSrv := &http.Server{Handler: httpHandler}
 go func() {
  if err := httpSrv.Serve(httpL); err != nil && err != http.ErrServerClosed {
   log.Printf("خطأ في خادم HTTP: %v", err)
  }
 }()

 log.Printf("الخادم المزدوج يعمل بنجاح على %s", addr)
 return m.Serve()
}
```

### الاستراتيجية الثانية: مضاعفة HTTP/2 الأصلية عبر `net/http`

عند التشغيل عبر TLS أو HTTP/2 النصي الواضح (h2c)، يمكن استدعاء `grpc.Server.ServeHTTP` مباشرة من معالج `http.Handler` الرئيسي:

```go
func UnifiedHandler(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
 return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
   grpcServer.ServeHTTP(w, r)
  } else {
   httpHandler.ServeHTTP(w, r)
  }
 })
}
```

> [!TIP]
> **توصية هندسية للإنتاج**:
> على الرغم من فائدة مضاعفة المنافذ في البيئات المقيدة بمنفذ واحد، فإن تشغيل مستمعين مستقلين (مثل `:8080` لـ HTTP/REST و `:9090` لـ gRPC) هو الخيار الأفضل للخدمات فائقة الإنتاجية والحرجة. هذا التوزيع يمنع مشكلة تجويع اتصالات HTTP/1.1 keep-alive من التأثير على تدفقات gRPC HTTP/2.

---

## 2. الربط ثنائي الاتجاه لأخطاء النطاق (Bidirectional Error Mapping)

تُعرف طبقة التطبيق أخطاء نطاق نقية تماماً، بينما تتولى طبقة النقل مسؤولية ترجمة أخطاء النطاق إلى دلالات البروتوكول المعني بدقة.

### أخطاء النطاق القياسية (Domain Errors)

```go
package domain

import "errors"

var (
 ErrNotFound           = errors.New("resource not found")
 ErrUnauthorized       = errors.New("unauthorized: missing or invalid credentials")
 ErrForbidden          = errors.New("forbidden: insufficient privileges")
 ErrInvalidInput       = errors.New("invalid input data")
 ErrConflict           = errors.New("resource conflict")
 ErrPreconditionFailed = errors.New("precondition failed")
 ErrRateLimited        = errors.New("rate limit exceeded")
 ErrDeadlineExceeded   = errors.New("operation deadline exceeded")
 ErrInternal           = errors.New("internal server error")
)
```

### مصفوفة الربط والمطابقة (Mapping Matrix)

| خطأ النطاق | رمز حالة HTTP | عنوان RFC 7807 | رمز gRPC (`codes.Code`) | التفسير الهندسي |
| :--- | :--- | :--- | :--- | :--- |
| `ErrNotFound` | 404 Not Found | Not Found | `codes.NotFound` | الكيان المستهدف غير موجود |
| `ErrUnauthorized` | 401 Unauthorized | Unauthorized | `codes.Unauthenticated` | هوية المتصل غير مؤكدة |
| `ErrForbidden` | 403 Forbidden | Forbidden | `codes.PermissionDenied` | المتصل يفتقر إلى الأذونات المطلوبة |
| `ErrInvalidInput` | 400 Bad Request | Bad Request | `codes.InvalidArgument` | حمولة غير صالحة أو فشل التحقق النحوي |
| `ErrConflict` | 409 Conflict | Conflict | `codes.AlreadyExists` | تكرار مفتاح فريد أو تحديث متزامن متضارب |
| `ErrPreconditionFailed` | 412 Precondition Failed | Precondition Failed | `codes.FailedPrecondition` | شرط مسبق لحالة الأعمال لم يتحقق |
| `ErrRateLimited` | 429 Too Many Requests | Rate Limit Exceeded | `codes.ResourceExhausted` | تجاوز الحصة أو حد معدل الطلبات |
| `ErrDeadlineExceeded` | 504 Gateway Timeout | Gateway Timeout | `codes.DeadlineExceeded` | إلغاء السياق أو انقضاء المهلة الزمنية |
| `ErrInternal` / panic | 500 Internal Server Error | Internal Server Error | `codes.Internal` | فشل داخلي غير متوقع وغير معالج |

---

## 3. تطبيق ترجمة الأخطاء بلغة Go

```go
package transport

import (
 "encoding/json"
 "errors"
 "net/http"

 "cashflow_backend/internal/domain"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/status"
)

// تمثيل تفاصيل المشكلات وفق معيار RFC 7807
type ProblemDetails struct {
 Type     string `json:"type,omitempty"`
 Title    string `json:"title"`
 Status   int    `json:"status"`
 Detail   string `json:"detail"`
 Instance string `json:"instance,omitempty"`
}

func WriteHTTPError(w http.ResponseWriter, reqID string, err error) {
 status := MapToHTTPStatus(err)
 w.Header().Set("Content-Type", "application/problem+json")
 w.WriteHeader(status)

 problem := ProblemDetails{
  Title:    http.StatusText(status),
  Status:   status,
  Detail:   err.Error(),
  Instance: reqID,
 }
 _ = json.NewEncoder(w).Encode(problem)
}

func MapToHTTPStatus(err error) int {
 switch {
 case errors.Is(err, domain.ErrNotFound):
  return http.StatusNotFound
 case errors.Is(err, domain.ErrUnauthorized):
  return http.StatusUnauthorized
 case errors.Is(err, domain.ErrForbidden):
  return http.StatusForbidden
 case errors.Is(err, domain.ErrInvalidInput):
  return http.StatusBadRequest
 case errors.Is(err, domain.ErrConflict):
  return http.StatusConflict
 case errors.Is(err, domain.ErrPreconditionFailed):
  return http.StatusPreconditionFailed
 case errors.Is(err, domain.ErrRateLimited):
  return http.StatusTooManyRequests
 case errors.Is(err, domain.ErrDeadlineExceeded):
  return http.StatusGatewayTimeout
 default:
  return http.StatusInternalServerError
 }
}

func MapToGRPCError(err error) error {
 code := codes.Internal
 switch {
 case errors.Is(err, domain.ErrNotFound):
  code = codes.NotFound
 case errors.Is(err, domain.ErrUnauthorized):
  code = codes.Unauthenticated
 case errors.Is(err, domain.ErrForbidden):
  code = codes.PermissionDenied
 case errors.Is(err, domain.ErrInvalidInput):
  code = codes.InvalidArgument
 case errors.Is(err, domain.ErrConflict):
  code = codes.AlreadyExists
 case errors.Is(err, domain.ErrPreconditionFailed):
  code = codes.FailedPrecondition
 case errors.Is(err, domain.ErrRateLimited):
  code = codes.ResourceExhausted
 case errors.Is(err, domain.ErrDeadlineExceeded):
  code = codes.DeadlineExceeded
 }
 return status.Error(code, err.Error())
}
```
