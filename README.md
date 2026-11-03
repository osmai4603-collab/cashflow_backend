# Cashflow Backend (Go Multi-Module Workspace)

مشروع backend لنظام التدفقات النقدية (Cashflow System) مبني بنمط **Go Multi-Module Workspace** عبر ملف `go.work`.

كل **Feature** عبارة عن Go Module مستقل تماماً يمتلك ملف `go.mod` خاص به، وتجمعهم معمارية Modular Monolith مرنة تتيح العمل عليها أو تشغيلها منفصلة مستقبلاً (Microservices).

---

## 📁 هيكل المشروع (Project Structure)

```text
cashflow_backend/
├── go.work                          # ملف مساحة العمل الذي يربط جميع الموديولات
├── Makefile                         # أوامر التشغيل، البناء، الفحص وإضافة features
├── .gitignore
│
├── shared/                          # موديول الأدوات المشتركة (cashflow.local/shared)
│   ├── go.mod
│   ├── config/                      # إعدادات النظام وقراءة البيئة
│   └── response/                    # استجابات الـ API الموحدة (JSON/Errors)
│
├── features/                        # الموديولات الوظيفية (3 Features أساسية)
│   ├── auth/                        # موديول المصادقة والمستخدمين (cashflow.local/auth)
│   │   ├── go.mod
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── handler.go
│   │   └── service_test.go
│   │
│   ├── accounts/                    # موديول الحسابات المالية (cashflow.local/accounts)
│   │   ├── go.mod
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── handler.go
│   │   └── service_test.go
│   │
│   └── transactions/                # موديول المعاملات المالية (cashflow.local/transactions)
│       ├── go.mod
│       ├── model.go
│       ├── service.go
│       ├── handler.go
│       └── service_test.go
│
└── apps/
    └── api/                         # المشغّل الرئيسي (cashflow.local/api)
        ├── go.mod
        └── main.go
```

---

## 🚀 أوامر التشغيل السريعة (Commands)

### 1. تشغيل الخادم محلياً

```bash
make run
# أو مباشرة:
go run ./cmd/server
```

### 2. بناء المشروع (Build)

```bash
make build
# سيتم إنشاء الملف التنفيذي داخل bin/api
```

### 3. تشغيل جميع اختبارات الـ Workspace

```bash
make test
```

### تهيئة دور PostgreSQL

يُنشئ الملف دور `cashflow` وقاعدة البيانات `train_db` عند غيابهما، ثم يمنح الدور
الصلاحيات المطلوبة. نفّذه كمسؤول PostgreSQL متصلاً بقاعدة الصيانة `postgres`:

```bash
sudo -u postgres psql -d postgres -f scripts/bootstrap_cashflow_role.sql
sudo -u postgres psql -d train_db
```

داخل `psql`، عيّن كلمة مرور للدور دون كتابتها في ملف:

```text
\password cashflow
\q
```

مرّر كلمة المرور للتطبيق عبر متغير البيئة `DB_PASSWORD`، ثم شغّله:

```bash
read -rsp "PostgreSQL password: " DB_PASSWORD
echo
export DB_PASSWORD
make run
unset DB_PASSWORD
```

### 4. إضافة Feature جديدة بسهولة

يمكنك إنشاء موديول feature جديد وإضافته إلى مساحة العمل تلقائياً بأمر واحد:

```bash
make add-feature name=budgets
```

يقوم هذا الأمر بـ:

1. إنشاء مجلد `features/budgets`.
2. إنشاء `features/budgets/go.mod` بالمسار `cashflow.local/budgets`.
3. تسجيل الموديول في `go.work` تلقائياً عبر `go work use ./features/budgets`.

---

## 🌐 نقاط النهاية المتوفرة (API Endpoints)

| المسار | الطريقة | الوصف |
| -------- | --------- | ------- |
| `/health` | `GET` | فحص صحة الخادم والموديولات النشطة |
| `/api/v1/auth/register` | `POST` | تسجيل مستخدم جديد |
| `/api/v1/auth/login` | `POST` | تسجيل الدخول |
| `/api/v1/accounts` | `POST` | إنشاء حساب مالي (بنك/كاش/محفظة) |
| `/api/v1/accounts` | `GET` | جلب قائمة الحسابات |
| `/api/v1/accounts/{id}` | `GET` | تفاصيل حساب محدد |
| `/api/v1/transactions` | `POST` | تسجيل معاملة (دخل أو مصروف) وتحديث الرصيد |
| `/api/v1/transactions` | `GET` | استعراض المعاملات |
