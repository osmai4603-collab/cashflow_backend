# المرحلة 25: التقارير المتقدمة ولوحات المعلومات - خطة التنفيذ التفصيلية

## الهدف والنطاق

تنفيذ نظام تقارير متقدم ولوحات معلومات (Dashboards) في مشروع Go بناءً على سلوك Odoo 19.0. يهدف هذا النظام إلى تحويل البيانات المالية والتشغيلية الخام إلى معلومات منظمة تدعم اتخاذ القرار، مع توفير مرونة في بناء التقارير عبر محرك حسابي (Reporting Engine) يدعم التجميع، التصفية، والمقارنة الزمنية.

## المرجعية من Odoo 19.0
- **نظام التقارير**: `addons/account/models/account_report.py` والكيانات المرتبطة (`line`, `expression`, `column`).
- **محركات الحساب**: `domain`, `account_codes`, `aggregation`, `custom`.
- **لوحات المعلومات**: `addons/spreadsheet_dashboard/` و `addons/digest/`.

## هيكل النظام المقترح (Architecture)

1. **محرك التقارير (Reporting Engine)**:
   - كيانات مرنة لتعريف التقارير برمجياً أو عبر قاعدة البيانات.
   - دعم التدرج الهرمي (Hierarchy) للخطوط.
   - دعم محركات الحساب المختلفة (SQL-based, Logic-based).

2. **التقارير المالية المتقدمة**:
   - ميزان المراجعة التفصيلي (Trial Balance).
   - دفتر الأستاذ العام ودفتر أستاذ الشركاء (General/Partner Ledger).
   - تقادم المستحقات (Aged Receivable/Payable).
   - قائمة التدفقات النقدية (Cash Flow Statement).

3. **لوحات معلومات KPI**:
   - مؤشرات الأداء للمحاسبة، المبيعات، المخزون، والموارد البشرية.
   - دعم المقارنة مع فترات سابقة وتتبع الاتجاهات (Trends).

---

## المراحل وحالة التنفيذ

| المرحلة | النطاق | الحالة | معيار الإغلاق |
|---|---|---|---|
| 0 | تصميم المحرك والعقود (Domain & Engine) | ✅ مكتملة | توثيق الكيانات (`Report`, `Line`, `Expression`) والواجهات |
| 1 | تنفيذ محرك الحسابات (Calculation Engine) | ✅ مكتملة | نجاح استخراج البيانات بناءً على Account Codes و Aggregation |
| 2 | التقارير المالية (Financial Reports) | ✅ مكتملة | تنفيذ P&L و Balance Sheet و Trial Balance المتقدمين |
| 3 | التقارير التشغيلية (Operational Reports) | ✅ مكتملة | تنفيذ تقارير المخزون (Valuation) وتحليل المبيعات |
| 4 | نظام لوحات المعلومات (Dashboards & KPIs) | ✅ مكتملة | تنفيذ endpoints للـ KPIs وتجميع البيانات للوحات المعلومات |
| 5 | واجهات HTTP والتصفية (API & Filters) | ✅ مكتملة | دعم الفلاتر المتقدمة (التاريخ، الشركة، التحليلي، الشريك) |
| 6 | الاختبارات النهائية والتحقق | ✅ مكتملة | نجاح اختبارات الدقة والموازنة (IsBalanced) وتكامل البيانات |

---

## المرحلة 0: تصميم المحرك والعقود (Domain & Engine)

الهدف هو استبدال الهياكل الثابتة في `reports.go` بنظام ديناميكي.

1. **تعريف الكيانات (internal/domain/report/report.go)**:
   - `Report`: الاسم، التسلسل، القوالب، الفلاتر المتاحة.
   - `ReportLine`: الاسم، الكود، الأب، التجميع حسب (Group By).
   - `ReportExpression`: المحرك (Engine)، الصيغة (Formula)، النطاق الزمني.
   - `ReportColumn`: الاسم، التسمية التوضيحية، نوع البيانات.

2. **واجهة المحرك (ports.go)**:
   - `ReportGenerator`: وظيفة تأخذ تعريف التقرير + خيارات (Options) وتعيد `ReportResult`.

---

## المرحلة 1: تنفيذ محرك الحسابات (Calculation Engine)

محاكاة محركات Odoo:
1. **Account Codes Engine**: جلب الأرصدة بناءً على بادئة كود الحساب (مثلاً `101*`).
2. **Aggregation Engine**: حساب سطر بناءً على أسطر أخرى (مثلاً `NET_PROFIT = TOTAL_INCOME - TOTAL_EXPENSES`).
3. **Tax Tags Engine**: جلب المبالغ المرتبطة بضرائب محددة.
4. **Domain Engine**: استعلامات SQL مخصصة مع فلاتر ديناميكية.

---

## المرحلة 2: التقارير المالية (Financial Reports)

إعادة بناء التقارير الأساسية باستخدام المحرك الجديد:
- **Balance Sheet**: تقسيم الأصول (Current/Non-current) والخصوم وحقوق الملكية.
- **Trial Balance**: أرصدة افتتاحية، حركات (مدين/دائن)، أرصدة ختامية.
- **Aged Reports**: تقسيم المستحقات حسب المدة (1-30, 31-60, 61-90, +90 يوم).

---

## المرحلة 3: التقارير التشغيلية (Operational Reports)

توسيع النظام ليشمل الموديولات الأخرى:
- **Stock Valuation**: القيمة الحالية للمخزون بناءً على `StockValuationLayer`.
- **Sales Analysis**: تحليل المبيعات حسب المنتج، العميل، والمنطقة.
- **Expense Analysis**: تحليل المصروفات حسب الموظف والحساب التحليلي.

---

## المرحلة 4: نظام لوحات المعلومات (Dashboards & KPIs)

تنفيذ منطق تجميع المؤشرات (internal/usecase/report/dashboard_usecase.go):
1. **Financial KPIs**: صافي الربح، هامش الربح، نسبة السيولة.
2. **Sales KPIs**: إجمالي المبيعات، عدد الطلبات، متوسط قيمة الطلب.
3. **Inventory KPIs**: معدل دوران المخزون، قيمة البضاعة التالفة.
4. **HR KPIs**: معدل الحضور، المصروفات المعلقة.

---

## المرحلة 5: واجهات HTTP والتصفية (API & Filters)

توفير Endpoints مرنة:
- `GET /api/v1/reports/{report_code}` مع Query Params:
  - `date_from`, `date_to`
  - `company_id`
  - `analytic_account_ids`
  - `partner_ids`
  - `comparison` (true/false)
- `GET /api/v1/dashboards/{dashboard_type}`

---

## المرحلة 6: الاختبارات النهائية والتحقق

1. **Unit Tests**: لاختبار محركات الحساب (Aggregation, Account Codes).
2. **Integration Tests**: التحقق من مطابقة تقرير ميزان المراجعة مع قيود اليومية الفعلية.
3. **Manual Verification**: مقارنة النتائج مع مخرجات Odoo لنفس البيانات التجريبية.

---

## مراجع التنفيذ وتبعيات الملفات

- **الملفات الجديدة**:
  - `internal/domain/report/`
  - `internal/usecase/report/`
  - `internal/adapters/http/report/`
  - `internal/adapters/storage/report/`
- **الملفات المراد تعديلها**:
  - `internal/domain/accounting/reports.go` (تحديث أو استبدال)
  - `internal/usecase/accounting/reports_usecase.go`
- **مصدر Odoo**:
  - `odoo-19.0/addons/account/models/account_report.py`
  - `odoo-19.0/addons/spreadsheet_dashboard/models/spreadsheet_dashboard.py`
