# مصفوفة تبعيات الطبقات (Layer Dependency Matrix)

تحدد هذه المصفوفة ما يُسمح لكل طبقة باستيراده (Imports) وما يُحظر عليها قطعياً للحفاظ على نقاء المعمارية واستقلاليتها عن التقنيات الخارجية.

---

## مصفوفة التبعيات المسموحة والمحظورة

| الطبقة (Layer) | المسار في المشروع | الاستيرادات المسموحة (Allowed) | الاستيرادات المحظورة قطعياً (FORBIDDEN) |
| :--- | :--- | :--- | :--- |
| **Domain** | `internal/domain` | حزم Go القياسية البحتة فقط (`time`, `math`, `errors`, `strings`) | `usecase`, `adapters`, `infrastructure`, `platform`, `net/http`, `database/sql`, `pgx`, أي مكتبة خارجية |
| **Use Case** | `internal/usecase` | `internal/domain` + حزم Go القياسية (`context`, `fmt`) | `adapters`, `infrastructure`, `platform`, `net/http`, `database/sql`, `pgx`, أي مكتبة خارجية لأطر العمل |
| **Adapters** | `internal/adapters` | `internal/domain`, `internal/usecase`, حزم Go القياسية | `internal/infrastructure` المباشر (إلا عبر Interfaces), `internal/platform` |
| **Infrastructure** | `internal/infrastructure` | حزم Go القياسية + برامج التشغيل الخارجية (`pgx`, `redis`, `slog`, إلخ) | الربط المباشر مع طبقة `adapters` الخاصة بالـ HTTP |
| **Platform** | `internal/platform` | **جميع الطبقات** (لأنها تمثل جذر التكوين Composition Root) | وضع أي منطق أعمال (Business Logic) داخلها |

---

## أمثلة على المخالفات الشائعة وعواقبها

### المخالفة 1: استيراد مكتبة قاعدة البيانات داخل الـ Usecase

```go
// ❌ خطأ كارثي داخل internal/usecase/invoice
import "github.com/jackc/pgx/v5"

func (uc *UseCase) Execute(ctx context.Context, tx pgx.Tx) error { ... }
```

- **العاقبة**: أصبح منطق أعمال شركتك مقيداً بـ PostgreSQL. لا يمكنك كتابة اختبارات وحدة بدون تشغيل Postgres حقيقي، ولا يمكنك استبدال قاعدة البيانات مستقبلاً.
- **التصحيح**: عرف واجهة منفذ (Port) داخل الـ usecase وانقل `pgx` إلى `internal/adapters/storage/postgres`.

### المخالفة 2: تسريب الـ HTTP Request/Response إلى الـ Domain أو Usecase

```go
// ❌ خطأ داخل internal/usecase/invoice
import "net/http"

func (uc *UseCase) HandleHTTP(w http.ResponseWriter, r *http.Request) { ... }
```

- **العاقبة**: تدمير استقلالية واجهة المستخدم. لا يمكن تشغيل حالة الاستخدام عبر CLI أو gRPC أو رسائل Queue.
- **التصحيح**: اجعل الـ Handler في `internal/adapters/http` يستقبل الـ HTTP ويستخرج منه DTO يحوله إلى أمر `Command` نظيف يمرره إلى الـ Usecase.
