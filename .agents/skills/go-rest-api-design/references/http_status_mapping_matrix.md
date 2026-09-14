# مصفوفة تحويل أخطاء الأعمال إلى رموز HTTP (HTTP Status Mapping Matrix)

تحدد هذه المصفوفة كيفية تحويل أخطاء النطاق البرمجية (Domain Errors) إلى رموز استجابة HTTP دقيقة وفق أفضل الممارسات.

---

## مصفوفة الأخطاء ورموز الاستجابة

| خطأ النطاق البرمجي | المعنى التشغيلي | رمز HTTP | كود الـ RFC 7807 المقترح |
| :--- | :--- | :---: | :--- |
| **`ErrNotFound`** | الكيان المطلوب غير موجود في قاعدة البيانات | `404 Not Found` | `RESOURCE_NOT_FOUND` |
| **`ErrUnauthorized`** | غياب رمز المصادقة أو انتهاء صلاحيته | `401 Unauthorized` | `UNAUTHENTICATED` |
| **`ErrForbidden`** | المستخدم مسجل ولكن ليس لديه صلاحية الوصول | `403 Forbidden` | `PERMISSION_DENIED` |
| **`ErrValidation`** | فشل في صيغة الإدخال (بريد غير صالح، حقل مفقود) | `422 Unprocessable` أو `400` | `VALIDATION_ERROR` |
| **`ErrConflict`** | الكيان موجود مسبقاً (مثل بريد مكرر) | `409 Conflict` | `RESOURCE_ALREADY_EXISTS` |
| **`ErrBusinessRule`** | انتهاك قاعدة أعمال (محاولة سحب رصيد غير كافٍ) | `422 Unprocessable Entity` | `BUSINESS_RULE_VIOLATION` |
| **`ErrIdempotencyConflict`** | طلب مكرر بنفس المفتاح قيد المعالجة الآن | `409 Conflict` | `IDEMPOTENCY_IN_FLIGHT` |
| **`ErrInternal`** | خطأ غير متوقع في قاعدة البيانات أو النظام الداخلي | `500 Internal Error` | `INTERNAL_SERVER_ERROR` |
