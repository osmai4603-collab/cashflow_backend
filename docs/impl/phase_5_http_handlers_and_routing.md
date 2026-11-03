# الخطة التنفيذية التفصيلية — المرحلة 5: طبقة النقل ومعالجات الـ HTTP المستقلة وتكامل المسارات

## (Phase 5: HTTP Transport Handlers & Route Registration)

> **مسار التوثيق:** `docs/impl/phase_5_http_handlers_and_routing.md`  
> **الحالة:** مكتملة التنفيذ
> **الملفات المستهدفة في الشيفرة:**  
>
> - `internal/features/accounting/handler.go`  
> - `internal/features/accounting/account_handler.go`  
> - `internal/features/accounting/period_handler.go`  
> - `internal/features/accounting/move_handler.go`  
> - `internal/features/accounting/report_handler.go`  
> - `internal/features/accounting/subledger_handler.go`  
> - `internal/features/accounting/handler_test.go`
> - `internal/features/accounting/list_accounts_usecase.go`
> - `internal/features/accounting/get_move_usecase.go`
> - `internal/features/accounting/wiring/postgres.go`
> - `internal/platform/app/router.go`
> - `internal/platform/app/router_test.go`
> - `internal/platform/app/app.go`

---

## 1. الأهداف والنتائج المستهدفة (Deliverables)

1. تطبيق مبدأ فصل معالجات الـ HTTP بحيث يتم إفراد كل مجال وظيفي بمعالج مستقل (`<domain>_handler.go`).
2. تجميع وتسجيل المسارات تحت بادئة موحدة `/api/v1/accounting` في `handler.go` باستخدام راوتر `go-chi/chi/v5`.
3. التحقق التلقائي من صحة مدخلات الـ JSON وإرجاع أخطاء منسقة وفق معايير المنصة (`Standard JSON Errors`).
4. اختبار مسارات الـ HTTP بواسطة `net/http/httptest` للتحقق من رموز الاستجابة (`200 OK`, `201 Created`, `400 Bad Request`, `422 Unprocessable Entity`).

---

## 2. تفاصيل الشيفرة والإجراءات الفعلية

### الخطوة 1: الموزع المركزي للمسارات (`handler.go`)

```go
package accounting

import (
 "encoding/json"
 "log/slog"
 "net/http"

 "github.com/go-chi/chi/v5"
)

// RegisterRoutes يسجل جميع مسارات المحاسبة تحت الراوتر المحدد
func RegisterRoutes(r chi.Router, uc *UseCases, logger *slog.Logger) {
 accHandler := NewAccountHandler(uc, logger)
 periodHandler := NewPeriodHandler(uc, logger)
 moveHandler := NewMoveHandler(uc, logger)
 reportHandler := NewReportHandler(uc, logger)
 subledgerHandler := NewSubledgerHandler(uc, logger)

 r.Route("/api/v1/accounting", func(sub chi.Router) {
  // 1. دليل الحسابات
  sub.Post("/accounts", accHandler.HandleCreate)
  sub.Get("/accounts", accHandler.HandleList)
  sub.Get("/accounts/tree", accHandler.HandleTree)

  // 2. الفترات المالية
  sub.Post("/fiscal-years", periodHandler.HandleCreateYear)
  sub.Post("/fiscal-periods/{id}/state", periodHandler.HandleChangeState)

  // 3. القيود والأستاذ العام
  sub.Post("/moves", moveHandler.HandleCreateDraft)
  sub.Get("/moves/{id}", moveHandler.HandleGet)
  sub.Post("/moves/{id}/post", moveHandler.HandlePost)
  sub.Post("/moves/{id}/reverse", moveHandler.HandleReverse)

  // 4. التقارير المالية
  sub.Get("/reports/trial-balance", reportHandler.HandleTrialBalance)

  // 5. استيعاب أحداث المحاسبة الفرعية (SLA)
  sub.Post("/subledger/events", subledgerHandler.HandleIngest)
 })
}

// دوال مساعدة موحدة للاستجابة
func respondJSON(w http.ResponseWriter, status int, data any) {
 w.Header().Set("Content-Type", "application/json")
 w.WriteHeader(status)
 _ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
 respondJSON(w, status, map[string]string{"error": message})
}
```

---

### الخطوة 2: معالج مسارات القيود اليومية والترحيل (`move_handler.go`)

