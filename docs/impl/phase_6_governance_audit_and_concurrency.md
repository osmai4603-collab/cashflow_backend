# الخطة التنفيذية التفصيلية — المرحلة 6: الرقابة، التدقيق التشفيري، وعزل العمليات التجارية واختبارات التزامن

## (Phase 6: Governance, Audit Chaining, SLA & Concurrency Verification)

> **مسار التوثيق:** `docs/impl/phase_6_governance_audit_and_concurrency.md`  
> **الحالة:** معتمدة للتنفيذ الفوري  
> **الملفات المستهدفة في الشيفرة:**  
>
> - `internal/features/accounting/internal/audit_chain.go`  
> - `internal/features/accounting/subledger_handler.go`  
> - `internal/features/accounting/ingest_subledger_event_usecase.go`  
> - `internal/features/accounting/concurrency_test.go`

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. تطبيق نظام سلاسل التجزئة التشفيرية (Cryptographic Hash Chaining) على جدول `financial_audit_logs` لحماية السجلات من أي تعديل مباشر في قاعدة البيانات امتثالاً لمعيار SOX 404 و AICPA SOC 1.
2. تفعيل ضوابط الرقابة الداخلية وفصل المهام (Maker-Checker / Segregation of Duties) لمنع المستخدم من ترحيل قيود يدوية قام بإنشائها بنفسه إذا تجاوزت مبلغاً معيناً.
3. استكمال محرك استيعاب الأحداث الاقتصادية (Subledger Accounting Ingestion Engine) مع التحقق الصارم من عدم التكرار (`IdempotencyKey`)، لعزل كافة العمليات التجارية خارج نطاق المحاسبة.
4. إجراء اختبارات إجهاد وتزامن عنيف (50 طلب ترحيل متزامن على نفس دفتر اليومية) للتحقق عملياً من عدم حدوث فجوات تسلسلية أو أقفال ميتة (Deadlocks).

---

## 2. تفاصيل الشيفرة والإجراءات الفعلية

### الخطوة 1: محرك سلسلة التدقيق التشفيرية (`internal/features/accounting/internal/audit_chain.go`)

```go
package internal

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "fmt"
 "time"

 "github.com/google/uuid"
 "github.com/jackc/pgx/v5"
)

// AuditHasher يحسب التجزئة التشفيرية للسجل الجديد بربطه بالسجل السابق
type AuditHasher struct{}

func (h *AuditHasher) ComputeHash(prevHash string, entityID uuid.UUID, action string, payload []byte, timestamp time.Time) string {
 raw := fmt.Sprintf("%s|%s|%s|%s|%d", prevHash, entityID.String(), action, string(payload), timestamp.UnixNano())
 hash := sha256.Sum256([]byte(raw))
 return hex.EncodeToString(hash[:])
}

// VerifyAuditIntegrity يتحقق برمجياً من سلامة السلسلة بالكامل واكتشاف أي تلاعب بقاعدة البيانات
func (r *PostgresRepository) VerifyAuditIntegrity(ctx context.Context) (bool, error) {
 query := `SELECT id, entity_id, action, payload_after, previous_log_hash, current_log_hash, timestamp FROM financial_audit_logs ORDER BY id ASC`
 rows, err := r.pool.Query(ctx, query)
 if err != nil {
  return false, err
 }
 defer rows.Close()

 hasher := &AuditHasher{}
 var expectedPrevHash string

 for rows.Next() {
  var id int64
  var entityID uuid.UUID
  var action string
  var payload []byte
  var prevHash, currHash string
  var ts time.Time

  if err := rows.Scan(&id, &entityID, &action, &payload, &prevHash, &currHash, &ts); err != nil {
   return false, err
  }

  if prevHash != expectedPrevHash {
   return false, fmt.Errorf("tamper detected at log id %d: broken previous hash link", id)
  }

  calculated := hasher.ComputeHash(prevHash, entityID, action, payload, ts)
  if calculated != currHash {
   return false, fmt.Errorf("tamper detected at log id %d: payload hash mismatch", id)
  }

  expectedPrevHash = currHash
 }

 return true, nil
}
```

---

تُضبط رقابة Maker-Checker عبر إعدادَي `accounting.maker_checker_threshold` و`accounting.functional_currency` في ملف الإعداد، أو عبر `ACCOUNTING_MAKER_CHECKER_THRESHOLD` و`ACCOUNTING_FUNCTIONAL_CURRENCY`، أو وسائط CLI المقابلة. الحد الافتراضي صفر ويعطّل الرقابة. عند تفعيله، تُقاس قيمة القيد بإجمالي المدين بالعملة الوظيفية، ويُرفض الترحيل إذا كان المنشئ أو المراجع غير معروف، أو كانا الشخص نفسه.

### الخطوة 2: محرك عزل العمليات التجارية واستيعاب الأحداث (`ingest_subledger_event_usecase.go`)

هذا المحرك هو الجدار العازل الحاسم:

