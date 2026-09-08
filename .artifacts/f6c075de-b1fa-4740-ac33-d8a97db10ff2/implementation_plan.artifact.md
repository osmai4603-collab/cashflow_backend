# إضافة ملفات الاختبار المفقودة (Missing Tests Implementation Plan)

بناءً على التحليل السابق للملفات التي تفتقر إلى اختبارات، تهدف هذه الخطة إلى تغطية الفجوات الحرجة في منطق الأعمال (Use Cases) ومحولات الـ HTTP.

## المقترحات الرئيسية

1.  **اختبار ZATCA Processor**: يعتبر محرك الفوترة الإلكترونية السعودي (ZATCA) حرجاً للغاية ويتطلب اختبارات دقيقة لترميز TLV وتوليد XML وتوقيع الفواتير.
2.  **اختبارات موديول الولاء (Loyalty)**: تغطية منطق الاسترداد (Redeem) وتطبيق القواعد المعقدة.
3.  **اختبارات HTTP Handlers**: التأكد من أن جميع الـ endpoints تستجيب بشكل صحيح وتتعامل مع الأخطاء.

## التغييرات المقترحة

### [Accounting]

#### [NEW] [zatca_processor_test.go](file:///home/osm/StudioProjects/cashflow_backend/internal/usecase/accounting/zatca_processor_test.go)
- اختبار `GenerateXML`: التأكد من توليد هيكل UBL 2.1 صحيح.
- اختبار `GenerateQRCode`: التحقق من ترميز Tag-Length-Value (TLV) وتحويله إلى Base64 حسب متطلبات هيئة الزكاة والضريبة والجمارك.
- اختبار `GetTransactionType`: التحقق من التصنيف الصحيح للفواتير (ضريبية، مبسطة، إشعار دائن/مدين).

### [Loyalty]

#### [NEW] [redeem_test.go](file:///home/osm/StudioProjects/cashflow_backend/internal/usecase/loyalty/redeem_test.go)
- اختبار `ApplyCode`: التحقق من تفعيل الأكواد لبرامج الكوبونات والبرومو كود.
- اختبار `EvaluateRules`: التأكد من مطابقة شروط الحد الأدنى للمبلغ والكمية والمنتجات المؤهلة.

### [Activity]

#### [NEW] [handler_test.go](file:///home/osm/StudioProjects/cashflow_backend/internal/adapters/http/activity/handler_test.go)
- اختبارات CRUD للأنشطة.
- اختبار تحويل حالة النشاط إلى "مكتمل" (Done).

## خطة التحقق

### الاختبارات المؤتمتة
- تشغيل `go test ./internal/usecase/accounting/...` للتأكد من نجاح اختبارات ZATCA.
- تشغيل `go test ./internal/usecase/loyalty/...` للتأكد من تغطية منطق المكافآت.
- تشغيل `go test ./internal/adapters/http/...` للتأكد من سلامة الـ API.

### التحقق اليدوي
- لا يتطلب الأمر التحقق اليدوي لأن هذه تغييرات في ملفات الاختبار فقط.
