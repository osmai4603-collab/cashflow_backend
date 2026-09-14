# استراتيجيات الاختبار والمحاكاة (Testing & Mocking Strategies)

إحدى أعظم مزايا المعمارية النظيفة هي تمكين المطور من كتابة **اختبارات وحدة (Unit Tests) فورية وخفيفة** تغطي أدق تفاصيل منطق الأعمال دون الحاجة إلى تشغيل خادم محلي أو قاعدة بيانات خارجية.

---

## 1. هرم الاختبارات في المعمارية النظيفة

```text
       ▲
      / \     اختبارات التكامل (Integration Tests)
     /   \    تختبر محولات الـ Adapters مع Postgres الحقيقي
    /─────\
   /       \   اختبارات حالات الاستخدام (Use Case Tests)
  /         \  تختبر سيناريوهات الأعمال بنسبة 100% باستخدام Mocks
 /───────────\
/             \ اختبارات النطاق (Domain Tests)
═══════════════ تختبر قواعد الأعمال البحتة (Invariants) بدون أي محاكاة
```

---

## 2. اختبار كائنات النطاق (Domain Testing)

كائنات النطاق لا تعتمد على أي شيء خارجي، لذا فإن اختبارها مباشر وسريع جداً:

```go
func TestInvoice_MarkAsPaid(t *testing.T) {
    inv := domain.NewInvoice("inv-1", 500)
    
    err := inv.MarkAsPaid()
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    
    // محاولة دفعها مرة أخرى يجب أن تفشل وفق قواعد النطاق
    err = inv.MarkAsPaid()
    if !errors.Is(err, domain.ErrInvoiceAlreadyPaid) {
        t.Fatalf("expected ErrInvoiceAlreadyPaid, got %v", err)
    }
}
```

---

## 3. اختبار الـ Use Cases باستخدام محاكيات الدوال (Function Mocks)

في Go، لا نحتاج بالضرورة إلى مكتبات محاكاة معقدة؛ تكفي هياكل بسيطة تحتوي على دوال مجهولة (Function Fields):

```go
type mockInvoiceRepo struct {
    saveFunc    func(ctx context.Context, inv *domain.Invoice) error
    getByIDFunc func(ctx context.Context, id string) (*domain.Invoice, error)
}

func (m *mockInvoiceRepo) SaveInvoice(ctx context.Context, inv *domain.Invoice) error {
    if m.saveFunc != nil {
        return m.saveFunc(ctx, inv)
    }
    return nil
}

func (m *mockInvoiceRepo) GetByID(ctx context.Context, id string) (*domain.Invoice, error) {
    if m.getByIDFunc != nil {
        return m.getByIDFunc(ctx, id)
    }
    return nil, nil
}
```

### الاختبار الموجه بالجداول (Table-Driven Test)

```go
func TestCreateInvoiceUseCase(t *testing.T) {
    tests := []struct {
        name        string
        inputAmount float64
        mockSaveErr error
        wantErr     bool
    }{
        {
            name:        "نجاح إنشاء فاتورة صالحة",
            inputAmount: 100.0,
            mockSaveErr: nil,
            wantErr:     false,
        },
        {
            name:        "فشل عند إدخال مبلغ غير صالح",
            inputAmount: -50.0,
            mockSaveErr: nil,
            wantErr:     true,
        },
        {
            name:        "فشل عند تعطل قاعدة البيانات أثناء الحفظ",
            inputAmount: 200.0,
            mockSaveErr: errors.New("connection failed"),
            wantErr:     true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockInvoiceRepo{
                saveFunc: func(ctx context.Context, inv *domain.Invoice) error {
                    return tt.mockSaveErr
                },
            }
            uc := invoice.NewCreateInvoiceUseCase(repo)
            
            err := uc.Execute(context.Background(), invoice.CreateCmd{Amount: tt.inputAmount})
            if (err != nil) != tt.wantErr {
                t.Fatalf("Execute() error = %v, wantErr = %v", err, tt.wantErr)
            }
        })
    }
}
```

هذا الاختبار يستغرق أجزاءً من الألف من الثانية، ولا يعتمد على بيئة تشغيل، ولا يمكن أن يصبح غير مستقر (Flaky).
