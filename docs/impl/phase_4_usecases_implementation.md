# الخطة التنفيذية التفصيلية — المرحلة 4: حالات الاستخدام المستقلة واختبارات الوحدة

## (Phase 4: Dedicated Use Cases Implementation & Testing)

> **مسار التوثيق:** `docs/impl/phase_4_usecases_implementation.md`  
> **الحالة:** منفذة
> **الملفات المستهدفة في الشيفرة:**  
>
> - `internal/features/accounting/usecases.go`  
> - `internal/features/accounting/create_account_usecase.go`  
> - `internal/features/accounting/create_fiscal_year_usecase.go`  
> - `internal/features/accounting/change_period_state_usecase.go`  
> - `internal/features/accounting/create_draft_move_usecase.go`  
> - `internal/features/accounting/post_move_usecase.go`  
> - `internal/features/accounting/reverse_move_usecase.go`  
> - `internal/features/accounting/revalue_fx_usecase.go`  
> - `internal/features/accounting/generate_trial_balance_usecase.go`  
> - `internal/features/accounting/ingest_subledger_event_usecase.go`  
> - `internal/features/accounting/*_usecase_test.go`

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. تطبيق مبدأ المسؤولية الواحدة (Single Responsibility Principle) عبر إفراد كل عملية مالية بملف مستقل (`<action>_usecase.go`).
2. تجميع حالات الاستخدام داخل البنية المركزية `usecases.go` ومصنع التهيئة `NewUseCases(...)`.
3. التحقق الصارم من قواعد النطاق في كل حالة استخدام: فحص الفترات المالية المفتوحة، التوازن الصفري، ومنع التكرار (Idempotency).
4. تغطية حالات الاستخدام باختبارات وحدة موثوقة تعتمد على محاكاة المستودع (`Mock Repository`).

---

## 2. تفاصيل الشيفرة والإجراءات الفعلية

### الخطوة 1: الحاوية المجمعة لحالات الاستخدام (`usecases.go`)

```go
package accounting

import "log/slog"

type UseCases struct {
 CreateAccount         *CreateAccountUseCase
 ListAccountsTree      *ListAccountsTreeUseCase
 CreateFiscalYear      *CreateFiscalYearUseCase
 ChangePeriodState     *ChangePeriodStateUseCase
 CreateDraftMove       *CreateDraftMoveUseCase
 PostMove              *PostMoveUseCase
 ReverseMove           *ReverseMoveUseCase
 RevalueFX             *RevalueFXUseCase
 GenerateTrialBalance  *GenerateTrialBalanceUseCase
 IngestSubledgerEvent  *IngestSubledgerEventUseCase
}

func NewUseCases(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *UseCases {
 log := logger.With(slog.String("feature", "accounting"))
 return &UseCases{
  CreateAccount:        NewCreateAccountUseCase(repo, log),
  ListAccountsTree:     NewListAccountsTreeUseCase(repo, log),
  CreateFiscalYear:     NewCreateFiscalYearUseCase(repo, log),
  ChangePeriodState:    NewChangePeriodStateUseCase(repo, log),
  CreateDraftMove:      NewCreateDraftMoveUseCase(repo, log),
  PostMove:             NewPostMoveUseCase(repo, publisher, log),
  ReverseMove:          NewReverseMoveUseCase(repo, publisher, log),
  RevalueFX:            NewRevalueFXUseCase(repo, log),
  GenerateTrialBalance: NewGenerateTrialBalanceUseCase(repo, log),
  IngestSubledgerEvent: NewIngestSubledgerEventUseCase(repo, log),
 }
}
```

---

### الخطوة 2: حالة استخدام ترحيل القيد الذري (`post_move_usecase.go`)

هذه هي الحالة المركزية والأخطر في النظام المالي:

