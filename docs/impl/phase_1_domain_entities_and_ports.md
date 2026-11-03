# الخطة التنفيذية التفصيلية — المرحلة 1: وثيقة النطاق، المنافذ التجريدية، ونماذج النطاق والقواعد الرياضية

## (Phase 1: Domain Entities, Ports & Invariants)

> **مسار التوثيق:** `docs/impl/phase_1_domain_entities_and_ports.md`  
> **الحالة:** معتمدة للتنفيذ الفوري  
> **الملفات المستهدفة في الشيفرة:**  
>
> - `internal/features/accounting/doc.go`  
> - `internal/features/accounting/ports.go`  
> - `internal/features/accounting/entity.go`  
> - `internal/features/accounting/entity_test.go`

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. توثيق حدود سياق المحاسبة المالية في `doc.go` وتأكيد انعزالها عن أي تفاصيل تجارية (مخزون، فواتير تجارية، أوامر بيع/شراء).
2. تعريف واجهة المنفذ المركزية `AccountingRepository` ومنافذ بث الأحداث في `ports.go`.
3. صياغة نماذج النطاق النقية (`Account`, `FiscalPeriod`, `Journal`, `AccountMove`, `AccountMoveLine`) في `entity.go` مع خلوها التام من وسوم الـ JSON وتبعيات HTTP.
4. برمجة قواعد العمل الصلبة (Invariants): التوازن الصفري الإلزامي ($\sum \text{Debit} - \sum \text{Credit} = 0$)، منع المبالغ السالبة، وحصانة القيود ضد التعديل بعد ترحيلها.
5. كتابة اختبارات الوحدة الشاملة لنماذج النطاق في `entity_test.go`.

---

## 2. تفاصيل الشيفرة والإجراءات الفعلية

### الخطوة 1: ملف التوثيق المرجعي للحزمة (`doc.go`)

```go
// Package accounting provides the core double-entry financial accounting engine
// for the Cashflow system, adhering to IFRS/IAS and US GAAP standards.
//
// Responsibilities:
//   - Chart of Accounts management and natural balance validation (Debit/Credit).
//   - Fiscal period calendar and hard/soft closing state enforcement.
//   - Immutable double-entry General Ledger (account_moves & account_move_lines).
//   - Zero-sum mathematical balance verification before atomic posting.
//   - Gapless sequential numbering for legal journals.
//   - Financial reporting generation (Trial Balance, Balance Sheet, Income Statement).
//   - Economic event ingestion via Subledger Accounting (SLA).
//
// Architectural Boundary:
//   This package is purely financial. It does NOT manage inventory items, sales orders,
//   purchase orders, customer CRM profiles, or physical delivery workflows.
//   External systems interface with accounting exclusively through economic events
//   or standardized accounting APIs.
package accounting
```

---

### الخطوة 2: ملف المنافذ التجريدية (`ports.go`)

```go
package accounting

import (
 "context"
 "time"

 "github.com/google/uuid"
 "github.com/jackc/pgx/v5"
)

// AccountingRepository المنفذ الأساسي لطبقة الوصول إلى البيانات
type AccountingRepository interface {
 // 1. دليل الحسابات
 CreateAccount(ctx context.Context, acc *Account) error
 GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error)
 GetAccountByCode(ctx context.Context, code string) (*Account, error)
 ListAccounts(ctx context.Context) ([]*Account, error)

 // 2. الفترات والسنوات المالية
 CreateFiscalYear(ctx context.Context, year *FiscalYear, periods []*FiscalPeriod) error
 GetPeriodByDate(ctx context.Context, date time.Time) (*FiscalPeriod, error)
 GetPeriodByID(ctx context.Context, id uuid.UUID) (*FiscalPeriod, error)
 UpdatePeriodState(ctx context.Context, id uuid.UUID, state PeriodState, actorID uuid.UUID) error

 // 3. دفاتر اليومية والترقيم التسلسلي القانوني
 GetJournalByID(ctx context.Context, id uuid.UUID) (*Journal, error)
 GetNextSequenceWithLock(ctx context.Context, tx pgx.Tx, journalID uuid.UUID) (string, error)

 // 4. القيود والأستاذ العام
 SaveDraftMove(ctx context.Context, move *AccountMove) error
 GetMoveByID(ctx context.Context, id uuid.UUID) (*AccountMove, error)
 PostMoveAtomic(ctx context.Context, move *AccountMove, audit *FinancialAuditLog) error
 ReverseMoveAtomic(ctx context.Context, originalMoveID uuid.UUID, reversalMove *AccountMove, audit *FinancialAuditLog) error

 // 5. استعلامات التقارير والقوائم المالية (Read Model)
 GetTrialBalanceData(ctx context.Context, fromDate, toDate time.Time) ([]*TrialBalanceRow, error)
 GetGeneralLedgerLines(ctx context.Context, accountID uuid.UUID, fromDate, toDate time.Time) ([]*GeneralLedgerLineRow, error)

 // 6. استيعاب الأحداث الاقتصادية (Idempotency)
 GetProcessedEventMove(ctx context.Context, idempotencyKey string) (*AccountMove, error)
 SaveProcessedEvent(ctx context.Context, key, eventType string, moveID uuid.UUID) error
}

// EventPublisher منفذ بث أحداث النطاق خارج حدود الميزة
type EventPublisher interface {
 Publish(ctx context.Context, eventName string, payload any) error
}
```

