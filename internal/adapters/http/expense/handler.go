package expensehttp

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/expense"
	"cashflow_backend/internal/platform/auth"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	expenseusecase "cashflow_backend/internal/usecase/expense"

	"github.com/go-chi/chi/v5"
)

type Handler struct{ useCase *expenseusecase.UseCase }

func NewHandler(useCase *expenseusecase.UseCase) *Handler { return &Handler{useCase: useCase} }

type refuseRequest struct {
	Reason string `json:"reason"`
}
type postRequest struct {
	JournalID               int64  `json:"journal_id"`
	CompanyPaymentAccountID *int64 `json:"company_payment_account_id,omitempty"`
}
type splitRequest struct {
	Splits []expense.ExpenseSplitLine `json:"splits"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var value expense.Expense
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if err := h.useCase.CreateExpense(r.Context(), &value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := expense.Filter{}
	if value := r.URL.Query().Get("employee_id"); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			response.Error(w, platformerrors.BadRequest("invalid employee_id"))
			return
		}
		filter.EmployeeID = &id
	}
	if value := r.URL.Query().Get("company_id"); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			response.Error(w, platformerrors.BadRequest("invalid company_id"))
			return
		}
		filter.CompanyID = &id
	}
	if value := r.URL.Query().Get("state"); value != "" {
		state := expense.ExpenseState(value)
		filter.State = &state
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		filter.Limit, _ = strconv.Atoi(value)
	}
	if value := r.URL.Query().Get("offset"); value != "" {
		filter.Offset, _ = strconv.Atoi(value)
	}
	values, err := h.useCase.ListExpenses(r.Context(), filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetExpense(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var value expense.Expense
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	value.ID = id
	if err := h.useCase.UpdateExpense(r.Context(), &value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteExpense(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.useCase.SubmitExpense)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil || claims.UserID <= 0 {
		response.Error(w, platformerrors.Unauthorized("authenticated user is required"))
		return
	}
	if err := h.useCase.ApproveExpense(r.Context(), id, claims.UserID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *Handler) Refuse(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil || claims.UserID <= 0 {
		response.Error(w, platformerrors.Unauthorized("authenticated user is required"))
		return
	}
	var request refuseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if err := h.useCase.RefuseExpense(r.Context(), id, claims.UserID, request.Reason); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "refused"})
}

func (h *Handler) Split(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request splitRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	ids, err := h.useCase.SplitExpense(r.Context(), expense.ExpenseSplitRequest{ExpenseID: id, Splits: request.Splits})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"expense_ids": ids})
}

func (h *Handler) Post(w http.ResponseWriter, r *http.Request) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request postRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if err := h.useCase.PostExpenseWithAccounting(r.Context(), id, expenseusecase.PostExpenseInput{JournalID: request.JournalID, CompanyPaymentAccountID: request.CompanyPaymentAccountID}); err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetExpense(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request, action func(context.Context, int64) error) {
	id, err := expenseID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := action(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func expenseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, platformerrors.BadRequest("invalid expense ID")
	}
	return id, nil
}
