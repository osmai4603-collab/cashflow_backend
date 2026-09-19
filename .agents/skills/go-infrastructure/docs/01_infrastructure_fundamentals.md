# أساسيات طبقة البنية التحتية والمحولات الثانوية (Infrastructure Layer Fundamentals)

تُمثل **طبقة البنية التحتية (`internal/infrastructure`)** في المعمارية النظيفة (Clean Architecture) والمعمارية السداسية (Hexagonal Ports & Adapters) الحافة الخارجية للتطبيق باتجاه الخدمات الشبكية والأنظمة البعيدة ومستودعات التخزين.

مهمتها الأساسية هي تجسيد واجهات منافذ النطاق الصادرة (Secondary / Driven Outbound Ports) عبر محولات ملموسة (Concrete Adapters) تخفي تعقيدات البروتوكولات ومكتبات المزودين عن النطاق.

---

## 1. ركائز البنية التحتية الأربع (The Four Infrastructure Pillars)

في أي نظام مؤسسي متكامل بلغة Go، تنقسم مسؤوليات البنية التحتية إلى 4 ركائز رئيسية:

1. **محركات التخزين وقواعد البيانات (Persistence & Caching)**:
   - إدارة مجمعات الاتصال ومآخذ قواعد البيانات (`database/sql` / `pgxpool`).
   - محولات التخزين المؤقت الموزع (Redis, Memcached).
   - محولات التخزين السحابي للملفات والكائنات (S3, GCS, MinIO, Local Disk).
2. **وسطاء الرسائل وتدفق الأحداث (Message Brokers & Event Streaming)**:
   - محولات الاتصال مع أنظمة الطوابير والبث الموزع (RabbitMQ, Apache Kafka, NATS JetStream, Redis Streams).
   - إدارة مجموعات المستهلكين (Consumer Groups)، والتأكيدات (Acks)، وطوابير الرسائل الميتة (DLQ).
3. **عملاء الشبكات والخدمات الخارجية (Resilient Outbound Clients & APIs)**:
   - عملاء HTTP و gRPC المحصنون ضد تقلبات الشبكة.
   - عزل مكتبات الطرف الثالث (Third-Party SDKs) وتطبيق قواطع الدوائر (Circuit Breakers).
   - سجل المزودين المتعددين (Provider Registry Pattern).
4. **بوابات الأمان وإدارة الأسرار (Identity & Secrets Management)**:
   - محولات جلب المفاتيح والأسرار من خوادم إدارة الأسرار (HashiCorp Vault, AWS Secrets Manager, KMS).
   - محولات التحقق من الهوية الخارجية (OAuth2 / OIDC Providers, Keycloak).

---

## 2. نمط المحول الصادر وقاعدة Go: "Accept interfaces, return structs"

تُطبق المعمارية النظيفة مبدأ عكس التبعية (Dependency Inversion Principle) بالتوافق مع توصيات Go الرسمية (`Go Code Review Comments`):

- **طبقة النطاق أو حالات الاستخدام**: تعلن الواجهة المصغرة (Port Interface) التي تحتاجها فقط دون أي تفاصيل تقنية:

  ```go
  type EntityRepository interface {
      GetByID(ctx context.Context, id string) (*domain.Entity, error)
      Save(ctx context.Context, entity *domain.Entity) error
  }
  ```

- **طبقة البنية التحتية**: توفر المحول الملموس (`PostgresRepository`) الذي يُلبي هذه الواجهة، وتعيد دالة البناء مؤشراً لتركيبة ملموسة:

  ```go
  func NewPostgresRepository(db *sql.DB) *PostgresRepository
  ```

---

## 3. العزل الصارم لمكتبات المزودين (Zero SDK Leakage)

تُعد طبقة البنية التحتية **الطبقة الوحيدة المصرح لها باستيراد مكتبات ومحركات الطرف الثالث** في المشروع:

- يُحظر تماماً استيراد أي SDK خارجي (مثل مكتبات الدفع، مكتبات البريد، أو عملاء السحابة) داخل `domain` أو `usecase` أو `platform`.
- في حال تبديل المزود الخارجي أو ترقية مكتبته البرمجية، تظل طبقات النطاق وحالات الاستخدام دون أي تعديل، وينحصر التغيير فقط في كتابة أو تحديث المحول في `internal/infrastructure`.

---

## 4. التعامل مع الطبيعة غير الموثوقة للشبكات

تعمل محولات البنية التحتية في بيئة غير موثوقة ومحكومة بقيود العالم الحقيقي (شبكات متقطعة، أزمنة استجابة متباينة، وخوادم قد تسقط في أي لحظة). لذا، يجب أن يتضمن كل محول شبكي خط دفاع ثلاثي:

1. **مهل زمنية صارمة مقيدة بالسياق (`context.WithTimeout`)**.
2. **قواطع دوائر (Circuit Breakers)** لإحباط الانهيارات المتتالية وتوفير Fail-Fast فوري.
3. **إعادة محاولة ذكية بتراجع أسي وتشتت عشوائي (Exponential Backoff with Full Jitter)**.