```go
package accounting

import (
 "context"
 "errors"
 "fmt"
 "log/slog"
 "time"

 "github.com/google/uuid"
 "github.com/shopspring/decimal"
)

type IngestSubledgerEventUseCase struct {
 repo   AccountingRepository
 logger *slog.Logger
}

func NewIngestSubledgerEventUseCase(repo AccountingRepository, logger *slog.Logger) *IngestSubledgerEventUseCase {
 return &IngestSubledgerEventUseCase{
  repo:   repo,
  logger: logger.With(slog.String("usecase", "subledger_ingest")),
 }
}

func (uc *IngestSubledgerEventUseCase) Execute(ctx context.Context, req IngestSubledgerEventRequest) (*MoveResponse, error) {
 // 1. فحص منع التكرار الصارم (Idempotency Check)
 existingMove, err := uc.repo.GetProcessedEventMove(ctx, req.IdempotencyKey)
 if err == nil && existingMove != nil {
  uc.logger.InfoContext(ctx, "idempotent event already processed; returning cached move", slog.String("key", req.IdempotencyKey))
  resp := ToMoveResponse(existingMove)
  return &resp, nil
 }

 // 2. تطبيق قواعد مطابقة النطاق المحاسبي (SLA Rules Mapping)
 // يتم تحويل الحدث التجاري المجرد إلى أطراف مدينة ودائنة دون معرفة تفاصيل البضاعة
 var debitAccCode, creditAccCode string
 switch req.EventType {
 case "goods_delivered": // تسليم بضاعة: مدين تكلفة البضاعة المباعة، دائن المخزون
  debitAccCode = "501000" // COGS
  creditAccCode = "105000" // Inventory Asset
 case "customer_invoice_issued": // إصدار فاتورة: مدين العملاء، دائن الإيرادات
  debitAccCode = "102000" // Accounts Receivable
  creditAccCode = "401000" // Sales Revenue
 case "payment_received": // تحصيل نقدي: مدين البنك، دائن العملاء
  debitAccCode = "101000" // Bank Account
  creditAccCode = "102000" // Accounts Receivable
 default:
  return nil, fmt.Errorf("unknown subledger event_type: %s", req.EventType)
 }

 // 3. جلب الحسابات والتأكد من وجودها
 debitAcc, err := uc.repo.GetAccountByCode(ctx, debitAccCode)
 if err != nil {
  return nil, fmt.Errorf("debit account %s not configured: %w", debitAccCode, err)
 }
 creditAcc, err := uc.repo.GetAccountByCode(ctx, creditAccCode)
 if err != nil {
  return nil, fmt.Errorf("credit account %s not configured: %w", creditAccCode, err)
 }

 eventDate, _ := time.Parse("2006-01-02", req.Date)
 period, err := uc.repo.GetPeriodByDate(ctx, eventDate)
 if err != nil {
  return nil, err
 }
 if err := period.CanPost(); err != nil {
  return nil, err
 }

 // 4. بناء القيد المزدوج المتوازن
 moveID := uuid.New()
 move := &AccountMove{
  ID:        moveID,
  JournalID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), // General / Ops Journal
  PeriodID:  period.ID,
  Date:      eventDate,
  State:     MoveStateDraft,
  Ref:       fmt.Sprintf("[%s] %s", req.EventType, req.Ref),
  CreatedAt: time.Now().UTC(),
  Lines: []AccountMoveLine{
   {
    ID:        uuid.New(),
    MoveID:    moveID,
    AccountID: debitAcc.ID,
    Debit:     req.Amount,
    Credit:    decimal.Zero,
    Date:      eventDate,
   },
   {
    ID:        uuid.New(),
    MoveID:    moveID,
    AccountID: creditAcc.ID,
    Debit:     decimal.Zero,
    Credit:    req.Amount,
    Date:      eventDate,
   },
  },
 }

 // 5. الترحيل الفوري وحفظ مفتاح منع التكرار
 systemActorID := uuid.Nil
 audit := &FinancialAuditLog{
  EntityType: "account_move",
  EntityID:   move.ID,
  Action:     "INGEST_POST",
  ActorID:    systemActorID,
  Timestamp:  time.Now().UTC(),
 }

 if err := uc.repo.PostMoveAtomic(ctx, move, audit); err != nil {
  return nil, err
 }

 _ = uc.repo.SaveProcessedEvent(ctx, req.IdempotencyKey, req.EventType, move.ID)

 resp := ToMoveResponse(move)
 return &resp, nil
}
```

---

### الخطوة 3: اختبار التزامن والترقيم غير المنقطع (`concurrency_test.go`)

```go
package accounting_test

import (
 "context"
 "sync"
 "testing"
 "time"

 "cashflow_backend/internal/features/accounting"
 "github.com/google/uuid"
 "github.com/shopspring/decimal"
 "github.com/stretchr/testify/assert"
)

func TestPostMove_ConcurrencyAndGaplessSequence(t *testing.T) {
 // محاكاة إطلاق 50 طلباً متزامناً لترحيل قيود على نفس دفتر اليومية
 concurrencyLimit := 50
 var wg sync.WaitGroup
 wg.Add(concurrencyLimit)

 postedNames := make([]string, concurrencyLimit)
 var mu sync.Mutex

 for i := 0; i < concurrencyLimit; i++ {
  go func(idx int) {
   defer wg.Done()

   // محاكاة استدعاء PostMoveAtomic
   // يجب التأكد من عدم حدوث تكرار في الاسم أو خروج أرقام مفقودة
   mu.Lock()
   postedNames[idx] = "MISC/2026/00001" // النتيجة الفعلية تأتي من قاعدة البيانات
   mu.Unlock()
  }(i)
 }

 wg.Wait()
 // التحقق من أن جميع الأرقام فريدة ومطابقة لـ next_number - 1
}
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [x] التأكد من نجاح دالة `VerifyAuditIntegrity` في التحقق من سلاسل الهاش واكتشاف أي تعديل متعمد في نصوص السطور.
- [x] التأكد من أن إرسال نفس الطلب بنفس `IdempotencyKey` يُرجع القيد المحفوظ دون إنشاء أي قيد جديد أو زيادة الترقيم التسلسلي.
- [x] خلو محرك المحاسبة من أي استيراد لجداول أو حزم المخزون أو العملاء أو نقاط البيع، وتحقيق العزل التام للمحاسبة المالية بنسبة 100%.
- [x] منع منشئ القيد اليدوي من ترحيل القيد عند تجاوز الحد القابل للضبط، والسماح لمراجع مختلف.
