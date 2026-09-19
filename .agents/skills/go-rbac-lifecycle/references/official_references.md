# المعايير الرسمية والمراجع المعمارية لـ RBAC

توثق هذه الصفحة المعايير الدولية المعتمدة، وتوثيقات لغة Go الرسمية، والتطبيقات المرجعية الرائدة التي تستند إليها معمارية دورة حياة التحكم بالوصول القائم على الأدوار (RBAC).

---

## 1. التوثيقات الرسمية للغة Go (`go.dev`)

- [معمارية وفلسفة أمان Go](https://go.dev/doc/security/): نظرة عامة رسمية حول قرارات تصميم لغة Go المتعلقة بالأمان، وأمان الذاكرة، والاستقلالية عن أطر العمل.
- [أفضل الممارسات الأمنية لمطوري Go](https://go.dev/security/best-practices): توجيه رسمي يغطي الفحص الآلي للثغرات (`govulncheck`)، واكتشاف سباق البيانات (`-race`)، واختبارات التشويش (`-fuzz`).
- [قاعدة بيانات ثغرات Go](https://go.dev/security/vuln/): متابعة لحظية للنشرات الأمنية في منظومة Go.
- [توثيق حزمة السياق القياسية `context`](https://pkg.go.dev/context): القواعد القياسية لنقل القيم وإلغاء العمليات واستخدام المفاتيح الخاصة غير المصدّرة.
- [حزمة المقارنة الزمنية الثابتة `crypto/subtle`](https://pkg.go.dev/crypto/subtle): المقارنة في زمن ثابت لمنع هجمات التوقيت الجانبية.
- [دليل Go الفعال (Effective Go)](https://go.dev/doc/effective_go): المعايير الاصطلاحية لتصميم الحزم البرمجية وأنماط التزامن.

---

## 2. المعايير الدولية الرسمية لإدارة الوصول

- **NIST SP 800-21d**: *Proposed NIST Standard for Role-Based Access Control*. الأساس الرياضي والهندسي لنماذج RBAC: الأساسي، الهرمي، المقيد، والمتناظر.
- **ANSI/INCITS 359-2012**: *American National Standard for Information Technology - Role Based Access Control*. المعيار المعتمد دولياً لأنظمة RBAC.
- **NIST SP 800-162**: *Guide to Attribute Based Access Control (ABAC) Definition and Considerations*. المرجع للنماذج الهجينة التي تجمع بين RBAC و ABAC.
- **RFC 7807**: *Problem Details for HTTP APIs*. التنسيق القياسي المهيكل لأخطاء HTTP (`application/problem+json`).
- **RFC 7519**: *JSON Web Token (JWT)*. المعيار العالمي لنقل ادعاءات وهوية الفاعل الموثقة بأمان.

---

## 3. التطبيقات المرجعية عالية الإنتاجية في Go

- [محرك التفويض في Kubernetes (`k8s.io/apiserver`)](https://github.com/kubernetes/kubernetes/tree/master/pkg/apis/rbac): المعيار الذهبي لفصل تعريف الأدوار (Roles) عن ربطها بالمستخدمين (RoleBindings).
- [دليل مراجع RBAC في Kubernetes](https://kubernetes.io/docs/reference/access-authn-authz/rbac/): نموذج النشر الإنتاجي للأدوار وأدوار العنقود (ClusterRoles) والأفعال والمستخدمين.
- [مكتبة Casbin للتفويض](https://github.com/casbin/casbin): محرك Go مفتوح المصدر الرائد المعتمد على نموذج PERM.
