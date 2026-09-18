package authlifecycle

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================================================
// 1. أخطاء الحراسة القياسية ومغلفات RFC 7807 (Sentinel Errors & Problem Details)
// ============================================================================

var (
	// ErrUnauthenticated يعاد عند غياب أو فساد البراهين التشفيرية
	ErrUnauthenticated = errors.New("security: unauthenticated, invalid or missing credentials")
	// ErrTokenExpired يعاد حصراً عند انتهاء الصلاحية الزمنية للرمز
	ErrTokenExpired = errors.New("security: token has expired")
	// ErrPermissionDenied يعاد عند ثبوت الهوية مع عجز الصلاحيات (403 Forbidden)
	ErrPermissionDenied = errors.New("security: permission denied, insufficient privileges")
	// ErrInvalidToken يعاد عند فشل التحقق من التوقيع أو تزوير خوارزمية التوقيع
	ErrInvalidToken = errors.New("security: malformed or invalid token signature")
	// ErrResourceNotFound يعاد لإخفاء وجود المورد عند الرغبة في منع تسريب البيانات
	ErrResourceNotFound = errors.New("security: resource not found")
)

// ProblemDetails مغلف قياسي موحد لتفاصيل المشاكل البرمجية وفق RFC 7807
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// WriteProblemDetails يصوغ استجابة مشكلة برمجية بصيغة JSON وفق معيار RFC 7807
func WriteProblemDetails(w http.ResponseWriter, status int, title, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Type:     fmt.Sprintf("https://httpstatuses.com/%d", status),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	})
}

// ============================================================================
// 2. عقود بيانات الهوية والمجال الموحدة (Identity & Domain Contracts)
// ============================================================================

// Principal يمثل كائن الهوية النهائي الموثق والمطهر وغير القابل للتعديل
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

// HasRole يفحص انتساب الهوية لدور معين في زمن فوري O(1)
func (p *Principal) HasRole(role string) bool {
	if p == nil {
		return false
	}
	_, ok := p.Roles[role]
	return ok
}

// HasScope يفحص امتلاك الرمز لنطاق تفويض OAuth معين في زمن فوري O(1)
func (p *Principal) HasScope(scope string) bool {
	if p == nil {
		return false
	}
	_, ok := p.Scopes[scope]
	return ok
}

// Action يحدد نوع العملية المطلوبة على المورد التجاري
type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionApprove Action = "approve"
)

// Resource يمثل الكيان التجاري الخاضع لتقييم السياسة الدقيقة (PDP)
type Resource struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// CustomClaims يمثل ادعاءات JWT الخاصة بنظامنا بالإضافة للادعاءات القياسية
type CustomClaims struct {
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Scopes   []string `json:"scopes"`
	jwt.RegisteredClaims
}

// ============================================================================
// 3. الواجهات الموحدة المحايدة للبروتوكول (Protocol-Agnostic Interfaces)
// ============================================================================

// HeaderCarrier واجهة تجريدية لقراءة الترويسات أياً كان البروتوكول الشبكي
type HeaderCarrier interface {
	Get(key string) string
}

// HTTPHeaderCarrier محوّل يتيح قراءة ترويسات *http.Request عبر HeaderCarrier
type HTTPHeaderCarrier struct {
	req *http.Request
}

func NewHTTPHeaderCarrier(r *http.Request) HTTPHeaderCarrier {
	return HTTPHeaderCarrier{req: r}
}

func (h HTTPHeaderCarrier) Get(key string) string {
	return h.req.Header.Get(key)
}

// MapHeaderCarrier ناقل تخزين بالذاكرة (مثالي لاختبارات الوحدة ومعترضات gRPC)
type MapHeaderCarrier map[string]string

func (m MapHeaderCarrier) Get(key string) string {
	if val, ok := m[strings.ToLower(key)]; ok {
		return val
	}
	return m[key]
}

// TokenValidator واجهة فك وتدقيق الرموز الرقمية تشفيرياً واستخراج الهوية
type TokenValidator interface {
	ValidateToken(ctx context.Context, rawToken string) (*Principal, error)
}

// Authorizer واجهة محرك اتخاذ قرار التفويض (Policy Decision Point - PDP)
type Authorizer interface {
	Authorize(ctx context.Context, sub *Principal, act Action, res Resource) (bool, error)
}

// ============================================================================
// 4. إدارة السياق بأمان تام (Safe Context Propagation)
// ============================================================================

// نوع بنية خاصة غير مصدّرة يضمن استحالة التصادم بين الحزم في السياق
type contextKey struct{}

