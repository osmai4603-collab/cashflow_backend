# جذر التكوين وربط التبعيات الصريح (Composition Root & Explicit Wiring)

يمثل **جذر التكوين (Composition Root)** الموقع المركزي والوحيد في التطبيق الذي يتم فيه بناء شجرة التبعيات بالكامل وربط الطبقات ببعضها البعض. في تطبيقات Go القياسية، يوضع هذا المنطق في `internal/platform/app` وتستدعيه نقطة الدخول `cmd/server/main.go`.

---

## 1. لماذا نفضل الربط الصريح على أطر حقن التبعيات؟

تتجنب لغة Go استخدام أطر عمل حقن التبعيات المبنية على الانعكاس (Reflection-based DI) لتفادي انهيارات وقت التشغيل وصعوبة تتبع مسار البرنامج.

### مزايا الربط الصريح في Go (Compile-Time Safety)

- **أمان وقت الترجمة**: التحقق من تمرير كافة التبعيات عبر دوال البناء (`Constructors`) في وقت التجميع، مانعاً أي مؤشرات فارغة (Nil Pointer Dereferences).
- **التوافق مع قاعدة Go**: تطبيق قاعدة **"Accept interfaces, return structs"** الصادرة في `Go Code Review Comments`، حيث يعيد كل محول تركيبته الملموسة، ويتم حقنها في واجهات المستهلكين.
- **شفافية شجرة النظام**: يستطيع أي مهندس برمجيات قراءة ملف `app.go` وفهم ترابط ومسار تشغيل النظام بأكمله خلال دقائق معدودة.

---

## 2. تسلسل الإقلاع الطوبولوجي (Topological Bootstrapping Sequence)

```text
المرحلة 1: الإعدادات والمراقبة (Configuration & Observability)
  ├── LoadConfig()
  └── InitLogger() (slog)

المرحلة 2: أدوات المنصة الأساسية (Platform Primitives)
  ├── InitClock() (RealClock)
  ├── InitIDGenerator() (UUIDv7 / ULID)
  ├── InitEventBus()
  └── InitWorkerPool()

المرحلة 3: اتصالات البنية التحتية وقواعد البيانات (Infrastructure Connections)
  ├── InitDatabasePool() (database/sql / pgxpool with tuned limits)
  ├── InitCacheClient() (Redis)
  └── InitMessageBrokers()

المرحلة 4: محولات المنافذ الصادرة (Secondary / Driven Outbound Adapters)
  ├── InitRepositories(dbPool, cacheClient)
  └── InitExternalGateways(httpClient, circuitBreakers)

المرحلة 5: حالات استخدام التطبيق (Application Use Cases)
  └── InitUseCases(repositories, gateways, eventBus, clock)

المرحلة 6: محولات الدخول وطبقة العرض (Primary / Driving Adapters & Routing)
  ├── InitHandlers(useCases)
  └── InitRouter(handlers, middlewares)

المرحلة 7: تشغيل الخادم والعمال الخلفيين (Execution & Supervision)
  ├── StartWorkers(ctx, workerPool)
  └── RunServer(ctx, router) with signal.NotifyContext
```

---

## 3. دورة الإنهاء والتفكيك المعكوس الصارم (Strict Reverse Teardown)

وفق أفضل ممارسات Go وهندسة النظم الموزعة، يجب إغلاق الموارد **بترتيب عكسي صارم لتاريخ إنشائها**:

```text
إشارة إنهاء (SIGINT / SIGTERM عبر signal.NotifyContext)
  │
  ▼
1. إيقاف خادم HTTP (srv.Shutdown مع مهلة مستقلة) ──> منع استقبال أي طلبات جديدة وتصريف الطلبات الحالية.
  │
  ▼
2. إيقاف العمال الخلفيين (WorkerPool.Shutdown) ──> انتظار اكتمال المهام الجارية دون قبول مهام جديدة.
  │
  ▼
3. تفريغ ناقل الأحداث في الذاكرة (EventBus) ──> إنهاء معالجة الأحداث الداخلية العالقة.
  │
  ▼
4. إغلاق اتصالات قواعد البيانات ومجمعات التخزين (db.Close & redis.Close) ──> تُغلق في النهاية لمنع انهيار أي استعلام جارٍ.
```

يضمن هذا التسلسل الحتمي سلامة البيانات وعدم مقاطعة أي معاملة بنكية أو استعلام في منتصف تنفيذه.
