# التصميم التكراري الآمن للمخطط (Idempotent Schema Design)

في أنظمة الترحيل الآلية، يجب أن تكون نصوص الـ SQL قابلة لإعادة التنفيذ الآمن دون أن تنهار إذا تم تشغيلها مرتين بالخطأ (Idempotency).

---

## 1. مبادئ الـ Idempotency في نصوص DDL

- **إنشاء الجداول**:

  ```sql
  CREATE TABLE IF NOT EXISTS accounts (
      id UUID PRIMARY KEY,
      balance NUMERIC(15, 4) NOT NULL DEFAULT 0,
      created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
  );
  ```

- **إضافة الأعمدة**:

  ```sql
  ALTER TABLE accounts ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'USD';
  ```

- **إنشاء الفهارس**:

  ```sql
  CREATE INDEX IF NOT EXISTS idx_accounts_currency ON accounts (currency);
  ```

---

## 2. جدول تتبع الإصدارات (`schema_migrations`)

تعتمد المهارة على جدول داخلي في قاعدة البيانات يسجل:

1. `version`: رقم الإصدار الفريد (مثل طابع زمني `20260914001`).
2. `name`: الاسم الوصفي للترحيل (مثل `create_invoices_table`).
3. `applied_at`: وقت التنفيذ الدقيق.

عند تشغيل أمر الترحيل (مثل `cmd/migrate`):

1. يفحص المحرك الجدول ويبحث عن الترحيلات التي لم يتم تطبيقها بعد.
2. يرتب الترحيلات الجديدة تصاعدياً برقم الإصدار.
3. يطبق كل ملف `up.sql` داخل Transaction ويسجل نجاحه في الجدول.