var principalContextKey = contextKey{}

// InjectPrincipal يحقن كائن الهوية Principal بأمان داخل context.Context
func InjectPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, p)
}

// ExtractPrincipal يسترجع كائن الهوية من السياق مع التحقق من النوع
func ExtractPrincipal(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalContextKey).(*Principal)
	return p, ok && p != nil
}

// MustExtractPrincipal يسترجع الهوية أو يرجع خطأ ErrUnauthenticated للمسارات المحمية
func MustExtractPrincipal(ctx context.Context) (*Principal, error) {
	p, ok := ExtractPrincipal(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	return p, nil
}

// ============================================================================
// 5. مدقق الرموز التشفيري ومكافحة الخلط الخوارزمي (JWT Token Validator)
// ============================================================================

// JWTTokenValidator مدقق إنتاجي لرموز JWT يعتمد على مفاتيح RSA العامة
type JWTTokenValidator struct {
	publicKey *rsa.PublicKey
	issuer    string
	audience  string
	leeway    time.Duration
}

// NewJWTTokenValidator ينشئ مدقق رموز جديد مع تحديد هامش التوقيت الزمني
func NewJWTTokenValidator(pubKey *rsa.PublicKey, issuer, audience string, leeway time.Duration) *JWTTokenValidator {
	if leeway <= 0 {
		leeway = 30 * time.Second
	}
	return &JWTTokenValidator{
		publicKey: pubKey,
		issuer:    issuer,
		audience:  audience,
		leeway:    leeway,
	}
}

// ValidateToken يتحقق تشفيرياً من صحة التوقيع والادعاءات ومكافحة هجمات Alg Confusion
func (v *JWTTokenValidator) ValidateToken(ctx context.Context, rawToken string) (*Principal, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, ErrUnauthenticated
	}

	token, err := jwt.ParseWithClaims(
		rawToken,
		&CustomClaims{},
		func(t *jwt.Token) (any, error) {
			// الحماية الصارمة ضد هجوم خلط الخوارزميات (Alg Confusion: منع "none" أو HMAC مع مفتاح RSA)
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("%w: unexpected signing algorithm %v", ErrInvalidToken, t.Header["alg"])
			}
			return v.publicKey, nil
		},
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithLeeway(v.leeway),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrUnauthenticated
	}

	// تحويل الأدوار والنطاقات إلى خرائط للبحث الفوري O(1)
	rolesMap := make(map[string]struct{}, len(claims.Roles))
	for _, r := range claims.Roles {
		rolesMap[r] = struct{}{}
	}

	scopesMap := make(map[string]struct{}, len(claims.Scopes))
	for _, s := range claims.Scopes {
		scopesMap[s] = struct{}{}
	}

	var issuedAt, expiresAt time.Time
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &Principal{
		ID:         claims.Subject,
		TenantID:   claims.TenantID,
		Roles:      rolesMap,
		Scopes:     scopesMap,
		IssuedAt:   issuedAt,
		ExpiresAt:  expiresAt,
		Attributes: make(map[string]any),
	}, nil
}

// ============================================================================
// 6. وسائط النقل ومعترضات gRPC (Transport Middlewares & Interceptors)
// ============================================================================

// HTTPAuthMiddleware وسيط مصادقة طلبات HTTP وحقن كائن Principal في السياق
func HTTPAuthMiddleware(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Missing Authorization header", r.URL.Path)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Invalid bearer authorization scheme", r.URL.Path)
				return
			}

			principal, err := validator.ValidateToken(r.Context(), parts[1])
			if err != nil {
				if errors.Is(err, ErrTokenExpired) {
					WriteProblemDetails(w, http.StatusUnauthorized, "Token Expired", "Access token has expired", r.URL.Path)
					return
				}
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Invalid or malformed credentials", r.URL.Path)
				return
			}

			ctx := InjectPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireScopeHTTPMiddleware وسيط تفويض خشن يتحقق من نطاقات OAuth2 ويرجع 403 Forbidden عند النقص
