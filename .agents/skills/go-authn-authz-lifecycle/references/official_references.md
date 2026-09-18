# المعايير الدولية والمراجع الرسمية (Official Standards & References)

توثق هذه الصفحة المرجعيات القياسية العالمية الصادرة عن هيئات المعايير الدولية (IETF و NIST)، والتوثيقات الرسمية للغة Go، ومشاريع المصادر المفتوحة المعتمدة التي تؤطر دورات حياة المصادقة والتفويض.

---

## 1. المعايير القياسية الدولية (IETF & NIST Standards)

- **RFC 6749**: *The OAuth 2.0 Authorization Framework*
  - الرابط الرسمي: [datatracker.ietf.org/doc/html/rfc6749](https://datatracker.ietf.org/doc/html/rfc6749)
  - يحدد أدوار المنظومة (Resource Owner, Resource Server, Client, Authorization Server) وتدفقات إصدار الرموز.
- **RFC 6750**: *The OAuth 2.0 Authorization Framework: Bearer Token Usage*
  - الرابط الرسمي: [datatracker.ietf.org/doc/html/rfc6750](https://datatracker.ietf.org/doc/html/rfc6750)
  - يحدد صياغة ترويسة النقل القياسية `Authorization: Bearer <token>` والمتطلبات الأمنية لحامل الرمز.
- **RFC 7519**: *JSON Web Token (JWT)*
  - الرابط الرسمي: [datatracker.ietf.org/doc/html/rfc7519](https://datatracker.ietf.org/doc/html/rfc7519)
  - يحدد المعيار الهيكلي لرموز JWT وادعاءاتها المعيارية (`iss`, `sub`, `aud`, `exp`, `nbf`, `iat`, `jti`).
- **RFC 9068**: *JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens*
  - الرابط الرسمي: [datatracker.ietf.org/doc/html/rfc9068](https://datatracker.ietf.org/doc/html/rfc9068)
  - المعيار الإلزامي الحديث لهيكلة حمولة رموز الوصول المعتمدة على JWT لضمان التشغيل البيني الآمن.
- **RFC 7662**: *OAuth 2.0 Token Introspection*
  - الرابط الرسمي: [datatracker.ietf.org/doc/html/rfc7662](https://datatracker.ietf.org/doc/html/rfc7662)
  - بروتوكول الاستعلام المباشر من خادم الهوية للتحقق من حالة الرموز المبهمة (Opaque Tokens).
- **RFC 7807**: *Problem Details for HTTP APIs*
  - الرابط الرسمي: [datatracker.ietf.org/doc/html/rfc7807](https://datatracker.ietf.org/doc/html/rfc7807)
  - التنسيق القياسي لمغلفات أخطاء JSON لرموز الحالة (`401 Unauthorized`, `403 Forbidden`).
- **NIST Special Publication 800-162**: *Guide to Attribute Based Access Control (ABAC)*
  - الرابط الرسمي: [csrc.nist.gov/publications/detail/sp/800-162/final](https://csrc.nist.gov/publications/detail/sp/800-162/final)
  - المرجع الهندسي الشامل لنمذجة كيانات PEP و PDP و PIP و PAP.
- **NIST Special Publication 800-207**: *Zero Trust Architecture*
  - الرابط الرسمي: [csrc.nist.gov/publications/detail/sp/800-207/final](https://csrc.nist.gov/publications/detail/sp/800-207/final)
  - المرجع العالمي لمعمارية الثقة المعدومة وفرض التحقق المستمر عند كل نقطة اتصال شبكية.

---

## 2. المراجع الرسمية للغة Go والأمان (Go Security Ecosystem)

- **توثيق حزمة السياق القياسية `context` في Go**:
  - الرابط: [pkg.go.dev/context](https://pkg.go.dev/context)
  - القواعد الصارمة لمنع تصادم المفاتيح وفرض استخدام أنواع البنى الخاصة غير المصدّرة.
- **توثيق حزمة الشبكة والنقل `net/http` في Go**:
  - الرابط: [pkg.go.dev/net/http](https://pkg.go.dev/net/http)
  - واجهات `http.Handler` وتصميم خطوط وسطاء المعالجة القابلة للتركيب.
- **أفضل الممارسات الأمنية الرسمية لفريق Go**:
  - [go.dev/security/best-practices](https://go.dev/security/best-practices)
  - حزم التشفير القياسية: [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)

---

## 3. محركات السياسات والمواصفات المفتوحة (Policy Engines)

- **Open Policy Agent (OPA)**:
  - المستودع الرسمي: [github.com/open-policy-agent/opa](https://github.com/open-policy-agent/opa)
  - التوثيق الرسمي ومحرك Rego: [openpolicyagent.org](https://www.openpolicyagent.org)
- **Casbin**:
  - المستودع الرسمي: [github.com/casbin/casbin/v2](https://github.com/casbin/casbin/v2)
  - التوثيق ونموذج PERM: [casbin.org](https://casbin.org)
- **Google Zanzibar وتطبيقاته المفتوحة**:
  - الورقة البحثية الأصلية: *Zanzibar: Google's Consistent, Global Authorization System* (USENIX ATC '19).
  - مشروع SpiceDB (Authzed): [github.com/authzed/authzed-go](https://github.com/authzed/authzed-go)
  - مشروع OpenFGA (CNCF Sandbox): [openfga.dev](https://openfga.dev)
