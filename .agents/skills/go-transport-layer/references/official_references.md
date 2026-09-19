# المعايير الرسمية والمراجع المعمارية لطبقة النقل (Official References)

توثق هذه الصفحة المعايير القياسية والتوثيقات الرسمية للغة Go والتطبيقات المرجعية التي تستند إليها معمارية طبقة النقل الموحدة لبروتوكولي HTTP و gRPC.

---

## 1. التوثيق الرسمي للغة Go (`go.dev`)

- [حزمة المكتبة القياسية `net/http`](https://pkg.go.dev/net/http): المرجع الحاسم لواجهات `http.Handler` و `http.ResponseWriter` و `http.Request` وتنسيق الترويسات القياسي (`CanonicalHeaderKey`).
- [حزمة المكتبة القياسية `context`](https://pkg.go.dev/context): أفضل الممارسات لإلغاء العمليات والمهل الزمنية وتجنب تصادم المفاتيح عبر الأنواع غير المصدرة.
- [حزمة المكتبة القياسية `log/slog`](https://pkg.go.dev/log/slog): التسجيل الهيكلي عالي الأداء لحدود الدخول الشبكي.
- [حزمة المكتبة القياسية `net`](https://pkg.go.dev/net): معالجة وتقسيم عناوين IP (`net.SplitHostPort`) والبدائيات الشبكية للمستمعات.

---

## 2. التوثيق الرسمي لنظام gRPC و RPC

- [التوثيق الرسمي لـ gRPC في Go](https://grpc.io/docs/languages/go/): أدلة شاملة للمعترضات الأحادية ومعترضات التدفق (Unary & Streaming Interceptors).
- [توثيق حزمة gRPC في Go (`google.golang.org/grpc`)](https://pkg.go.dev/google.golang.org/grpc): تفاصيل `UnaryServerInterceptor` و `StreamServerInterceptor` ودورة حياة البيانات الوصفية.
- [مرجع بيانات gRPC الوصفية (`google.golang.org/grpc/metadata`)](https://pkg.go.dev/google.golang.org/grpc/metadata): دلالات أزواج المفتاح والقيمة لترويسات الطلب والاستجابة (`metadata.MD`).
- [مواصفات ConnectRPC](https://connectrpc.com/): الإرشادات الرسمية لتوحيد RPC عبر HTTP/1.1 و HTTP/2 و `http.Handler` القياسي.
- [مشروع gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway): نمط الوكيل العكسي لترجمة REST/JSON إلى gRPC.
- [معمارية النقل في Go kit](https://gokit.io/docs/architecture/): التصميم التأسيسي ثلاثي الطبقات (Transport -> Endpoint -> Service).

---

## 3. المعايير الدولية ومواصفات RFC

- **RFC 7807**: *تفاصيل المشكلات لواجهات برمجية تطبيقات HTTP (Problem Details for HTTP APIs)*. التنسيق المعياري لحمولات أخطاء HTTP القابلة للقراءة آلياً (`application/problem+json`).
- **RFC 9110**: *دلالات HTTP (HTTP Semantics)*. المعايير المعتمدة لرموز حالة HTTP، وعدم التكرار الحسابي (Idempotency)، وتحليل الترويسات.
- **مواصفات W3C Trace Context**: الترويسات المعيارية `traceparent` و `tracestate` للتتبع الموزع عبر الخدمات المصغرة باستخدام HTTP و gRPC.
- **RFC 7519**: *رمز ويب JSON (JWT)*. التنسيق القياسي لرموز الحامل المنقولة عبر ترويسة `Authorization`.
- **RFC 7239**: *امتداد التمرير لـ HTTP (Forwarded HTTP Extension)*. البنية القياسية لترويسات تمرير الوكيل (`Forwarded` و `X-Forwarded-For`).
