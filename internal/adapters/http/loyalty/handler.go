package loyaltyhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/loyalty"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/response"
	"cashflow_backend/internal/platform/pagination"
	loyaltyusecase "cashflow_backend/internal/usecase/loyalty"

	"github.com/go-chi/chi/v5"
)

// Handler exposes the loyalty & rewards API.
type Handler struct {
	usecase *loyaltyusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(usecase *loyaltyusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{usecase: usecase, logger: logger}
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// ─────────────────────────────────────────────────────────────────────────────
// Programs
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateProgram(w http.ResponseWriter, r *http.Request) {
	var req CreateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	program, err := h.usecase.CreateProgram(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, program)
}

func (h *Handler) GetProgram(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	program, err := h.usecase.GetProgram(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, program)
}

func (h *Handler) ListPrograms(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	q := r.URL.Query()
	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}
	if pType := q.Get("program_type"); pType != "" {
		f.Add("program_type", filter.OpEqual, pType)
	}
	if appliesOn := q.Get("applies_on"); appliesOn != "" {
		f.Add("applies_on", filter.OpEqual, appliesOn)
	}
	if active := q.Get("active"); active != "" {
		if b, err := strconv.ParseBool(active); err == nil {
			f.Add("active", filter.OpEqual, b)
		}
	}

	result, err := h.usecase.ListPrograms(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) UpdateProgram(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	var req UpdateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	program, err := h.usecase.UpdateProgram(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, program)
}

func (h *Handler) SetProgramType(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	var req SetProgramTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	program, err := h.usecase.SetProgramType(r.Context(), id, req.ProgramType)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, program)
}

func (h *Handler) DeleteProgram(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	if err := h.usecase.DeleteProgram(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// Child records: rules, rewards, mails
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	programID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	var rule loyalty.LoyaltyRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	created, err := h.usecase.CreateRule(r.Context(), programID, &rule)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid rule ID", err))
		return
	}
	var rule loyalty.LoyaltyRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	updated, err := h.usecase.UpdateRule(r.Context(), id, &rule)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid rule ID", err))
		return
	}
	if err := h.usecase.DeleteRule(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateReward(w http.ResponseWriter, r *http.Request) {
	programID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	var reward loyalty.LoyaltyReward
	if err := json.NewDecoder(r.Body).Decode(&reward); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	created, err := h.usecase.CreateReward(r.Context(), programID, &reward)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) UpdateReward(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid reward ID", err))
		return
	}
	var reward loyalty.LoyaltyReward
	if err := json.NewDecoder(r.Body).Decode(&reward); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	updated, err := h.usecase.UpdateReward(r.Context(), id, &reward)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteReward(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid reward ID", err))
		return
	}
	if err := h.usecase.DeleteReward(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateMail(w http.ResponseWriter, r *http.Request) {
	programID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid program ID", err))
		return
	}
	var mail loyalty.LoyaltyMail
	if err := json.NewDecoder(r.Body).Decode(&mail); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	created, err := h.usecase.CreateMail(r.Context(), programID, &mail)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) UpdateMail(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid mail ID", err))
		return
	}
	var mail loyalty.LoyaltyMail
	if err := json.NewDecoder(r.Body).Decode(&mail); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	updated, err := h.usecase.UpdateMail(r.Context(), id, &mail)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteMail(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid mail ID", err))
		return
	}
	if err := h.usecase.DeleteMail(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cards
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GenerateCards(w http.ResponseWriter, r *http.Request) {
	var req GenerateCardsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	cards, err := h.usecase.GenerateCards(r.Context(), loyaltyusecase.GenerateCardsInput{
		ProgramID:      req.ProgramID,
		PartnerID:      req.PartnerID,
		ExpirationDate: req.ExpirationDate,
		Count:          req.Count,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, cards)
}

func (h *Handler) GetCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid card ID", err))
		return
	}
	card, err := h.usecase.GetCard(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, card)
}

func (h *Handler) ListCards(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	q := r.URL.Query()
	if code := q.Get("code"); code != "" {
		f.Add("code", filter.OpEqual, code)
	}
	if programID := q.Get("program_id"); programID != "" {
		f.Add("program_id", filter.OpEqual, programID)
	}
	if partnerID := q.Get("partner_id"); partnerID != "" {
		f.Add("partner_id", filter.OpEqual, partnerID)
	}
	if active := q.Get("active"); active != "" {
		if b, err := strconv.ParseBool(active); err == nil {
			f.Add("active", filter.OpEqual, b)
		}
	}

	result, err := h.usecase.ListCards(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) ArchiveCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid card ID", err))
		return
	}
	card, err := h.usecase.ArchiveCard(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, card)
}

func (h *Handler) CardHistory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid card ID", err))
		return
	}
	history, err := h.usecase.ListCardHistory(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, history)
}

func (h *Handler) CheckCoupon(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		response.Error(w, platformerrors.BadRequest("coupon code is required", nil))
		return
	}
	info, err := h.usecase.CheckCoupon(r.Context(), code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, info)
}

// ─────────────────────────────────────────────────────────────────────────────
// Order operations
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) PreviewOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid order ID", err))
		return
	}
	preview, err := h.usecase.PreviewOrder(r.Context(), orderID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, preview)
}

func (h *Handler) EarnCoupons(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid order ID", err))
		return
	}
	preview, err := h.usecase.EarnCoupons(r.Context(), orderID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, preview)
}

func (h *Handler) ClaimCoupon(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid order ID", err))
		return
	}
	var req ClaimCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	preview, err := h.usecase.ClaimCoupon(r.Context(), orderID, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, preview)
}

func (h *Handler) ApplyCode(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid order ID", err))
		return
	}
	var req ApplyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	preview, err := h.usecase.ApplyCode(r.Context(), orderID, req.RuleID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, preview)
}

func (h *Handler) RedeemCoupon(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid order ID", err))
		return
	}
	var req RedeemCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}
	preview, err := h.usecase.RedeemCoupon(r.Context(), orderID, loyaltyusecase.RedeemInput{
		Code:     req.Code,
		RewardID: req.RewardID,
		Points:   req.Points,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, preview)
}

func (h *Handler) RemoveCoupon(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid order ID", err))
		return
	}
	couponID, err := strconv.ParseInt(chi.URLParam(r, "coupon_id"), 10, 64)
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid coupon ID", err))
		return
	}
	preview, err := h.usecase.RemoveCoupon(r.Context(), orderID, couponID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, preview)
}