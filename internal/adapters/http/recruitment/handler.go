package recruitmenthttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/recruitment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	recruitmentusecase "cashflow_backend/internal/usecase/recruitment"

	"github.com/go-chi/chi/v5"
)

type Handler struct{ useCase *recruitmentusecase.UseCase }

func NewHandler(useCase *recruitmentusecase.UseCase) *Handler { return &Handler{useCase: useCase} }
func decodeRecruitment(writer http.ResponseWriter, request *http.Request, target any) bool {
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return false
	}
	return true
}
func recruitmentID(writer http.ResponseWriter, request *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(request, "id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid recruitment ID", err))
		return 0, false
	}
	return id, true
}

func (handler *Handler) ListStages(writer http.ResponseWriter, request *http.Request) {
	companyID, err := strconv.ParseInt(request.URL.Query().Get("company_id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("company_id is required", err))
		return
	}
	result, err := handler.useCase.ListStages(request.Context(), companyID)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, result)
}
func (handler *Handler) ListApplicants(writer http.ResponseWriter, request *http.Request) {
	companyID, err := strconv.ParseInt(request.URL.Query().Get("company_id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("company_id is required", err))
		return
	}
	result, err := handler.useCase.ListApplicants(request.Context(), companyID)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, result)
}
func (handler *Handler) CreateApplicant(writer http.ResponseWriter, request *http.Request) {
	var applicant recruitment.Applicant
	if !decodeRecruitment(writer, request, &applicant) {
		return
	}
	created, err := handler.useCase.CreateApplicant(request.Context(), &applicant)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}
func (handler *Handler) MoveApplicant(writer http.ResponseWriter, request *http.Request) {
	id, ok := recruitmentID(writer, request)
	if !ok {
		return
	}
	stageID, err := strconv.ParseInt(request.URL.Query().Get("stage_id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("stage_id is required", err))
		return
	}
	applicant, err := handler.useCase.MoveApplicant(request.Context(), id, stageID)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, applicant)
}
func (handler *Handler) HireApplicant(writer http.ResponseWriter, request *http.Request) {
	id, ok := recruitmentID(writer, request)
	if !ok {
		return
	}
	employeeID, err := handler.useCase.Hire(request.Context(), id)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, map[string]int64{"employee_id": employeeID})
}
func (handler *Handler) CreateInterview(writer http.ResponseWriter, request *http.Request) {
	var interview recruitment.ApplicantInterview
	if !decodeRecruitment(writer, request, &interview) {
		return
	}
	created, err := handler.useCase.CreateInterview(request.Context(), &interview)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}
