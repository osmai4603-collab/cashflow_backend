# محولات البوابات وسجل المزودين المتعددين (Gateway Adapters & Provider Registry)

في التطبيقات الحديثة والأنظمة المؤسسية، نادراً ما يعتمد النظام على مزود خارجي وحيد لوظيفة معينة. قد يحتاج النظام لدعم عدة مزودين لخدمات البريد (SendGrid, Postmark, AWS SES)، أو بوابات الدفع المتعددة، أو خوادم الرسائل، أو أنظمة التخزين السحابي.

---

## 1. نمط سجل المزودين (The Provider Registry Pattern)

بدلاً من تشعيب منطق الأعمال بعبارات `switch/case` مليئة بالمزودين وتفاصيلهم الملموسة، يُستخدم **سجل آمن متزامن (`ProviderRegistry`)** يُسجل فيه كل محول تحت كود تعريفي فريد:

```go
type OutboundProvider interface {
    ProviderCode() string
    Execute(ctx context.Context, req ExternalRequest) (*ExternalResponse, error)
}

type ProviderRegistry struct {
    mu        sync.RWMutex
    providers map[string]OutboundProvider
}

func NewProviderRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        providers: make(map[string]OutboundProvider),
    }
}

func (r *ProviderRegistry) Register(p OutboundProvider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[p.ProviderCode()] = p
}

func (r *ProviderRegistry) Resolve(code string) (OutboundProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    p, ok := r.providers[code]
    if !ok {
        return nil, fmt.Errorf("provider %q not registered", code)
    }
    return p, nil
}
```

---

## 2. إدارة المحولات الخارجية بحسب تصنيف الخدمة

### أ. محولات الاتصال والتواصل (Communication Gateways)
- **محولات البريد والرسائل (Email / SMS)**:
  - ينفذ المحول واجهة المنفذ المعرفة في النطاق (`NotificationGateway`).
  - يتولى المحول بناء الترويسات والمصادقة (API Keys / Bearer Tokens / Mutual TLS).
  - يترجم رموز الخطأ الخاصة بالمزود إلى أخطاء نطاق مفهومة دون تسريب كائنات المزود.

### ب. محولات الواجهات المالية والتجارية (Financial & Billing Gateways)
- إدارة التوقيع الرقمي، وتشفير حمولات الطلبات.
- فرض ترويسات عدم التكرار (`Idempotency-Key`) على كافة العمليات التعديلية.
- التعامل الحذر مع العمليات المشكوك فيها (In-Doubt) عند انتهاء المهل.

### ج. محولات تكامل الأنظمة الخارجية (Enterprise RPC & SaaS Integration)
- ترجمة طلبات النطاق إلى بروتوكولات gRPC أو REST أو SOAP/XML الخاصة بالأنظمة الخارجية.
- إدارة جلسات المصادقة وإعادة تجديد الرموز آلياً (Token Refresh) عند انتهائها دون إشعار طبقة حالات الاستخدام.
