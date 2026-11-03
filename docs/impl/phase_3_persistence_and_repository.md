# الخطة التنفيذية التفصيلية — المرحلة 3: طبقة التخزين الذرية والمستودع الموحد

## (Phase 3: Persistence, Atomic Transactions & SQL Queries)

> **مسار التوثيق:** `docs/impl/phase_3_persistence_and_repository.md`  
> **الحالة:** معتمدة للتنفيذ الفوري  
> **الملفات المستهدفة في الشيفرة:**  
>
> - `internal/features/accounting/internal/repository.go`  
> - `internal/features/accounting/internal/queries.go`  
> - `internal/features/accounting/internal/repository_test.go`

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. تنفيذ المنفذ `AccountingRepository` بالكامل داخل المجلد المحمي `internal/features/accounting/internal/`.
2. حماية سلامة الترقيم التسلسلي القانوني غير المنقطع (Gapless Sequential Numbering) ضد التنازع المتزامن عبر قفل صف دفتر اليومية الصارم: `SELECT sequence_prefix, next_number FROM journals WHERE id = $1 FOR UPDATE`.
3. تنفيذ عملية الترحيل الذرية (`PostMoveAtomic`) وعملية العكس الذرية (`ReverseMoveAtomic`) داخل معاملة قاعدة بيانات واحدة (`pgx.Tx`) تضمن سلامة قيود ACID.
4. كتابة استعلامات ميزان المراجعة والأستاذ العام التراكمية في `queries.go` بكفاءة عالية على مستوى محرك PostgreSQL مباشرة لتجنب استهلاك الذاكرة.

---

## 2. تفاصيل الشيفرة والإجراءات الفعلية

### الخطوة 1: ملف المستودع والتنفيذ الذري (`internal/repository.go`)

