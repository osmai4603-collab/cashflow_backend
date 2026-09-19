# التحكم بالوصول القائم على الأدوار (RBAC) والصلاحيات الدقيقة في Go

يربط نموذج التحكم بالوصول القائم على الأدوار (RBAC) بين الهويات والأدوار، وبين الأدوار وصلاحيات تشغيلية محددة. في المعمارية النظيفة لـ Go، يجب فصل التفويض تماماً عن منطق الأعمال مع الحفاظ على قابلية التدقيق الأمني.

---

## 1. الأدوار مقابل الصلاحيات (Roles vs Permissions)

أحد الأخطاء الشائعة هو فحص الأدوار مباشرة في كود الأعمال (مثل `if user.Role == "admin"`). هذا يخلق كوداً هشاً يتطلب تعديلات مستمرة عند إضافة أدوار جديدة (مثل `auditor`, `finance_manager`).

- **الصلاحيات (Permissions - العمليات)**: إمكانات ذرية محددة، مثل: `invoices:create`, `invoices:approve`, `reports:export`.
- **الأدوار (Roles - الحزم)**: مجموعات منطقية من الصلاحيات تُسند للمستخدمين، مثل:
  - `Admin`: `["*"]`
  - `Accountant`: `["invoices:create", "invoices:read", "invoices:update"]`
  - `Auditor`: `["invoices:read", "reports:read"]`

---

## 2. معمارية التفويض على مستويين (Two-Tier Authorization)

تفرض تطبيقات Go الإنتاجية التفويض عند نقطتي فحص متميزتين:

```text
┌────────────────────────────────────────────────────────┐
│ 1. مستوى المسار والوسيط (حارس خشن عند النقل)           │
│    "هل يملك المتصل صلاحية استدعاء POST /api؟"           │
│    RequiresPermission("invoices:create")               │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│ 2. مستوى المجال وحالة الاستخدام (حارس سياسة دقيق)       │
│    "هل يستطيع هذا المستخدم اعتماد فاتورة تتجاوز $50k؟" │
│    policy.CanApproveAmount(user, invoice.Amount)       │
└────────────────────────────────────────────────────────┘
```

---

## 3. نمط وسيط حراسة المسار (Route Guard Middleware)

يجب أن يكون وسيط المسار موجزاً وقابلاً للتركيب بسهولة مع توقيع `net/http` القياسي:

```go
// RequirePermission ينشئ وسيط HTTP يتحقق من امتلاك المتصل للصلاحية المطلوبة
func RequirePermission(authorizer Authorizer, requiredPerm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, err := auth.FromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !authorizer.HasPermission(tenant.Roles, requiredPerm) {
				http.Error(w, `{"error":"forbidden","message":"صلاحيات غير كافية"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

---

## 4. مطابقة الأنماط والهياكل الهرمية (Wildcard Matching)

دعم مطابقة الصلاحيات الهرمية باستخدام النقطتين الرأسيتين (`:`):

- `*` يطابق كافة العمليات والصلاحيات.
- `invoices:*` يطابق `invoices:read`، و `invoices:create`، و `invoices:approve`.
- `invoices:read` يطابق فقط `invoices:read`.

```go
func MatchPermission(grantedPattern, required string) bool {
	if grantedPattern == "*" || grantedPattern == required {
		return true
	}
	if strings.HasSuffix(grantedPattern, ":*") {
		prefix := strings.TrimSuffix(grantedPattern, ":*")
		return strings.HasPrefix(required, prefix+":")
	}
	return false
}
```
