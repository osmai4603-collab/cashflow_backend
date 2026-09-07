package userhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	userusecase "cashflow_backend/internal/usecase/user"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the User domain.
type Handler struct {
	useCase userusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(useCase userusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Create handles POST /api/v1/users
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreateUser(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToUserResponse(created))
}

// GetByID handles GET /api/v1/users/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid user ID in path", err))
		return
	}

	u, err := h.useCase.GetUser(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToUserResponse(u))
}

// Update handles PUT /api/v1/users/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid user ID in path", err))
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateUser(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToUserResponse(updated))
}

// Delete handles DELETE /api/v1/users/{id} (soft-delete)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid user ID in path", err))
		return
	}

	if err := h.useCase.DeleteUser(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// List handles GET /api/v1/users with filtering and pagination
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)
	f := parseUserFilterFromQuery(r)

	result, err := h.useCase.ListUsers(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToUserResponseList(result.Items), result)
}

// Login handles POST /api/v1/users/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	result, err := h.useCase.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	userResp := ToUserResponse(result.User)
	response.JSON(w, http.StatusOK, LoginResponse{
		Token: result.Token,
		User:  &userResp,
	})
}

func parseUserID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}

func parseUserFilterFromQuery(r *http.Request) *filter.Filter {
	q := r.URL.Query()
	f := filter.NewFilter()

	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}

	if login := q.Get("login"); login != "" {
		f.Add("login", filter.OpILike, login)
	}

	if email := q.Get("email"); email != "" {
		f.Add("email", filter.OpEqual, email)
	}

	if companyID := q.Get("company_id"); companyID != "" {
		if id, err := strconv.ParseInt(companyID, 10, 64); err == nil {
			f.Add("company_id", filter.OpEqual, id)
		}
	}

	if len(f.Criteria) == 0 {
		return nil
	}
	return f
}