```go
package accounting

import (
 "context"
 "errors"
 "fmt"
 "log/slog"
 "time"

 "github.com/google/uuid"
)

type PostMoveUseCase struct {
 repo      AccountingRepository
 publisher EventPublisher
 logger    *slog.Logger
}

func NewPostMoveUseCase(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *PostMoveUseCase {
 return &PostMoveUseCase{
  repo:      repo,
  publisher: publisher,
  logger:    logger.With(slog.String("usecase", "post_move")),
 }
}

func (uc *PostMoveUseCase) Execute(ctx context.Context, moveID, actorID uuid.UUID) (*MoveResponse, error) {
 // 1. جلب مسودة القيد
 move, err := uc.repo.GetMoveByID(ctx, moveID)
 if err != nil {
  return nil, fmt.Errorf("failed to fetch move: %w", err)
 }
 if move.State == MoveStatePosted {
  return nil, ErrMoveAlreadyPosted
 }

 // 2. التحقق من التوازن الرياضي الصارم
 if err := move.ValidateBalance(); err != nil {
  uc.logger.WarnContext(ctx, "unbalanced move rejected", slog.String("move_id", moveID.String()), slog.Any("error", err))
  return nil, err
 }

 // 3. التحقق من أن الفترة المالية مفتوحة للترحيل
 period, err := uc.repo.GetPeriodByDate(ctx, move.Date)
 if err != nil {
  return nil, fmt.Errorf("no fiscal period found for date %s: %w", move.Date.Format("2006-01-02"), err)
 }
 if err := period.CanPost(); err != nil {
  return nil, err
 }
 move.PeriodID = period.ID

 // 4. إعداد سجل التدقيق
 now := time.Now().UTC()
 move.PostedAt = &now
 move.PostedBy = &actorID

 audit := &FinancialAuditLog{
  EntityType: "account_move",
  EntityID:   move.ID,
  Action:     "POST",
  ActorID:    actorID,
  Timestamp:  now,
 }

 // 5. التنفيذ الذري في المستودع
 if err := uc.repo.PostMoveAtomic(ctx, move, audit); err != nil {
  uc.logger.ErrorContext(ctx, "atomic posting failed", slog.String("move_id", moveID.String()), slog.Any("error", err))
  return nil, fmt.Errorf("atomic post failure: %w", err)
 }

 uc.logger.InfoContext(ctx, "move posted successfully", slog.String("move_id", move.ID.String()), slog.String("legal_name", move.Name))

 // 6. بث حدث النطاق للمستهلكين الخارجيين
 if uc.publisher != nil {
  _ = uc.publisher.Publish(ctx, "accounting.move.posted", map[string]any{
   "move_id":    move.ID,
   "move_name":  move.Name,
   "date":       move.Date,
   "journal_id": move.JournalID,
  })
 }

 resp := ToMoveResponse(move)
 return &resp, nil
}
```

---

### الخطوة 3: حالة استخدام القيد العكسي (`reverse_move_usecase.go`)

```go
package accounting

import (
 "context"
 "fmt"
 "log/slog"
 "time"

 "github.com/google/uuid"
 "github.com/shopspring/decimal"
)

type ReverseMoveUseCase struct {
 repo      AccountingRepository
 publisher EventPublisher
 logger    *slog.Logger
}

func NewReverseMoveUseCase(repo AccountingRepository, publisher EventPublisher, logger *slog.Logger) *ReverseMoveUseCase {
 return &ReverseMoveUseCase{
  repo:      repo,
  publisher: publisher,
  logger:    logger.With(slog.String("usecase", "reverse_move")),
 }
}

func (uc *ReverseMoveUseCase) Execute(ctx context.Context, originalMoveID, actorID uuid.UUID, reversalDate time.Time, reason string) (*MoveResponse, error) {
 orig, err := uc.repo.GetMoveByID(ctx, originalMoveID)
 if err != nil {
  return nil, err
 }
 if orig.State != MoveStatePosted {
  return nil, ErrCannotReverseUnpostedMove
 }

 // التحقق من أن تاريخ العكس يقع في فترة مالية مفتوحة
 period, err := uc.repo.GetPeriodByDate(ctx, reversalDate)
 if err != nil {
  return nil, err
 }
 if err := period.CanPost(); err != nil {
  return nil, err
 }

 // إنشاء قيد عكسي مطابق (معكوس الأطراف)
 revMoveID := uuid.New()
 revLines := make([]AccountMoveLine, len(orig.Lines))
 for i, l := range orig.Lines {
  revLines[i] = AccountMoveLine{
   ID:             uuid.New(),
   MoveID:         revMoveID,
   AccountID:      l.AccountID,
   PartnerID:      l.PartnerID,
   Debit:          l.Credit, // المدين يصبح دائناً
   Credit:         l.Debit,  // الدائن يصبح مديناً
   AmountCurrency: l.AmountCurrency.Neg(),
   CurrencyID:     l.CurrencyID,
   CostCenterID:   l.CostCenterID,
   Date:           reversalDate,
   CreatedAt:      time.Now().UTC(),
  }
 }

 now := time.Now().UTC()
 reversalMove := &AccountMove{
  ID:             revMoveID,
  JournalID:      orig.JournalID,
  PeriodID:       period.ID,
  Date:           reversalDate,
  State:          MoveStateDraft,
  Ref:            fmt.Sprintf("Reversal of %s - %s", orig.Name, reason),
  ReversedMoveID: &originalMoveID,
  CreatedAt:      now,
  Lines:          revLines,
 }

 audit := &FinancialAuditLog{
  EntityType: "account_move",
  EntityID:   orig.ID,
  Action:     "REVERSE",
  ActorID:    actorID,
  Timestamp:  now,
 }

 if err := uc.repo.ReverseMoveAtomic(ctx, originalMoveID, reversalMove, audit); err != nil {
  return nil, fmt.Errorf("reversal failed: %w", err)
 }

 resp := ToMoveResponse(reversalMove)
 return &resp, nil
}
```

