# دورة حياة عمال الخلفية والإشراف (Worker Lifecycle & Supervision)

في الخدمات المبنية بلغة Go، تُعد الـ Goroutines أداة فائقة الخفة والقوة لتنفيذ المهام غير المتزامنة (مثل معالجة رسائل البريد، أو تحديث العقود، أو تنظيف السجلات المؤقتة). ولكن سوء إدارة هذه الخيوط يؤدي إلى انهيار الخادم بالكامل في حال حدوث Panic غير معالج.

---

## 1. نمط الإشراف والتعافي التلقائي (Supervision & Panic Recovery)

قاعدة ذهبية في Go:
> **أي Panic يحدث داخل Goroutine مستقلة ولا يتم التقاطه بواسطة `recover()` سيؤدي إلى إنهاء عملية البرنامج بالكامل (`SIGABRT` / Crash)، حتى لو كان السيرفر الرئيسي يعمل بشكل سليم.**

### هيكل حلقة العامل المحمية

```go
func StartSupervisedWorker(ctx context.Context, name string, task func(context.Context) error) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                stack := debug.Stack()
                slog.Error("worker recovered from panic",
                    "worker", name,
                    "panic", r,
                    "stack", string(stack),
                )
                // إعادة تشغيل العامل بعد مهلة قصيرة إذا كان السياق لا يزال نشطاً
                if ctx.Err() == nil {
                    time.Sleep(1 * time.Second)
                    StartSupervisedWorker(ctx, name, task)
                }
            }
        }()

        for {
            select {
            case <-ctx.Done():
                slog.Info("worker stopped cleanly", "worker", name)
                return
            default:
                if err := task(ctx); err != nil {
                    slog.Warn("worker task execution error", "worker", name, "error", err)
                }
            }
        }
    }()
}
```

---

## 2. التكامل مع التوقف الآمن (Graceful Shutdown Integration)

عمال الخلفية جزء لا يتجزأ من مراحل دورة حياة الخادم الثمانية:

1. **وقت البدء (Startup - Phase 4)**: تبدأ حلقات العمال فقط بعد فتح اتصالات قاعدة البيانات وتأكيد جاهزية الخادم.
2. **وقت الإيقاف (Teardown - Phase 7 & 8)**:
   - عند استقبال إشارة الإيقاف، يتم إلغاء السياق المشترك (`cancel()`).
   - ينتظر المشرف الرئيسي انتهاء العمال عبر `sync.WaitGroup.Wait()`.
   - لا يتم إغلاق مجمّع اتصالات قاعدة البيانات (`dbPool.Close()`) إلا بعد خروج كافة العمال بالكامل لضمان عدم فشل المعاملات الجارية.
