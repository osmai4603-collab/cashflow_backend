# المهام المجدولة وإدارة العدادات الزمنية (Scheduled Tasks & Tickers)

العديد من المهام تتطلب تكراراً دورياً (كل دقيقة، كل ساعة، أو كل يوم)، مثل مزامنة العقود المنتهية أو إرسال التقارير المحاسبية.

---

## 1. الفخ القاتل: استخدام `time.Tick` أو `time.Sleep`

### أ) خطأ `time.Tick` وتسريب الذاكرة

دالة `time.Tick(duration)` القياسية في Go تُنشئ قناة لا يمكن استدعاء `Stop()` عليها، ويبقى المؤقت في الذاكرة للأبد:
> [!CAUTION]
> توثيق Go الرسمي يحذر صراحة:
> "Tick is convenient but leaks resources if the loop is intended to terminate. Use time.NewTicker instead."

### ب) خطأ `time.Sleep` وتجاهل إشارات الإيقاف

إذا كان العامل ينفذ `time.Sleep(10 * time.Minute)`، واستقبل السيرفر إشارة `SIGTERM` للإيقاف أو إعادة النشر، فإن العامل سيعلق لمدة 10 دقائق كاملة ولن يستجيب لأمر الإيقاف!

---

## 2. النمط المعياري للعدادات الدورية (The Safe Ticker Loop)

```go
func RunRecurringTask(ctx context.Context, interval time.Duration, task func(context.Context)) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop() // حتمي لمنع تسريب الموارد

    // تنفيذ المهمة فوراً عند البدء إذا رغبت، أو الانتظار حتى أول دقة
    for {
        select {
        case <-ctx.Done():
            // الاستجابة الفورية لإشارة الإيقاف
            return
        case <-ticker.C:
            task(ctx)
        }
    }
}
```

---

## 3. التعامل مع المهام طويلة الأمد (Overlapping Executions)

إذا كانت المهمة المجدولة تستغرق وقتاً أطول من الفترة الزمنية المحددة (مثلاً المهمة استغرقت 70 ثانية بينما الـ Ticker يعمل كل 60 ثانية):

- **النمط المتسلسل (Sequential)**: ينتظر حتى تكتمل الدورة الحالية قبل بدء الدورة التالية.
- **النمط المحمي بقفل (Mutex / Distributed Lock)**: استخدام قفل لتفادي تشغيل نسختين من المهمة في نفس الوقت.