---

### الخطوة 3: كائنات النطاق وقواعد العمل الصلبة (`entity.go`)

```go
package accounting

import (
 "errors"
 "fmt"
 "time"

 "github.com/google/uuid"
 "github.com/shopspring/decimal"
)

// ثوابت أنواع الحسابات وطبيعتها
type AccountType string

const (
 AccountTypeAsset     AccountType = "asset"
 AccountTypeLiability AccountType = "liability"
 AccountTypeEquity    AccountType = "equity"
 AccountTypeIncome    AccountType = "income"
 AccountTypeExpense   AccountType = "expense"
)

type NormalBalance string

const (
 NormalBalanceDebit  NormalBalance = "debit"
 NormalBalanceCredit NormalBalance = "credit"
)

// ثوابت حالات الفترات المحاسبية
type PeriodState string

const (
 PeriodStateOpen       PeriodState = "open"
 PeriodStateSoftClosed PeriodState = "soft_closed"
 PeriodStateHardClosed PeriodState = "hard_closed"
)

// ثوابت حالات القيود
type MoveState string

const (
 MoveStateDraft    MoveState = "draft"
 MoveStatePosted   MoveState = "posted"
 MoveStateReversed MoveState = "reversed"
)

// أخطاء النطاق المعيارية
var (
 ErrAccountNotFound               = errors.New("accounting: account not found")
 ErrDuplicateAccountCode          = errors.New("accounting: duplicate account code")
 ErrPeriodClosed                  = errors.New("accounting: fiscal period is closed for posting")
 ErrPeriodNotFound                = errors.New("accounting: no active fiscal period for date")
 ErrMoveRequiresAtLeastTwoLines   = errors.New("accounting: a journal entry requires at least two lines")
 ErrNegativeAmountsNotAllowed     = errors.New("accounting: negative debit or credit amounts are prohibited")
 ErrZeroAmountLine                = errors.New("accounting: move line must have non-zero debit or credit")
 ErrUnbalancedMove                = errors.New("accounting: move is unbalanced; debits must strictly equal credits")
 ErrMoveAlreadyPosted             = errors.New("accounting: move is already posted and immutable")
 ErrCannotReverseUnpostedMove     = errors.New("accounting: only posted moves can be reversed")
)

// Account كيان الحساب المالي
type Account struct {
 ID             uuid.UUID
 Code           string
 Name           string
 Type           AccountType
 NormalBalance  NormalBalance
 CurrencyID     string
 ParentID       *uuid.UUID
 IsReconcilable bool
 IsActive       bool
 CreatedAt      time.Time
 UpdatedAt      time.Time
}

// FiscalYear كيان السنة المالية
type FiscalYear struct {
 ID        uuid.UUID
 Year      int
 StartDate time.Time
 EndDate   time.Time
 State     string
 CreatedAt time.Time
}

// FiscalPeriod كيان الفترة المحاسبية الشهرية
type FiscalPeriod struct {
 ID           uuid.UUID
 FiscalYearID uuid.UUID
 PeriodNumber int
 Name         string
 StartDate    time.Time
 EndDate      time.Time
 State        PeriodState
 ClosedAt     *time.Time
 ClosedBy     *uuid.UUID
 CreatedAt    time.Time
}

func (p *FiscalPeriod) CanPost() error {
 if p.State != PeriodStateOpen {
  return fmt.Errorf("%w: period %s is in %s state", ErrPeriodClosed, p.Name, p.State)
 }
 return nil
}

// Journal كيان دفتر اليومية القانوني
type Journal struct {
 ID             uuid.UUID
 Code           string
 Name           string
 Type           string
 SequencePrefix string
 NextNumber     int64
 IsActive       bool
 CreatedAt      time.Time
}

// AccountMoveLine سطر القيد المحاسبي
type AccountMoveLine struct {
 ID             uuid.UUID
 MoveID         uuid.UUID
 AccountID      uuid.UUID
 PartnerID      *uuid.UUID
 Debit          decimal.Decimal
 Credit         decimal.Decimal
 AmountCurrency decimal.Decimal
 CurrencyID     string
 CostCenterID   *uuid.UUID
 MatchingNumber string
 Date           time.Time
 CreatedAt      time.Time
}

// AccountMove قيد اليومية (Aggregate Root)
type AccountMove struct {
 ID             uuid.UUID
 Name           string
 JournalID      uuid.UUID
 PeriodID       uuid.UUID
 Date           time.Time
 State          MoveState
 Ref            string
 ReversedMoveID *uuid.UUID
 HashSignature  string
 PostedAt       *time.Time
 PostedBy       *uuid.UUID
 CreatedAt      time.Time
 UpdatedAt      time.Time
 Lines          []AccountMoveLine
}

// ValidateBalance التحقق الرياضي الصارم من توازن القيد المزدوج
func (m *AccountMove) ValidateBalance() error {
 if len(m.Lines) < 2 {
  return ErrMoveRequiresAtLeastTwoLines
 }

 var totalDebit, totalCredit decimal.Decimal

 for _, line := range m.Lines {
  if line.Debit.IsNegative() || line.Credit.IsNegative() {
   return ErrNegativeAmountsNotAllowed
  }
  if line.Debit.IsZero() && line.Credit.IsZero() {
   return ErrZeroAmountLine
  }
  totalDebit = totalDebit.Add(line.Debit)
  totalCredit = totalCredit.Add(line.Credit)
 }

 if !totalDebit.Equal(totalCredit) {
  return fmt.Errorf("%w: debit sum (%s) != credit sum (%s)", ErrUnbalancedMove, totalDebit, totalCredit)
 }

 return nil
}

// FinancialAuditLog سجل التدقيق المالي المشفر
type FinancialAuditLog struct {
 ID              int64
 EntityType      string
 EntityID        uuid.UUID
 Action          string
 ActorID         uuid.UUID
 PayloadBefore   []byte
 PayloadAfter    []byte
 IPAddress       string
 PreviousLogHash string
 CurrentLogHash  string
 Timestamp       time.Time
}
```

