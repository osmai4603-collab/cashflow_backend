# حراس وسائط PEP ومعمارية فرض RFC 7807 في Go

توضح هذه الوثيقة كيفية بناء نقاط فرض سياسات (PEPs) قوية ومرنة في Go، تشمل وسائط HTTP، ومعترضات gRPC، ومغلفات أخطاء RFC 7807 الآمنة لمنع تسريب بيانات النظام.

---

## 1. استراتيجية الفرض ثنائية المستويات (Dual-Tier Enforcement)

في المعمارية النظيفة لـ Go، يُفرض التحكم بالوصول على مستويين منفصلين:

```text
الطلب القادم
     │
     ▼
┌────────────────────────────────────────────────────────┐
│ المستوى 1: حارس PEP عند البروتوكول (HTTP / gRPC)       │
│ النطاق: حراسة المسارات بالتفويض الخشن                  │
│ الفحص: هل يحمل الفاعل صلاحية "invoices:create"؟        │
└──────────────────────────┬─────────────────────────────┘
                           │ (مسموح)
                           ▼
┌────────────────────────────────────────────────────────┐
│ المستوى 2: الفرض داخل طبقة التطبيق والمجال             │
│ النطاق: فحص الكائن الدقيق وحدود المستأجر               │
│ الفحص: هل يطابق مستأجر الفاعل حقل invoice.company_id؟  │
└────────────────────────────────────────────────────────┘
```

---

## 2. تطبيق وسيط HTTP PEP

يجب أن يقوم وسيط HTTP الإنتاجي بـ:
1. استخراج والتحقق من الهوية من `r.Context()`.
2. تقييم الصلاحيات عبر محرك الذاكرة الحية.
3. معالجة حالات الرفض بأمان دون تسريب أي معلومات عن بنية النظام.

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

func WriteProblemDetails(w http.ResponseWriter, status int, title, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Type:     "https://golang.org/errors/access-denied",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	})
}

func Guard(engine *Engine, logger *slog.Logger, required Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			subject, ok := SubjectFromContext(r.Context())
			if !ok {
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
				return
			}

			if !engine.HasPermission(subject.Roles, required) {
				logger.Warn("تم رفض وصول RBAC",
					"subject_id", subject.ID,
					"roles", subject.Roles,
					"required_permission", required,
					"path", r.URL.Path,
					"latency_us", time.Since(start).Microseconds(),
				)
				// منع تسريب المعلومات: لا تذكر "نقص صلاحية: invoices:delete" للعميل الخارجي
				WriteProblemDetails(w, http.StatusForbidden, "Forbidden", "ليس لديك الصلاحية الكافية للوصول لهذا المورد", r.URL.Path)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

---

## 3. معترض gRPC الأحادي لحراسة المسارات (Unary Interceptor PEP Guard)

للخدمات المصغرة التي تعتمد على gRPC، يتم تطبيق معترض أحادي:

```go
package rbac

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MethodPermissionMap map[string]Permission

func UnaryServerInterceptor(engine *Engine, methodPerms MethodPermissionMap) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		required, exists := methodPerms[info.FullMethod]
		if !exists {
			// الفشل الآمن: إذا لم تكن الدالة مسجلة، تمنع افتراضياً
			return nil, status.Errorf(codes.PermissionDenied, "تم رفض الوصول وفق السياسة الافتراضية")
		}

		subject, ok := SubjectFromContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "متصل غير مصادق عليه")
		}

		if !engine.HasPermission(subject.Roles, required) {
			return nil, status.Errorf(codes.PermissionDenied, "صلاحيات غير كافية")
		}

		return handler(ctx, req)
	}
}
```
