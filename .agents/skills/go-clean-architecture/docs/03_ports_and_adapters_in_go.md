# نمط المنافذ والمحولات في Go (Ports & Adapters)

يُعرف نمط المعمارية السداسية (Hexagonal Architecture) بنمط "المنافذ والمحولات" (Ports & Adapters). في Go، يتم تطبيق هذا النمط بشكل طبيعي جداً بالاعتماد على ميزة الـ Implicit Interfaces.

---

## 1. ما هو المنفذ (Port) وما هو المحول (Adapter)؟

- **المنفذ (Port)**: هو **Interface** يحدد عقداً وظيفياً (Contract) تحتاجه حالة الاستخدام لإنجاز مهمتها. المنفذ يعبر عن **"ماذا"** يريد التطبيق، وليس "كيف".
- **المحول (Adapter)**: هو **Struct ملموس** ينفذ هذا العقد بالتعامل مع تقنية معينة (مثل PostgreSQL، أو AWS S3، أو Stripe API). المحول هو الذي يعرف **"كيف"**.

---

## 2. أين تُعرف الـ Interfaces؟ (القاعدة الذهبية في Go)

> **في لغات مثل Java أو C#، يُعرف المطور Interface في حزمة ويضع تنفيذها في حزمة فرعية. في Go، القاعدة المعيارية هي: عرّف الواجهة في الحزمة التي تستهلكها (Consumer-defined Interface).**

### مثال تطبيقي

داخل `internal/usecase/invoice/create.go`:

```go
package invoice

import (
    "context"
    "cashflow/internal/domain"
)

// المنفذ (Port): يعرف هنا لأن usecase هي التي تحتاجه وتستهلكه
type Repository interface {
    SaveInvoice(ctx context.Context, inv *domain.Invoice) error
    GetNextSequence(ctx context.Context) (int64, error)
}

type CreateInvoiceUseCase struct {
    repo Repository
}

func NewCreateInvoiceUseCase(repo Repository) *CreateInvoiceUseCase {
    return &CreateInvoiceUseCase{repo: repo}
}
```

وداخل `internal/adapters/storage/postgres/invoice_repo.go`:

```go
package postgres

import (
    "context"
    "cashflow/internal/domain"
    "github.com/jackc/pgx/v5/pgxpool"
)

// المحول (Adapter): ينفذ الواجهة تلقائياً وبشكل ضمني دون الحاجة لكلمة implements
type InvoiceRepository struct {
    pool *pgxpool.Pool
}

func (r *InvoiceRepository) SaveInvoice(ctx context.Context, inv *domain.Invoice) error {
    _, err := r.pool.Exec(ctx, "INSERT INTO invoices ...", inv.ID, inv.Amount)
    return err
}

func (r *InvoiceRepository) GetNextSequence(ctx context.Context) (int64, error) {
    var seq int64
    err := r.pool.QueryRow(ctx, "SELECT nextval('invoice_seq')").Scan(&seq)
    return seq, err
}
```

---

## 3. إدارة المعاملات المالية (Transactions across Repositories)

في الأنظمة المحاسبية (مثل Cashflow)، تحتاج العديد من العمليات إلى تحديث أكثر من جدول داخل Transaction واحدة:

- **المعضلة**: كيف نفتح Transaction في قاعدة البيانات دون تسريب كائن `*pgx.Tx` إلى طبقة الـ Usecase؟
- **الحل المعماري**: استخدام نمط `UnitOfWork` أو ممرر السياق الذري:

```go
type UnitOfWork interface {
    Do(ctx context.Context, fn func(ctx context.Context) error) error
}
```

تستدعي الـ Usecase دالة `uow.Do(ctx, func(txCtx) { ... })` وتمرر `txCtx` لمستودعات البيانات، بينما يتولى المحول الملموس في `adapters/storage` فتح الـ Transaction والـ Commit أو الـ Rollback تلقائياً.

---

## 4. كائنات نقل البيانات والمحولات (DTOs & Mappers)

- **طلب الـ HTTP**: يحتوي على حقول بصيغة JSON قد تحتوي على نصوص أو أرقام تحتاج إلى تحقق (Validation).
- **المحول (HTTP Handler)**: يفكك الـ JSON إلى هيكل `CreateInvoiceRequestDTO`.
- **التحويل (Mapping)**: يقوم الـ Handler أو دالة Mapper بتحويل الـ DTO إلى `usecase.CreateInvoiceCommand`.
- **الأمان**: لا يسمح أبداً للـ JSON بالوصول مباشرة إلى كائنات الـ Domain دون مرور بمرحلة التحقق والتحويل.
