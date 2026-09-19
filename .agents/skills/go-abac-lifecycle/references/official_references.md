# المعايير الرسمية والمراجع المعمارية لـ ABAC

توثق هذه الصفحة المعايير الدولية المعتمدة، وتوثيقات لغة Go الرسمية، والتطبيقات المرجعية الرائدة التي تستند إليها معمارية دورة حياة التحكم بالوصول القائم على السمات (ABAC).

---

## 1. التوثيقات الرسمية للغة Go (`go.dev`)

- [معمارية أمان Go وقرارات التصميم](https://go.dev/doc/security/): نظرة عامة رسمية حول قرارات تصميم Go المتعلقة بالأمان، وأمان الذاكرة، وتقليل التبعيات الخارجية، والاستقلالية عن أطر العمل.
- [أفضل الممارسات الأمنية لمطوري Go](https://go.dev/security/best-practices): توجيه رسمي يغطي الفحص الآلي للثغرات (`govulncheck`)، واكتشاف سباق البيانات (`-race`)، واختبارات التشويش العشوائي (`-fuzz`).
- [قاعدة بيانات ثغرات Go](https://go.dev/security/vuln/): تتبع لحظي للنشرات والتحذيرات الأمنية في منظومة Go.
- [توثيق حزمة السياق القياسية `context`](https://pkg.go.dev/context): أفضل الممارسات لنقل القيم وإلغاء العمليات ومنع تصادم المفاتيح باستخدام أنواع خاصة غير مصدّرة.
- [حزمة المقارنة الزمنية الثابتة `crypto/subtle`](https://pkg.go.dev/crypto/subtle): المقارنة في زمن ثابت لمنع هجمات القنوات الجانبية الزمنية (Timing Attacks) أثناء فحص التوكنات والسمات الحساسة.
- [حزمة العمليات الذرية `sync/atomic`](https://pkg.go.dev/sync/atomic): تقنيات المؤشرات الذرية (`atomic.Pointer`) للاستبدال اللحظي الخالي من الأقفال للسياسات.

---

## 2. المعايير الدولية الرسمية لإدارة الوصول

- **NIST SP 800-162**: *Guide to Attribute Based Access Control (ABAC) Definition and Considerations*. المعيار الفيدرالي التأسيسي الذي يحدد معمارية ABAC، وتصنيف السمات (Subject, Resource, Action, Environment)، والمكونات الوظيفية (PEP, PDP, PIP, PAP).
- **OASIS XACML 3.0**: *eXtensible Access Control Markup Language (XACML) Version 3.0*. المواصفات القياسية لخوارزميات دمج القرارات (`deny-overrides`, `permit-overrides`, `first-applicable`) ودلالات تقييم السياسات.
- **RFC 7807**: *Problem Details for HTTP APIs*. التنسيق القياسي المهيكل لاستجابات أخطاء HTTP بصيغة JSON (`application/problem+json`) دون تسريب قواعد التفويض الداخلية.
- **RFC 7519**: *JSON Web Token (JWT)*. المعيار العالمي لنقل هوية الفاعل وادعاءاته الموثقة بأمان.
- **OWASP API Security Top 10**: ثغرات API1:2023 (Broken Object Level Authorization - BOLA)، و API5:2023 (Broken Function Level Authorization - BFLA).

---

## 3. التطبيقات المرجعية عالية الإنتاجية في Go

- [مفوض ABAC في Kubernetes (`k8s.io/apiserver`)](https://github.com/kubernetes/kubernetes/tree/master/pkg/apis/abac): التطبيق الأكثر انتشاراً عالمياً لنموذج ABAC المكتوب بلغة Go الصافية، ويوضح واجهات السمات النقية (`authorizer.Attributes`).
- [محرك Google CEL لـ Go (`google/cel-go`)](https://github.com/google/cel-go): لغة التعبيرات العامة المطورة من Google بلغة Go، وتتميز بأمان الذاكرة والسرعة الفائقة، وتعتمد عليها شروط Google Cloud IAM و Kubernetes و Envoy/Istio.
- [محرك Open Policy Agent (OPA)](https://github.com/open-policy-agent/opa): مشروع متخرج من مؤسسة CNCF ومكتوب بالكامل في Go، يتيح تقييم السياسات ككود تصريحي (Policy-as-Code) للخدمات المصغرة السحابية.
- [مكتبة Casbin للتفويض](https://github.com/casbin/casbin): محرك التفويض متعدد النماذج الرائد في Go، ويدعم نماذج ABAC و RBAC والمطابقات المخصصة.
