# حدود المعاملات وقفل الصفوف (Transaction Boundaries & Row-Level Locking)

في الأنظمة المالية وتطبيقات المحاسبة، لا يمكن التسامح مع حالات السباق (Race Conditions) أثناء تعديل الأرصدة أو حجز المخزون.

---

## 1. قفل الصفوف المالي (`SELECT ... FOR UPDATE`)

الخطأ الشائع:

```go
// ❌ عرضة لسباق التعديل وفقدان البيانات
var balance float64
db.QueryRow("SELECT balance FROM accounts WHERE id = $1", id).Scan(&balance)
balance -= amount
db.Exec("UPDATE accounts SET balance = $1 WHERE id = $2", balance, id)
```

إذا حدث طلبان متزامنان في نفس اللحظة، سيقرأ كلاهما نفس الرصيد المبدئي، وسيؤدي الحفظ إلى ضياع إحدى العمليتين!

### الحل المعياري الصارم

فتح معاملة وقفل الصف المطلوب حصرياً حتى اكتمال المعاملة:

```sql
SELECT balance FROM accounts WHERE id = $1 FOR UPDATE;
```

أي طلب متزامن آخر سيضطر إلى الانتظار حتى تنتهي المعاملة الحالية بـ `COMMIT` أو `ROLLBACK`.

---

## 2. تجنب مشاكل القفل المتبادل (Deadlock Prevention)

عندما تحتاج معاملة مالية إلى قفل حسابين (مثلاً: تحويل من حساب أ إلى حساب ب):

- إذا قفلت المعاملة الأولى `A` ثم طلبت `B`، والمعاملة الثانية قفلت `B` ثم طلبت `A`، سيحدث **Deadlock** فوري وتلغي قاعدة البيانات إحدى المعاملتين.
- **الحل الهندسي**: **الترتيب الصارم للأقفال (Consistent Lock Ordering)**:
  ترتيب المعرفات تصاعدياً دائماً قبل القفل:
  $$\text{Lock}(\min(\text{ID}_A, \text{ID}_B)) \longrightarrow \text{Lock}(\max(\text{ID}_A, \text{ID}_B))$$
  هذا يضمن أن كافة المعاملات تطلب الموارد بنفس الترتيب، مما يستحيل معه حدوث Deadlock.