---

### الخطوة 4: حالة استخدام استخراج ميزان المراجعة (`generate_trial_balance_usecase.go`)

```go
package accounting

import (
 "context"
 "log/slog"
 "time"

 "github.com/shopspring/decimal"
)

type GenerateTrialBalanceUseCase struct {
 repo   AccountingRepository
 logger *slog.Logger
}

func NewGenerateTrialBalanceUseCase(repo AccountingRepository, logger *slog.Logger) *GenerateTrialBalanceUseCase {
 return &GenerateTrialBalanceUseCase{
  repo:   repo,
  logger: logger.With(slog.String("usecase", "generate_trial_balance")),
 }
}

func (uc *GenerateTrialBalanceUseCase) Execute(ctx context.Context, from, to time.Time) (*TrialBalanceResponse, error) {
 rows, err := uc.repo.GetTrialBalanceData(ctx, from, to)
 if err != nil {
  return nil, err
 }

 var totalDebit, totalCredit decimal.Decimal
 accounts := make([]TrialBalanceRow, len(rows))

 for i, r := range rows {
  accounts[i] = *r
  totalDebit = totalDebit.Add(r.EndingDebit)
  totalCredit = totalCredit.Add(r.EndingCredit)
 }

 isBalanced := totalDebit.Equal(totalCredit)

 return &TrialBalanceResponse{
  FromDate:    from.Format("2006-01-02"),
  ToDate:      to.Format("2006-01-02"),
  TotalDebit:  totalDebit,
  TotalCredit: totalCredit,
  IsBalanced:  isBalanced,
  Accounts:    accounts,
 }, nil
}
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [x] كتابة اختبارات وحدة لحالة `PostMoveUseCase` تختبر: النجاح، رفض القيد غير الموزون، ورفض الترحيل في فترة مغلقة.
- [x] التأكد من أن جميع حالات الاستخدام تستقبل `context.Context` وتمرر معرف التتبع والسجل المنظم `slog`.
- [x] ضمان استقلالية كل حالة استخدام في ملف منفرد لا يتجاوز 150-200 سطر لسهولة القراءة والصيانة.

### ملاحظات التنفيذ

- يقرأ `RepositoryFXRevaluationProvider` أرصدة العملات الأجنبية الموقعة من دفتر الأستاذ ويطلب أسعار الإقفال من `FXClosingRateProvider`، ثم يحسب `RevalueFXUseCase` فروق الأصول والالتزامات وينشئ قيد تسوية كمسودة. يركّب `NewUseCasesWithFXRates` هذه الاعتماديات وإعدادات الحسابات والدفتر؛ يجب أن يوفّر التطبيق محول أسعار موثوقًا.
- يتم ترحيل حدث الـ subledger وتسجيل مفتاح عدم التكرار في المعاملة نفسها لتفادي إنشاء قيد مكرر عند تزامن إعادة المحاولة.
