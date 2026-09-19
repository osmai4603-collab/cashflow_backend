# عزل بيانات المستأجرين المتعددين ونطاق السياق في Go

يتطلب تعدد المستأجرين (Multi-Tenancy) في خدمات Go المؤسسية فصلاً تاماً ومطلقاً لبيانات الشركات والمنظمات (`company_id` / `tenant_id`). يؤدي أي إخفاق في عزل المستأجرين إلى ثغرات تجاوز الصلاحيات على مستوى الكائن (BOLA / IDOR)، مما يتيح للمخترقين قراءة أو تعديل السجلات السرية لشركات أخرى.

---

## 1. مبدأ عزل المستأجر غير القابل للاختراق

1. **المصدر الوحيد للحقيقة (Source of Truth)**: يجب أن يصدر معرف المستأجر **حصرياً** من براهين اعتماد تم التحقق منها تشفيرياً (مثل ادعاءات JWT الموثقة أو جلسة خادم آمنة). ولا يجوز **إطلاقاً** قبوله من مدخلات العميل غير الموثوقة كمعاملات الروابط (`?company_id=123`) أو جسم الطلب أو ترويسات عشوائية.
2. **التمرير عبر السياق (Context Propagation)**: بمجرد التحقق من الهوية، يُربط معرف المستأجر بـ `context.Context` عبر نوع مفتاح خاص غير مصدّر.
3. **التصفية الإلزامية في الاستعلامات**: كل استعلام لقاعدة البيانات أو تحديث أو حذف على جداول المستأجرين يجب أن يتضمن شرط المستأجر في جملة `WHERE`:

   ```sql
   SELECT id, title, amount FROM invoices WHERE id = $1 AND company_id = $2;
   UPDATE invoices SET status = $1 WHERE id = $2 AND company_id = $3;
   DELETE FROM invoices WHERE id = $1 AND company_id = $2;
   ```

4. **تكامل المفاتيح الأجنبية (Foreign Keys)**: في المخططات متعددة المستأجرين، يجب أن تشير المفاتيح المركبة والأجنبية إلى معرف المستأجر لضمان السلامة المرجعية.

---

## 2. نمط تمرير المستأجر عبر السياق في Go

يجب استخدام أنواع بنى غير مصدّرة لمنع تصادم المفاتيح بين الحزم البرمجية:

```go
package auth

import (
 "context"
 "errors"
)

type contextKey struct{ name string }

var tenantContextKey = &contextKey{name: "tenant_context"}

var ErrMissingTenant = errors.New("سياق المستأجر مفقود من الطلب")

// TenantContext يحتوي على بيانات المستأجر الموثقة غير القابلة للتعديل
type TenantContext struct {
 CompanyID string
 UserID    string
 Roles     []string
}

// WithTenant يعيد سياقاً جديداً محتوياً على سياق المستأجر الموثق
func WithTenant(ctx context.Context, tenant TenantContext) context.Context {
 return context.WithValue(ctx, tenantContextKey, tenant)
}

// FromContext يستخرج سياق المستأجر بأمان مع تأكيد النوع
func FromContext(ctx context.Context) (TenantContext, error) {
 val := ctx.Value(tenantContextKey)
 if val == nil {
  return TenantContext{}, ErrMissingTenant
 }
 tc, ok := val.(TenantContext)
 if !ok {
  return TenantContext{}, errors.New("نوع سياق المستأجر غير صالح في السياق")
 }
 return tc, nil
}
```

---

## 3. الدفاع في العمق على مستوى طبقة المستودعات (Repository Layer)

لا تعتمد أبداً على موجه المسارات أو وحدات التحكم وحدها لفرض العزل. يجب أن تفرض المستودعات قيد المستأجر مباشرة:

### التطبيق الصحيح: تمرير المستأجر من السياق إلى الاستعلام

```go
func (r *InvoiceRepository) FindByID(ctx context.Context, id string) (*Invoice, error) {
 tenant, err := auth.FromContext(ctx)
 if err != nil {
  return nil, fmt.Errorf("المستودع يتطلب سياق المستأجر: %w", err)
 }

 query := `
  SELECT id, company_id, title, amount, status, created_at 
  FROM invoices 
  WHERE id = $1 AND company_id = $2`

 var inv Invoice
 err = r.db.QueryRowContext(ctx, query, id, tenant.CompanyID).Scan(
  &inv.ID, &inv.CompanyID, &inv.Title, &inv.Amount, &inv.Status, &inv.CreatedAt,
 )
 if errors.Is(err, sql.ErrNoRows) {
  return nil, ErrNotFound
 }
 return &inv, err
}
```

### التطبيق الخاطئ (غير الآمن): قبول معرف الشركة كمعامل غير موثق

```go
// خطأ أمني فادح: إذا مرر المتصل companyID من جسم JSON غير موثق، تحدث ثغرة IDOR!
func (r *InvoiceRepository) FindByIDInsecure(ctx context.Context, id, companyID string) (*Invoice, error) {
    // ...
}
```

---

## 4. أمان مستوى السطر في قاعدة البيانات (PostgreSQL RLS)

لمبدأ الدفاع في العمق، يمكن دمج ميزة Row-Level Security في PostgreSQL مع فحوصات التطبيق. عند سحب اتصال من تجمع الاتصالات:

```sql
SET LOCAL app.current_company_id = 'tenant_123';
```

وفي ملفات هجرة قاعدة البيانات:

```sql
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON invoices
    FOR ALL
    TO application_role
    USING (company_id = current_setting('app.current_company_id', true));
```

يضمن هذا أنه حتى لو نسي المطور كتابة `AND company_id = $2` في أحد الاستعلامات، فإن محرك PostgreSQL سيستبعد أسطر الشركات الأخرى تلقائياً.
