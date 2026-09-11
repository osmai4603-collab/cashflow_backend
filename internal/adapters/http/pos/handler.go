package poshttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/pos"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	posusecase "cashflow_backend/internal/usecase/pos"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *posusecase.UseCase
}

func NewHandler(useCase *posusecase.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

type openSessionRequest struct {
	ConfigID       int64   `json:"config_id"`
	UserID         int64   `json:"user_id"`
	CompanyID      int64   `json:"company_id"`
	OpeningBalance float64 `json:"opening_balance"`
}

type closeSessionRequest struct {
	ExpectedCash float64 `json:"expected_cash"`
	CountedCash  float64 `json:"counted_cash"`
}

func (handler *Handler) CreateConfig(writer http.ResponseWriter, request *http.Request) {
	var config pos.PosConfig
	if err := json.NewDecoder(request.Body).Decode(&config); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	created, err := handler.useCase.CreateConfig(request.Context(), &config)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}

func (handler *Handler) OpenSession(writer http.ResponseWriter, request *http.Request) {
	var input openSessionRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	session, err := handler.useCase.OpenSession(request.Context(), input.ConfigID, input.UserID, input.CompanyID, input.OpeningBalance)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, session)
}

func (handler *Handler) CloseSession(writer http.ResponseWriter, request *http.Request) {
	sessionID, err := strconv.ParseInt(chi.URLParam(request, "id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid session ID", err))
		return
	}
	var input closeSessionRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	session, err := handler.useCase.CloseSession(request.Context(), sessionID, input.ExpectedCash, input.CountedCash)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, session)
}

func (handler *Handler) CreateOrder(writer http.ResponseWriter, request *http.Request) {
	var order pos.PosOrder
	if err := json.NewDecoder(request.Body).Decode(&order); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	created, err := handler.useCase.CreateOrder(request.Context(), &order)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}

func (handler *Handler) SyncOrders(writer http.ResponseWriter, request *http.Request) {
	var batch pos.SyncBatch
	if err := json.NewDecoder(request.Body).Decode(&batch); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	result, err := handler.useCase.SyncOrders(request.Context(), &batch)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, result)
}