func RequireScopeHTTPMiddleware(requiredScope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := MustExtractPrincipal(r.Context())
			if err != nil {
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
				return
			}

			if !principal.HasScope(requiredScope) {
				WriteProblemDetails(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("Missing required scope: %s", requiredScope), r.URL.Path)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoleHTTPMiddleware وسيط تفويض خشن يتحقق من الأدوار العامة ويرجع 403 Forbidden عند النقص
func RequireRoleHTTPMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := MustExtractPrincipal(r.Context())
			if err != nil {
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
				return
			}

			if !principal.HasRole(requiredRole) {
				WriteProblemDetails(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("Missing required role: %s", requiredRole), r.URL.Path)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// أنواع معترضات gRPC القياسية والمحايدة
type UnaryHandler func(ctx context.Context, req any) (any, error)

type UnaryServerInfo struct {
	FullMethod string
}

type UnaryServerInterceptor func(ctx context.Context, req any, info *UnaryServerInfo, handler UnaryHandler) (any, error)

// GRPCUnaryAuthInterceptor معترض مصادقة أحادي لـ gRPC
func GRPCUnaryAuthInterceptor(validator TokenValidator, extractCarrier func(ctx context.Context) (HeaderCarrier, bool)) UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *UnaryServerInfo, handler UnaryHandler) (any, error) {
		carrier, ok := extractCarrier(ctx)
		if !ok {
			return nil, ErrUnauthenticated
		}

		rawAuth := carrier.Get("authorization")
		if rawAuth == "" {
			return nil, ErrUnauthenticated
		}

		parts := strings.SplitN(rawAuth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, ErrUnauthenticated
		}

		principal, err := validator.ValidateToken(ctx, parts[1])
		if err != nil {
			return nil, ErrUnauthenticated
		}

		newCtx := InjectPrincipal(ctx, principal)
		return handler(newCtx, req)
	}
}

// GRPCUnaryScopeInterceptor معترض تفويض خشن أحادي لـ gRPC لفحص النطاقات
func GRPCUnaryScopeInterceptor(requiredScope string) UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *UnaryServerInfo, handler UnaryHandler) (any, error) {
		principal, err := MustExtractPrincipal(ctx)
		if err != nil {
			return nil, ErrUnauthenticated
		}

		if !principal.HasScope(requiredScope) {
			return nil, ErrPermissionDenied
		}

		return handler(ctx, req)
	}
}

// ============================================================================
// 7. طبقة التطبيق والتفويض الدقيق (Domain Fine-Grained PEP & PIP)
// ============================================================================

// Invoice كيان الفاتورة التجاري في طبقة المجال
type Invoice struct {
	ID       string  `json:"id"`
	TenantID string  `json:"tenant_id"`
	OwnerID  string  `json:"owner_id"`
	Status   string  `json:"status"` // "draft", "approved", "paid"
	Amount   float64 `json:"amount"`
}

// InvoiceRepository واجهة مستودع الفواتير (Policy Information Point - PIP)
type InvoiceRepository interface {
	GetByID(ctx context.Context, tenantID, invoiceID string) (*Invoice, error)
	Update(ctx context.Context, inv *Invoice) error
}

// ApproveInvoiceUseCase حالة استخدام معتمدة توضح التفويض الدقيق وعزل المستأجر
type ApproveInvoiceUseCase struct {
	repo       InvoiceRepository
	authorizer Authorizer
}

func NewApproveInvoiceUseCase(repo InvoiceRepository, authorizer Authorizer) *ApproveInvoiceUseCase {
	return &ApproveInvoiceUseCase{
		repo:       repo,
		authorizer: authorizer,
	}
}

func (uc *ApproveInvoiceUseCase) Execute(ctx context.Context, invoiceID string) error {
	// 1. استخراج الهوية الموثقة والمطهرة من السياق
	principal, err := MustExtractPrincipal(ctx)
	if err != nil {
		return ErrUnauthenticated
	}

	// 2. استدعاء المستودع (PIP) مع تقييد صارم بمعرف المستأجر لمنع ثغرات BOLA
	inv, err := uc.repo.GetByID(ctx, principal.TenantID, invoiceID)
	if err != nil {
		return err
	}
	if inv == nil {
		return ErrResourceNotFound
	}

	// 3. تمثيل الكيان ككائن مورد (Resource) بكامل سماته
	res := Resource{
		Type:     "invoice",
		ID:       inv.ID,
		TenantID: inv.TenantID,
		Attributes: map[string]any{
			"owner_id": inv.OwnerID,
			"status":   inv.Status,
			"amount":   inv.Amount,
		},
	}

	// 4. تقييم السياسة الدقيقة عبر استدعاء محرك القرارات (PDP)
	allowed, err := uc.authorizer.Authorize(ctx, principal, ActionApprove, res)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrPermissionDenied
	}

	// 5. فحص قيود وقواعد الأعمال وتعديل الحالة
	if inv.Status != "draft" {
		return errors.New("business rule: only draft invoices can be approved")
	}

	inv.Status = "approved"
	return uc.repo.Update(ctx, inv)
}
