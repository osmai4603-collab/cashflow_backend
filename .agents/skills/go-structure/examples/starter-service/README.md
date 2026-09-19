# مشروع نموذجي لمهارة go-structure (Starter Service Example)

هذا المشروع يُعد قالباً عملياً حياً وتطبيقاً مباشراً للمعايير الهندسية الرسمية المعتمدة في مهارة `go-structure`.

## 📁 الهيكل الشجري للمشروع (Project Tree)

```text
starter-service/
├── cmd/
│   └── server/
│       └── main.go                    # جذر التكوين (Composition Root) ودورة حياة الخدمة
├── internal/
│   ├── core/                          # النواة المعمارية النقية الخالية من التبعيات
│   │   ├── domain/
│   │   │   └── wallet.go              # الكيان النقي وأخطاء النطاق الدلالية
│   │   └── ports/
│   │       └── repositories.go        # العقود والواجهات التجريدية (Ports)
│   ├── usecases/
│   │   ├── transfer.go                # حالة الاستخدام (تنسيق الأعمال والمعاملات الذرية)
│   │   └── transfer_test.go           # اختبار أحادي نقي بدون أي قواعد بيانات خارجية
│   ├── adapters/
│   │   ├── primary/
│   │   │   └── http/
│   │   │       ├── handler.go         # محول طلبات REST وتطبيع الأخطاء
│   │   │       └── server.go          # خادم الويب ومسارات /livez و /readyz
│   │   └── secondary/
│   │       └── memory/
│   │           └── wallet_repo.go     # محول الذاكرة المنفذ لعقد التخزين والـ UoW
│   └── platform/
│       └── config/
│           └── config.go              # تحميل الإعدادات والفحص الصارم المبكر (Fail-Fast)
├── go.mod
└── README.md
```

## 🚀 كيفية تشغيل المشروع وتجربته

### 1. تشغيل الخادم
```bash
go run ./cmd/server
```

### 2. فحص الجاهزية والحياة (Health Probes)
```bash
curl -i http://localhost:8080/livez
curl -i http://localhost:8080/readyz
```

### 3. تنفيذ عملية تحويل أموال (REST API Transfer)
```bash
curl -i -X POST http://localhost:8080/api/v1/wallets/transfer \
  -H "Content-Type: application/json" \
  -d '{"from_wallet_id":"wallet-1","to_wallet_id":"wallet-2","amount":1500}'
```

### 4. تشغيل الاختبارات الأحادية
```bash
go test -v ./...
```
