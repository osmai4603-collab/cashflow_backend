# مصفوفة أسبقية الإعدادات والمصادر (Priority Hierarchy Matrix)

تحدد هذه المصفوفة الترتيب الصارم لدمج الإعدادات من مصادرها المختلفة، مع توضيح أمثلة تطبيقية وقواعد التجاوز (Overrides).

---

## مصفوفة الطبقات وقواعد الدمج

| الأسبقية | الطبقة (Layer) | المصدر (Source) | مثال المدخل | الغرض التشغيلي |
| :---: | :--- | :--- | :--- | :--- |
| **5 (الأعلى)** | **Runtime Overrides** | برمجي عبر `WithRuntimeOverrides` | `cfg.Server.Port = "0"` | اختبارات التكامل الآلية والـ Ephemeral Ports |
| **4** | **CLI Flags** | وسائط سطر الأوامر | `--port 9090` | تجاوز سريع ومباشر من قبل مشغل النظام |
| **3** | **Environment** | متغيرات النظام والـ Docker | `PORT=8070`, `DB_HOST=pg` | تكوين الحاويات وبيئات Kubernetes |
| **2** | **Config File** | ملف JSON/YAML على القرص | `config/app.json` | الإعدادات الثابتة لبيئة العمل (Staging/Production) |
| **1 (الأدنى)** | **Defaults** | كود Go الصلب (`Defaults()`) | `Port: "8080"` | قيم آمنة تضمن إقلاع الخادم دون أي ملف خارجي |

---

## مصفوفة الأسماء البديلة (Aliases Mapping)

عند قراءة المتغيرات البيئية، يدعم المحرك الأسماء القياسية السحابية وأسماء PostgreSQL المعتمدة:

| الحقل في Go | المتغير البيئي الأساسي (Canonical) | المتغيرات البديلة المدعومة (Aliases) |
| :--- | :--- | :--- |
| `Server.Port` | `PORT` | `HTTP_PORT`, `CASHFLOW_PORT` |
| `Server.Interface`| `HTTP_INTERFACE` | `HOST`, `BIND_ADDRESS` |
| `Database.Host` | `DB_HOST` | `PGHOST`, `POSTGRES_HOST` |
| `Database.Port` | `DB_PORT` | `PGPORT`, `POSTGRES_PORT` |
| `Database.Name` | `DB_NAME` | `PGDATABASE`, `POSTGRES_DB` |
| `Database.User` | `DB_USER` | `PGUSER`, `POSTGRES_USER` |
| `Database.Password`| `DB_PASSWORD` | `PGPASSWORD`, `POSTGRES_PASSWORD` |
| `Database.DatabaseURL`| `DATABASE_URL` | `DB_URL`, `POSTGRES_URL` |

---

## سلوك المفاتيح الغائبة مقابل الفارغة (Absent vs Empty Keys)

- **المفتاح الغائب (Absent Key)**: إذا لم يرد المفتاح في الملف أو البيئة، يحتفظ التطبيق بالقيمة الافتراضية (`Defaults`).
- **المفتاح الفارغ الصريح (Explicit Empty String `""`)**: إذا قام المشغل بتعيين `DB_PASSWORD=""`، يتم اعتماد السلسلة الفارغة صراحة (مفيد في قواعد البيانات المحلية بدون كلمة مرور).