```go
package internal

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "fmt"
 "time"

 "cashflow_backend/internal/features/accounting"
 "github.com/google/uuid"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
 pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
 return &PostgresRepository{pool: pool}
}

// 1. دليل الحسابات
func (r *PostgresRepository) CreateAccount(ctx context.Context, acc *accounting.Account) error {
 query := `
  INSERT INTO accounts (id, code, name, account_type, normal_balance, currency_id, parent_id, is_reconcilable, is_active, created_at, updated_at)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
 `
 _, err := r.pool.Exec(ctx, query,
  acc.ID, acc.Code, acc.Name, acc.Type, acc.NormalBalance, acc.CurrencyID,
  acc.ParentID, acc.IsReconcilable, acc.IsActive, acc.CreatedAt, acc.UpdatedAt,
 )
 return err
}

func (r *PostgresRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*accounting.Account, error) {
 query := `
  SELECT id, code, name, account_type, normal_balance, currency_id, parent_id, is_reconcilable, is_active, created_at, updated_at
  FROM accounts WHERE id = $1
 `
 row := r.pool.QueryRow(ctx, query, id)
 var a accounting.Account
 err := row.Scan(
  &a.ID, &a.Code, &a.Name, &a.Type, &a.NormalBalance, &a.CurrencyID,
  &a.ParentID, &a.IsReconcilable, &a.IsActive, &a.CreatedAt, &a.UpdatedAt,
 )
 if errors.Is(err, pgx.ErrNoRows) {
  return nil, accounting.ErrAccountNotFound
 }
 return &a, err
}

func (r *PostgresRepository) ListAccounts(ctx context.Context) ([]*accounting.Account, error) {
 query := `
  SELECT id, code, name, account_type, normal_balance, currency_id, parent_id, is_reconcilable, is_active, created_at, updated_at
  FROM accounts ORDER BY code ASC
 `
 rows, err := r.pool.Query(ctx, query)
 if err != nil {
  return nil, err
 }
 defer rows.Close()

 var list []*accounting.Account
 for rows.Next() {
  var a accounting.Account
  if err := rows.Scan(
   &a.ID, &a.Code, &a.Name, &a.Type, &a.NormalBalance, &a.CurrencyID,
   &a.ParentID, &a.IsReconcilable, &a.IsActive, &a.CreatedAt, &a.UpdatedAt,
  ); err != nil {
   return nil, err
  }
  list = append(list, &a)
 }
 return list, nil
}

// 2. إدارة الترقيم القانوني المقفل (SELECT ... FOR UPDATE)
func (r *PostgresRepository) GetNextSequenceWithLock(ctx context.Context, tx pgx.Tx, journalID uuid.UUID) (string, error) {
 query := `
  SELECT sequence_prefix, next_number 
  FROM journals 
  WHERE id = $1 
  FOR UPDATE
 `
 var prefix string
 var nextNum int64
 if err := tx.QueryRow(ctx, query, journalID).Scan(&prefix, &nextNum); err != nil {
  return "", fmt.Errorf("failed to lock journal sequence: %w", err)
 }

 seqName := fmt.Sprintf("%s%05d", prefix, nextNum)

 updateQuery := `UPDATE journals SET next_number = next_number + 1 WHERE id = $1`
 if _, err := tx.Exec(ctx, updateQuery, journalID); err != nil {
  return "", fmt.Errorf("failed to increment journal sequence: %w", err)
 }

 return seqName, nil
}

// 3. الترحيل الذري الشامل (Atomic Post)
func (r *PostgresRepository) PostMoveAtomic(ctx context.Context, move *accounting.AccountMove, audit *accounting.FinancialAuditLog) error {
 tx, err := r.pool.Begin(ctx)
 if err != nil {
  return err
 }
 defer tx.Rollback(ctx)

 // أ) حجز وتوليد الرقم التسلسلي القانوني بقفل الصف
 seqName, err := r.GetNextSequenceWithLock(ctx, tx, move.JournalID)
 if err != nil {
  return err
 }
 move.Name = seqName

 // ب) احتساب التجزئة التشفيرية للحصانة
 hashPayload := fmt.Sprintf("%s:%s:%s:%s", move.ID, move.Name, move.Date.Format("2006-01-02"), move.JournalID)
 h := sha256.Sum256([]byte(hashPayload))
 move.HashSignature = hex.EncodeToString(h[:])

 // ج) تحديث رأس القيد إلى posted
 updateMoveQuery := `
  UPDATE account_moves
  SET name = $1, state = $2, hash_signature = $3, posted_at = $4, posted_by = $5, updated_at = NOW()
  WHERE id = $6 AND state = 'draft'
 `
 cmdTag, err := tx.Exec(ctx, updateMoveQuery, move.Name, accounting.MoveStatePosted, move.HashSignature, move.PostedAt, move.PostedBy, move.ID)
 if err != nil {
  return err
 }
 if cmdTag.RowsAffected() == 0 {
  return errors.New("move is already posted or does not exist")
 }

 // د) حذف السطور القديمة إن وجدت مسودة ثم إعادة إدراج السطور المعتمدة
 delLinesQuery := `DELETE FROM account_move_lines WHERE move_id = $1`
 if _, err := tx.Exec(ctx, delLinesQuery, move.ID); err != nil {
  return err
 }

 insertLineQuery := `
  INSERT INTO account_move_lines (id, move_id, account_id, partner_id, debit, credit, amount_currency, currency_id, cost_center_id, matching_number, date, created_at)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
 `
 for _, line := range move.Lines {
  _, err := tx.Exec(ctx, insertLineQuery,
   line.ID, move.ID, line.AccountID, line.PartnerID, line.Debit, line.Credit,
   line.AmountCurrency, line.CurrencyID, line.CostCenterID, line.MatchingNumber, line.Date, line.CreatedAt,
  )
  if err != nil {
   return fmt.Errorf("failed to insert move line: %w", err)
  }
 }

 // هـ) إدراج سجل التدقيق المشفر
 if audit != nil {
  auditQuery := `
   INSERT INTO financial_audit_logs (entity_type, entity_id, action, actor_id, payload_before, payload_after, current_log_hash, timestamp)
   VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
  `
  _, err = tx.Exec(ctx, auditQuery,
   audit.EntityType, audit.EntityID, audit.Action, audit.ActorID,
   audit.PayloadBefore, audit.PayloadAfter, audit.CurrentLogHash, audit.Timestamp,
  )
  if err != nil {
   return fmt.Errorf("failed to insert audit log: %w", err)
  }
 }

 return tx.Commit(ctx)
}
```

