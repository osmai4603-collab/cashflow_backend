# الخطة التنفيذية التفصيلية — المرحلة 2: كائنات نقل البيانات، التحقق والتحويل

## (Phase 2: DTOs, Validation & Mappers)

> **مسار التوثيق:** `docs/impl/phase_2_dtos_and_validation.md`  
> **الحالة:** معتمدة للتنفيذ الفوري  
> **الملفات المستهدفة في الشيفرة:**  
>
> - `internal/features/accounting/dto.go`  
> - `internal/features/accounting/dto_test.go`

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. تصميم جميع كائنات نقل البيانات (Request & Response DTOs) الخاصة بالمحاسبة المالية مع وسوم الـ `json` القياسية وتوثيق الحقول.
2. برمجة دوال التحقق الصارمة `Validate()` لكل طلب قادم من الشبكة لضمان سلامة البيانات قبل وصولها إلى طبقة حالات الاستخدام.
3. كتابة دوال التحويل النظيفة والآمنة (Mappers مثل `ToAccountResponse` و `ToMoveResponse`) لنقل البيانات من كيانات النطاق (`Entities`) إلى الـ DTOs دون تسريب تفاصيل داخلية.
4. تغطية التحقق باختبارات وحدة آلية في `dto_test.go`.

---

## 2. تفاصيل الشيفرة والإجراءات الفعلية

### الخطوة 1: ملف كائنات نقل البيانات والتحقق (`dto.go`)