```go
package accounting

import (
 "encoding/json"
 "errors"
 "log/slog"
 "net/http"
 "time"

 "github.com/go-chi/chi/v5"
 "github.com/google/uuid"
)

type MoveHandler struct {
 uc     *UseCases
 logger *slog.Logger
}

func NewMoveHandler(uc *UseCases, logger *slog.Logger) *MoveHandler {
 return &MoveHandler{
  uc:     uc,
  logger: logger.With(slog.String("handler", "move")),
 }
}

// HandleCreateDraft معالجة إنشاء مسودة قيد
func (h *MoveHandler) HandleCreateDraft(w http.ResponseWriter, r *http.Request) {
 var req CreateMoveRequest
 if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
  respondError(w, http.StatusBadRequest, "invalid request payload")
  return
 }

 if err := req.Validate(); err != nil {
  respondError(w, http.StatusUnprocessableEntity, err.Error())
  return
 }

 resp, err := h.uc.CreateDraftMove.Execute(r.Context(), req)
 if err != nil {
  respondError(w, http.StatusInternalServerError, err.Error())
  return
 }

 respondJSON(w, http.StatusCreated, resp)
}

// HandlePost معالجة طلب الترحيل النهائي
func (h *MoveHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
 idStr := chi.URLParam(r, "id")
 moveID, err := uuid.Parse(idStr)
 if err != nil {
  respondError(w, http.StatusBadRequest, "invalid move id")
  return
 }

 // استخراج هوية المستخدم الحالي من سياق الطلب (Context)
 actorID := uuid.Nil // أو من auth context في النظام الفعلي

 resp, err := h.uc.PostMove.Execute(r.Context(), moveID, actorID)
 if err != nil {
  switch {
  case errors.Is(err, ErrUnbalancedMove):
   respondError(w, http.StatusUnprocessableEntity, err.Error())
  case errors.Is(err, ErrPeriodClosed):
   respondError(w, http.StatusConflict, err.Error())
  case errors.Is(err, ErrMoveAlreadyPosted):
   respondError(w, http.StatusConflict, err.Error())
  default:
   respondError(w, http.StatusInternalServerError, err.Error())
  }
  return
 }

 respondJSON(w, http.StatusOK, resp)
}

// HandleReverse معالجة طلب العكس (Reversal)
func (h *MoveHandler) HandleReverse(w http.ResponseWriter, r *http.Request) {
 idStr := chi.URLParam(r, "id")
 moveID, err := uuid.Parse(idStr)
 if err != nil {
  respondError(w, http.StatusBadRequest, "invalid move id")
  return
 }

 var req struct {
  Date   string `json:"date"`
  Reason string `json:"reason"`
 }
 _ = json.NewDecoder(r.Body).Decode(&req)

 revDate := time.Now().UTC()
 if req.Date != "" {
  if d, err := time.Parse("2006-01-02", req.Date); err == nil {
   revDate = d
  }
 }

 actorID := uuid.Nil
 resp, err := h.uc.ReverseMove.Execute(r.Context(), moveID, actorID, revDate, req.Reason)
 if err != nil {
  respondError(w, http.StatusBadRequest, err.Error())
  return
 }

 respondJSON(w, http.StatusOK, resp)
}

// HandleGet جلب قيد بواسطة المعرف
func (h *MoveHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
 idStr := chi.URLParam(r, "id")
 moveID, err := uuid.Parse(idStr)
 if err != nil {
  respondError(w, http.StatusBadRequest, "invalid move id")
  return
 }

 move, err := h.uc.CreateDraftMove.repo.GetMoveByID(r.Context(), moveID)
 if err != nil {
  respondError(w, http.StatusNotFound, "move not found")
  return
 }

 respondJSON(w, http.StatusOK, ToMoveResponse(move))
}
```

---

### الخطوة 3: معالج مسارات التقارير المالية (`report_handler.go`)

```go
package accounting

import (
 "log/slog"
 "net/http"
 "time"
)

type ReportHandler struct {
 uc     *UseCases
 logger *slog.Logger
}

func NewReportHandler(uc *UseCases, logger *slog.Logger) *ReportHandler {
 return &ReportHandler{
  uc:     uc,
  logger: logger.With(slog.String("handler", "report")),
 }
}

func (h *ReportHandler) HandleTrialBalance(w http.ResponseWriter, r *http.Request) {
 fromStr := r.URL.Query().Get("from")
 toStr := r.URL.Query().Get("to")

 from, err := time.Parse("2006-01-02", fromStr)
 if err != nil {
  from = time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
 }

 to, err := time.Parse("2006-01-02", toStr)
 if err != nil {
  to = time.Now().UTC()
 }

 resp, err := h.uc.GenerateTrialBalance.Execute(r.Context(), from, to)
 if err != nil {
  respondError(w, http.StatusInternalServerError, err.Error())
  return
 }

 respondJSON(w, http.StatusOK, resp)
}
```

---

## 3. التحقق ومعايير القبول (Acceptance Criteria)

- [x] التأكد من أن جميع المعالجات تُرجع كود `422 Unprocessable Entity` عند فشل التحقق في `Validate()`.
- [x] التأكد من معالجة أخطاء الفترات المغلقة وتضارب الترحيل المكرر بكود `409 Conflict`.
- [x] كتابة اختبارات `handler_test.go` تستخدم `httptest.NewServer` للتحقق من استجابات الـ JSON.

> **ملاحظة النطاق:** رُبطت المسارات براوتر التطبيق عند تشغيل PostgreSQL. المصادقة والتفويض غير مضمنين في هذه المرحلة، وسيتم تنفيذهما ضمن خطط مستقلة.