---

### الخطوة 2: استعلامات التقارير والقوائم المالية (`internal/queries.go`)

```go
package internal

import (
 "context"
 "time"

 "cashflow_backend/internal/features/accounting"
)

// GetTrialBalanceData استعلام ميزان المراجعة المجمع
func (r *PostgresRepository) GetTrialBalanceData(ctx context.Context, fromDate, toDate time.Time) ([]*accounting.TrialBalanceRow, error) {
 query := `
  WITH initial_balances AS (
   SELECT 
    l.account_id,
    COALESCE(SUM(l.debit), 0) AS init_debit,
    COALESCE(SUM(l.credit), 0) AS init_credit
   FROM account_move_lines l
   JOIN account_moves m ON l.move_id = m.id
   WHERE m.state = 'posted' AND l.date < $1
   GROUP BY l.account_id
  ),
  period_balances AS (
   SELECT 
    l.account_id,
    COALESCE(SUM(l.debit), 0) AS p_debit,
    COALESCE(SUM(l.credit), 0) AS p_credit
   FROM account_move_lines l
   JOIN account_moves m ON l.move_id = m.id
   WHERE m.state = 'posted' AND l.date >= $1 AND l.date <= $2
   GROUP BY l.account_id
  )
  SELECT 
   a.id,
   a.code,
   a.name,
   COALESCE(i.init_debit, 0) AS initial_debit,
   COALESCE(i.init_credit, 0) AS initial_credit,
   COALESCE(p.p_debit, 0) AS period_debit,
   COALESCE(p.p_credit, 0) AS period_credit,
   (COALESCE(i.init_debit, 0) + COALESCE(p.p_debit, 0)) AS ending_debit,
   (COALESCE(i.init_credit, 0) + COALESCE(p.p_credit, 0)) AS ending_credit
  FROM accounts a
  LEFT JOIN initial_balances i ON a.id = i.account_id
  LEFT JOIN period_balances p ON a.id = p.account_id
  ORDER BY a.code ASC;
 `

 rows, err := r.pool.Query(ctx, query, fromDate, toDate)
 if err != nil {
  return nil, err
 }
 defer rows.Close()

 var results []*accounting.TrialBalanceRow
 for rows.Next() {
  var row accounting.TrialBalanceRow
  if err := rows.Scan(
   &row.AccountID, &row.AccountCode, &row.AccountName,
   &row.InitialDebit, &row.InitialCredit,
   &row.PeriodDebit, &row.PeriodCredit,
   &row.EndingDebit, &row.EndingCredit,
  ); err != nil {
   return nil, err
  }
  results = append(results, &row)
 }

 return results, nil
}
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [x] التأكد من أن الترحيل يحدث في `pgx.Tx` واحدة تضمن التراجع التام في حال فشل أي سطر أو فشل إدراج سجل التدقيق.
- [x] التأكد من تطبيق قفل `FOR UPDATE` على دفتر اليومية لضمان تسلسل ترقيم غير منقطع حتى تحت ضغط التزامن العالي.
- [x] خلو جداول القيود والسطور من أي أوامر `UPDATE` تمس السطور المرحلة، حمايةً للمبدأ المحاسبي الخالد (Immutability).

### اختبارات التكامل على PostgreSQL

يستخدم اختبار المستودع مخططاً مؤقتاً ومعزولاً، ويتحقق من ترحيل 50 قيداً بالتزامن، والعكس، والتقارير، وعدم استهلاك رقم عند فشل سجل التدقيق. شغّله عند توفر قاعدة اختبار:

```sh
ACCOUNTING_TEST_DATABASE_URL='postgres:///postgres?host=/var/run/postgresql' \
  go test ./internal/features/accounting/internal -run TestPostgresRepositoryIntegration -count=1
```