---

### الخطوة 4: اختبارات النطاق والتوازن (`entity_test.go`)

```go
package accounting_test

import (
 "testing"
 "time"

 "cashflow_backend/internal/features/accounting"
 "github.com/google/uuid"
 "github.com/shopspring/decimal"
 "github.com/stretchr/testify/assert"
)

func TestAccountMove_ValidateBalance(t *testing.T) {
 acc1 := uuid.New()
 acc2 := uuid.New()

 t.Run("balanced move passes validation", func(t *testing.T) {
  move := accounting.AccountMove{
   ID:   uuid.New(),
   Date: time.Now(),
   Lines: []accounting.AccountMoveLine{
    {AccountID: acc1, Debit: decimal.NewFromFloat(500.50), Credit: decimal.Zero},
    {AccountID: acc2, Debit: decimal.Zero, Credit: decimal.NewFromFloat(500.50)},
   },
  }
  err := move.ValidateBalance()
  assert.NoError(t, err)
 })

 t.Run("unbalanced move fails validation", func(t *testing.T) {
  move := accounting.AccountMove{
   ID:   uuid.New(),
   Date: time.Now(),
   Lines: []accounting.AccountMoveLine{
    {AccountID: acc1, Debit: decimal.NewFromFloat(500.50), Credit: decimal.Zero},
    {AccountID: acc2, Debit: decimal.Zero, Credit: decimal.NewFromFloat(400.00)},
   },
  }
  err := move.ValidateBalance()
  assert.ErrorIs(t, err, accounting.ErrUnbalancedMove)
 })

 t.Run("single line move fails validation", func(t *testing.T) {
  move := accounting.AccountMove{
   ID:   uuid.New(),
   Date: time.Now(),
   Lines: []accounting.AccountMoveLine{
    {AccountID: acc1, Debit: decimal.NewFromFloat(100), Credit: decimal.Zero},
   },
  }
  err := move.ValidateBalance()
  assert.ErrorIs(t, err, accounting.ErrMoveRequiresAtLeastTwoLines)
 })

 t.Run("negative amount fails validation", func(t *testing.T) {
  move := accounting.AccountMove{
   ID:   uuid.New(),
   Date: time.Now(),
   Lines: []accounting.AccountMoveLine{
    {AccountID: acc1, Debit: decimal.NewFromFloat(-100), Credit: decimal.Zero},
    {AccountID: acc2, Debit: decimal.Zero, Credit: decimal.NewFromFloat(-100)},
   },
  }
  err := move.ValidateBalance()
  assert.ErrorIs(t, err, accounting.ErrNegativeAmountsNotAllowed)
 })
}
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [x] تشغيل `go test -v ./internal/features/accounting` والتأكد من نجاح اختبارات التوازن بنسبة 100%.
- [x] خلو `entity.go` و `ports.go` تماماً من أي وسوم `json` أو استيراد لمكتبات `net/http` أو `chi`.
- [x] مطابقة التوازن الرياضي الصارم مع دعم الكسور بدقة متناهية (`decimal.Decimal`).