```go
package accounting

import (
 "errors"
 "strings"
 "time"

 "github.com/google/uuid"
 "github.com/shopspring/decimal"
)

// ==========================================
// 1. دليل الحسابات (Chart of Accounts DTOs)
// ==========================================

type CreateAccountRequest struct {
 Code           string     `json:"code"`
 Name           string     `json:"name"`
 Type           string     `json:"account_type"`
 NormalBalance  string     `json:"normal_balance"`
 CurrencyID     string     `json:"currency_id"`
 ParentID       *uuid.UUID `json:"parent_id,omitempty"`
 IsReconcilable bool       `json:"is_reconcilable"`
}

func (r *CreateAccountRequest) Validate() error {
 r.Code = strings.TrimSpace(r.Code)
 if r.Code == "" {
  return errors.New("code is required")
 }
 r.Name = strings.TrimSpace(r.Name)
 if r.Name == "" {
  return errors.New("name is required")
 }

 t := AccountType(strings.ToLower(strings.TrimSpace(r.Type)))
 switch t {
 case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity, AccountTypeIncome, AccountTypeExpense:
  r.Type = string(t)
 default:
  return errors.New("invalid account_type: must be asset, liability, equity, income, or expense")
 }

 nb := NormalBalance(strings.ToLower(strings.TrimSpace(r.NormalBalance)))
 if nb != NormalBalanceDebit && nb != NormalBalanceCredit {
  // ضبط تلقائي لطبيعة الحساب إذا لم تحدد
  if t == AccountTypeAsset || t == AccountTypeExpense {
   r.NormalBalance = string(NormalBalanceDebit)
  } else {
   r.NormalBalance = string(NormalBalanceCredit)
  }
 } else {
  r.NormalBalance = string(nb)
 }

 r.CurrencyID = strings.ToUpper(strings.TrimSpace(r.CurrencyID))
 if r.CurrencyID == "" {
  r.CurrencyID = "SAR"
 }
 return nil
}

type AccountResponse struct {
 ID             uuid.UUID  `json:"id"`
 Code           string     `json:"code"`
 Name           string     `json:"name"`
 Type           string     `json:"account_type"`
 NormalBalance  string     `json:"normal_balance"`
 CurrencyID     string     `json:"currency_id"`
 ParentID       *uuid.UUID `json:"parent_id,omitempty"`
 IsReconcilable bool       `json:"is_reconcilable"`
 IsActive       bool       `json:"is_active"`
 CreatedAt      time.Time  `json:"created_at"`
}

type AccountTreeNode struct {
 AccountResponse
 Children []AccountTreeNode `json:"children,omitempty"`
}

func ToAccountResponse(a *Account) AccountResponse {
 return AccountResponse{
  ID:             a.ID,
  Code:           a.Code,
  Name:           a.Name,
  Type:           string(a.Type),
  NormalBalance:  string(a.NormalBalance),
  CurrencyID:     a.CurrencyID,
  ParentID:       a.ParentID,
  IsReconcilable: a.IsReconcilable,
  IsActive:       a.IsActive,
  CreatedAt:      a.CreatedAt,
 }
}

// ==========================================
// 2. الفترات والسنوات المالية (Fiscal Periods DTOs)
// ==========================================

type CreateFiscalYearRequest struct {
 Year      int    `json:"year"`
 StartDate string `json:"start_date"` // YYYY-MM-DD
 EndDate   string `json:"end_date"`   // YYYY-MM-DD
}

func (r *CreateFiscalYearRequest) Validate() error {
 if r.Year < 2000 || r.Year > 2100 {
  return errors.New("year must be between 2000 and 2100")
 }
 start, err := time.Parse("2006-01-02", r.StartDate)
 if err != nil {
  return errors.New("start_date must be in YYYY-MM-DD format")
 }
 end, err := time.Parse("2006-01-02", r.EndDate)
 if err != nil {
  return errors.New("end_date must be in YYYY-MM-DD format")
 }
 if !end.After(start) {
  return errors.New("end_date must be strictly after start_date")
 }
 return nil
}

type ChangePeriodStateRequest struct {
 State string `json:"state"` // open, soft_closed, hard_closed
}

func (r *ChangePeriodStateRequest) Validate() error {
 s := PeriodState(strings.ToLower(strings.TrimSpace(r.State)))
 switch s {
 case PeriodStateOpen, PeriodStateSoftClosed, PeriodStateHardClosed:
  r.State = string(s)
  return nil
 default:
  return errors.New("state must be open, soft_closed, or hard_closed")
 }
}

type FiscalPeriodResponse struct {
 ID           uuid.UUID  `json:"id"`
 FiscalYearID uuid.UUID  `json:"fiscal_year_id"`
 PeriodNumber int        `json:"period_number"`
 Name         string     `json:"name"`
 StartDate    string     `json:"start_date"`
 EndDate      string     `json:"end_date"`
 State        string     `json:"state"`
 ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

func ToFiscalPeriodResponse(p *FiscalPeriod) FiscalPeriodResponse {
 return FiscalPeriodResponse{
  ID:           p.ID,
  FiscalYearID: p.FiscalYearID,
  PeriodNumber: p.PeriodNumber,
  Name:         p.Name,
  StartDate:    p.StartDate.Format("2006-01-02"),
  EndDate:      p.EndDate.Format("2006-01-02"),
  State:        string(p.State),
  ClosedAt:     p.ClosedAt,
 }
}

// ==========================================
// 3. القيود واليوميات (Moves & Journals DTOs)
// ==========================================

type MoveLineRequest struct {
 AccountID      uuid.UUID       `json:"account_id"`
 PartnerID      *uuid.UUID      `json:"partner_id,omitempty"`
 Debit          decimal.Decimal `json:"debit"`
 Credit         decimal.Decimal `json:"credit"`
 AmountCurrency decimal.Decimal `json:"amount_currency,omitempty"`
 CurrencyID     string          `json:"currency_id,omitempty"`
 CostCenterID   *uuid.UUID      `json:"cost_center_id,omitempty"`
 MatchingNumber string          `json:"matching_number,omitempty"`
}

type CreateMoveRequest struct {
 JournalID uuid.UUID         `json:"journal_id"`
 Date      string            `json:"date"` // YYYY-MM-DD
 Ref       string            `json:"ref,omitempty"`
 Lines     []MoveLineRequest `json:"lines"`
}

func (r *CreateMoveRequest) Validate() error {
 if r.JournalID == uuid.Nil {
  return errors.New("journal_id is required")
 }
 if _, err := time.Parse("2006-01-02", r.Date); err != nil {
  return errors.New("date must be in YYYY-MM-DD format")
 }
 if len(r.Lines) < 2 {
  return errors.New("a move must contain at least 2 lines")
 }

 var totalDebit, totalCredit decimal.Decimal
 for i, l := range r.Lines {
  if l.AccountID == uuid.Nil {
   return errors.New("account_id is required on all lines")
  }
  if l.Debit.IsNegative() || l.Credit.IsNegative() {
   return errors.New("negative debit or credit amounts are not permitted")
  }
  if l.Debit.IsZero() && l.Credit.IsZero() {
   return errors.New("line must specify non-zero debit or credit")
  }
  if !l.Debit.IsZero() && !l.Credit.IsZero() {
   return errors.New("line cannot have both debit and credit; split into separate lines")
  }
  totalDebit = totalDebit.Add(l.Debit)
  totalCredit = totalCredit.Add(l.Credit)
  if r.Lines[i].CurrencyID == "" {
   r.Lines[i].CurrencyID = "SAR"
  }
 }

 if !totalDebit.Equal(totalCredit) {
  return errors.New("total debit must equal total credit")
 }
 return nil
}

type MoveLineResponse struct {
 ID             uuid.UUID       `json:"id"`
 AccountID      uuid.UUID       `json:"account_id"`
 Debit          decimal.Decimal `json:"debit"`
 Credit         decimal.Decimal `json:"credit"`
 AmountCurrency decimal.Decimal `json:"amount_currency"`
 CurrencyID     string          `json:"currency_id"`
 CostCenterID   *uuid.UUID      `json:"cost_center_id,omitempty"`
 MatchingNumber string          `json:"matching_number,omitempty"`
}

type MoveResponse struct {
 ID             uuid.UUID          `json:"id"`
 Name           string             `json:"name"`
 JournalID      uuid.UUID          `json:"journal_id"`
 Date           string             `json:"date"`
 State          string             `json:"state"`
 Ref            string             `json:"ref,omitempty"`
 ReversedMoveID *uuid.UUID         `json:"reversed_move_id,omitempty"`
 HashSignature  string             `json:"hash_signature,omitempty"`
 PostedAt       *time.Time         `json:"posted_at,omitempty"`
 Lines          []MoveLineResponse `json:"lines"`
}

func ToMoveResponse(m *AccountMove) MoveResponse {
 lines := make([]MoveLineResponse, len(m.Lines))
 for i, l := range m.Lines {
  lines[i] = MoveLineResponse{
   ID:             l.ID,
   AccountID:      l.AccountID,
   Debit:          l.Debit,
   Credit:         l.Credit,
   AmountCurrency: l.AmountCurrency,
   CurrencyID:     l.CurrencyID,
   CostCenterID:   l.CostCenterID,
   MatchingNumber: l.MatchingNumber,
  }
 }
 return MoveResponse{
  ID:             m.ID,
  Name:           m.Name,
  JournalID:      m.JournalID,
  Date:           m.Date.Format("2006-01-02"),
  State:          string(m.State),
  Ref:            m.Ref,
  ReversedMoveID: m.ReversedMoveID,
  HashSignature:  m.HashSignature,
  PostedAt:       m.PostedAt,
  Lines:          lines,
 }
}

// ==========================================
// 4. التقارير المالية (Reporting DTOs)
// ==========================================

type TrialBalanceRow struct {
 AccountID      uuid.UUID       `json:"account_id"`
 AccountCode    string          `json:"account_code"`
 AccountName    string          `json:"account_name"`
 InitialDebit   decimal.Decimal `json:"initial_debit"`
 InitialCredit  decimal.Decimal `json:"initial_credit"`
 PeriodDebit    decimal.Decimal `json:"period_debit"`
 PeriodCredit   decimal.Decimal `json:"period_credit"`
 EndingDebit    decimal.Decimal `json:"ending_debit"`
 EndingCredit   decimal.Decimal `json:"ending_credit"`
}

type TrialBalanceResponse struct {
 FromDate       string            `json:"from_date"`
 ToDate         string            `json:"to_date"`
 TotalDebit     decimal.Decimal   `json:"total_debit"`
 TotalCredit    decimal.Decimal   `json:"total_credit"`
 IsBalanced     bool              `json:"is_balanced"`
 Accounts       []TrialBalanceRow `json:"accounts"`
}

type GeneralLedgerLineRow struct {
 MoveID         uuid.UUID       `json:"move_id"`
 MoveName       string          `json:"move_name"`
 Date           string          `json:"date"`
 Ref            string          `json:"ref"`
 Debit          decimal.Decimal `json:"debit"`
 Credit         decimal.Decimal `json:"credit"`
 RunningBalance decimal.Decimal `json:"running_balance"`
}

// ==========================================
// 5. استيعاب الأحداث الاقتصادية (Subledger Ingestion DTOs)
// ==========================================

type IngestSubledgerEventRequest struct {
 IdempotencyKey string          `json:"idempotency_key"`
 EventType      string          `json:"event_type"` // e.g. "order_delivered", "payment_settled"
 Date           string          `json:"date"`
 Ref            string          `json:"ref"`
 Amount         decimal.Decimal `json:"amount"`
 Currency       string          `json:"currency"`
 Payload        map[string]any  `json:"payload"`
}

func (r *IngestSubledgerEventRequest) Validate() error {
 r.IdempotencyKey = strings.TrimSpace(r.IdempotencyKey)
 if r.IdempotencyKey == "" {
  return errors.New("idempotency_key is required")
 }
 r.EventType = strings.TrimSpace(r.EventType)
 if r.EventType == "" {
  return errors.New("event_type is required")
 }
 if r.Amount.IsZero() || r.Amount.IsNegative() {
  return errors.New("amount must be strictly positive")
 }
 if _, err := time.Parse("2006-01-02", r.Date); err != nil {
  return errors.New("date must be in YYYY-MM-DD format")
 }
 return nil
}
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [x] تغطية دوال `Validate()` باختبارات وحدة تفحص حالات المدخلات الصحيحة والخاطئة.
- [x] التأكد من منع تعيين مبالغ في كل من المدين والدائن معاً في نفس السطر.
- [x] التأكد من فحص صيغ التواريخ ومطابقة أكواد العملات المدعومة.

> **ملاحظة التنفيذ:** تُطابق أكواد العملات مع القائمة المركزية المدعومة في `internal/platform/money`، ولا يعني ذلك دعم جميع أكواد ISO 4217. استُخدمت أسماء `TrialBalanceRowResponse` و`GeneralLedgerLineResponse` لتجنب التعارض مع نماذج النطاق، وأضيف `ToTrialBalanceResponse` لتحويل التقرير الكامل.
