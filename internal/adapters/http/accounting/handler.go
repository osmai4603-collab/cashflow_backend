package accountinghttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/accounting"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	accountingusecase "cashflow_backend/internal/usecase/accounting"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Core Accounting domain.
type Handler struct {
	useCase *accountingusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs an accounting Handler.
func NewHandler(useCase *accountingusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func parseID(param string) (int64, error) {
	return strconv.ParseInt(param, 10, 64)
}

func parseDateParam(val string) time.Time {
	if val == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", val)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	acc, err := h.useCase.CreateAccount(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToAccountResponse(acc))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}

	acc, err := h.useCase.GetAccount(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAccountResponse(acc))
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}

	var req UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateAccount(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAccountResponse(updated))
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}

	if err := h.useCase.DeleteAccount(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	result, err := h.useCase.ListAccounts(r.Context(), nil, page)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]AccountResponse, len(result.Items))
	for i, a := range result.Items {
		items[i] = ToAccountResponse(&a)
	}

	response.Paginated(w, http.StatusOK, items, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// Journals Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateJournal(w http.ResponseWriter, r *http.Request) {
	var req CreateJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	j, err := h.useCase.CreateJournal(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToJournalResponse(j))
}

func (h *Handler) GetJournal(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid journal ID in path", err))
		return
	}

	j, err := h.useCase.GetJournal(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToJournalResponse(j))
}

func (h *Handler) UpdateJournal(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid journal ID in path", err))
		return
	}

	var req UpdateJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateJournal(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToJournalResponse(updated))
}

func (h *Handler) DeleteJournal(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid journal ID in path", err))
		return
	}

	if err := h.useCase.DeleteJournal(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListJournals(w http.ResponseWriter, r *http.Request) {
	journals, err := h.useCase.ListJournals(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]JournalResponse, len(journals))
	for i, j := range journals {
		items[i] = ToJournalResponse(&j)
	}

	response.JSON(w, http.StatusOK, items)
}

// ─────────────────────────────────────────────────────────────────────────────
// Taxes Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateTax(w http.ResponseWriter, r *http.Request) {
	var req CreateTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	t, err := h.useCase.CreateTax(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToTaxResponse(t))
}

func (h *Handler) GetTax(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid tax ID in path", err))
		return
	}

	t, err := h.useCase.GetTax(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToTaxResponse(t))
}

func (h *Handler) UpdateTax(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid tax ID in path", err))
		return
	}

	var req UpdateTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateTax(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToTaxResponse(updated))
}

func (h *Handler) DeleteTax(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid tax ID in path", err))
		return
	}

	if err := h.useCase.DeleteTax(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListTaxes(w http.ResponseWriter, r *http.Request) {
	var scope *accounting.TaxScope
	if s := r.URL.Query().Get("type_tax_use"); s != "" {
		sc := accounting.TaxScope(s)
		scope = &sc
	}

	taxes, err := h.useCase.ListTaxes(r.Context(), scope)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]TaxResponse, len(taxes))
	for i, t := range taxes {
		items[i] = ToTaxResponse(&t)
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) ComputeTax(w http.ResponseWriter, r *http.Request) {
	var req ComputeTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	res, err := h.useCase.ComputeTax(r.Context(), req.TaxID, req.Amount)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Payment Terms Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreatePaymentTerm(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	pt, err := h.useCase.CreatePaymentTerm(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToPaymentTermResponse(pt))
}

func (h *Handler) GetPaymentTerm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid payment term ID in path", err))
		return
	}

	pt, err := h.useCase.GetPaymentTerm(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentTermResponse(pt))
}

func (h *Handler) UpdatePaymentTerm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid payment term ID in path", err))
		return
	}

	var req UpdatePaymentTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdatePaymentTerm(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentTermResponse(updated))
}

func (h *Handler) DeletePaymentTerm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid payment term ID in path", err))
		return
	}

	if err := h.useCase.DeletePaymentTerm(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListPaymentTerms(w http.ResponseWriter, r *http.Request) {
	terms, err := h.useCase.ListPaymentTerms(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]PaymentTermResponse, len(terms))
	for i, pt := range terms {
		items[i] = ToPaymentTermResponse(&pt)
	}

	response.JSON(w, http.StatusOK, items)
}

// ─────────────────────────────────────────────────────────────────────────────
// Moves & Invoices Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateJournalEntry(w http.ResponseWriter, r *http.Request) {
	var req CreateJournalEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	move, err := h.useCase.CreateJournalEntry(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToMoveResponse(move))
}

func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	invoice, err := h.useCase.CreateInvoice(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToMoveResponse(invoice))
}

func (h *Handler) GetMove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid move ID in path", err))
		return
	}

	move, err := h.useCase.GetMove(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToMoveResponse(move))
}

func (h *Handler) UpdateMove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid move ID in path", err))
		return
	}

	var req UpdateMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateMove(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToMoveResponse(updated))
}

func (h *Handler) DeleteMove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid move ID in path", err))
		return
	}

	if err := h.useCase.DeleteMove(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) PostMove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid move ID in path", err))
		return
	}

	posted, err := h.useCase.PostMove(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToMoveResponse(posted))
}

func (h *Handler) CancelMove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid move ID in path", err))
		return
	}

	cancelled, err := h.useCase.CancelMove(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToMoveResponse(cancelled))
}

func (h *Handler) ReverseMove(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid move ID in path", err))
		return
	}

	var req ReverseMoveRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	reversed, err := h.useCase.ReverseMove(r.Context(), id, req.ReversalDate, req.Ref)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToMoveResponse(reversed))
}

func (h *Handler) ListMoves(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	result, err := h.useCase.ListMoves(r.Context(), nil, page)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]MoveResponse, len(result.Items))
	for i, m := range result.Items {
		items[i] = ToMoveResponse(&m)
	}

	response.Paginated(w, http.StatusOK, items, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// Financial Reports Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetTrialBalance(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromDate := parseDateParam(q.Get("from_date"))
	toDate := parseDateParam(q.Get("to_date"))
	onlyPosted := q.Get("target_move") != "all"

	report, err := h.useCase.GetTrialBalance(r.Context(), fromDate, toDate, onlyPosted)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, report)
}

func (h *Handler) GetProfitAndLoss(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromDate := parseDateParam(q.Get("from_date"))
	toDate := parseDateParam(q.Get("to_date"))

	report, err := h.useCase.GetProfitAndLoss(r.Context(), fromDate, toDate)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, report)
}

func (h *Handler) GetBalanceSheet(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	asOfDate := parseDateParam(q.Get("as_of_date"))

	report, err := h.useCase.GetBalanceSheet(r.Context(), asOfDate)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, report)
}

func (h *Handler) GetGeneralLedger(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var accID, partnerID *int64
	if aStr := q.Get("account_id"); aStr != "" {
		if id, err := parseID(aStr); err == nil {
			accID = &id
		}
	}
	if pStr := q.Get("partner_id"); pStr != "" {
		if id, err := parseID(pStr); err == nil {
			partnerID = &id
		}
	}

	var fromDate, toDate *time.Time
	if fStr := q.Get("from_date"); fStr != "" {
		d := parseDateParam(fStr)
		if !d.IsZero() {
			fromDate = &d
		}
	}
	if tStr := q.Get("to_date"); tStr != "" {
		d := parseDateParam(tStr)
		if !d.IsZero() {
			toDate = &d
		}
	}

	items, err := h.useCase.GetGeneralLedger(r.Context(), accID, partnerID, fromDate, toDate)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}
