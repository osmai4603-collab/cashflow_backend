package producthttp

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
	productusecase "cashflow_backend/internal/usecase/product"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Product & Catalog domain.
type Handler struct {
	useCase productusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a Handler.
func NewHandler(useCase productusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Products & Templates Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreateProduct(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToProductResponse(created))
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid product ID in path", err))
		return
	}

	pt, err := h.useCase.GetProduct(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToProductResponse(pt))
}

func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid product ID in path", err))
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateProduct(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToProductResponse(updated))
}

func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid product ID in path", err))
		return
	}

	if err := h.useCase.DeleteProduct(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)
	f := parseProductFilterFromQuery(r)

	result, err := h.useCase.ListProducts(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToProductResponseList(result.Items), result)
}

// ─────────────────────────────────────────────────────────────────────────────
// Variant Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateProductVariant(w http.ResponseWriter, r *http.Request) {
	tmplID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid product template ID in path", err))
		return
	}

	var req CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	variant, err := h.useCase.CreateProductVariant(r.Context(), tmplID, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToVariantResponse(variant))
}

func (h *Handler) GetProductVariants(w http.ResponseWriter, r *http.Request) {
	tmplID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid product template ID in path", err))
		return
	}

	variants, err := h.useCase.GetProductVariants(r.Context(), tmplID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToVariantResponseList(variants))
}

func (h *Handler) GetVariant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid variant ID in path", err))
		return
	}

	variant, err := h.useCase.GetVariant(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToVariantResponse(variant))
}

func (h *Handler) DeleteVariant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid variant ID in path", err))
		return
	}

	if err := h.useCase.DeleteVariant(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// Category Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreateCategory(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToCategoryResponse(created))
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid category ID in path", err))
		return
	}

	cat, err := h.useCase.GetCategory(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCategoryResponse(cat))
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid category ID in path", err))
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateCategory(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCategoryResponse(updated))
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid category ID in path", err))
		return
	}

	if err := h.useCase.DeleteCategory(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.useCase.ListCategories(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCategoryResponseList(cats))
}

// ─────────────────────────────────────────────────────────────────────────────
// Units of Measure (UoM) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateUoM(w http.ResponseWriter, r *http.Request) {
	var req CreateUoMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	uom, err := h.useCase.CreateUoM(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToUoMResponse(uom))
}

func (h *Handler) GetUoM(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid unit of measure ID in path", err))
		return
	}

	uom, err := h.useCase.GetUoM(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToUoMResponse(uom))
}

func (h *Handler) UpdateUoM(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid unit of measure ID in path", err))
		return
	}

	var req UpdateUoMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateUoM(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToUoMResponse(updated))
}

func (h *Handler) DeleteUoM(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid unit of measure ID in path", err))
		return
	}

	if err := h.useCase.DeleteUoM(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListUoMs(w http.ResponseWriter, r *http.Request) {
	uoms, err := h.useCase.ListUoMs(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToUoMResponseList(uoms))
}

// ─────────────────────────────────────────────────────────────────────────────
// Pricelist Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreatePricelist(w http.ResponseWriter, r *http.Request) {
	var req CreatePricelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreatePricelist(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPricelistResponse(created))
}

func (h *Handler) GetPricelist(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid pricelist ID in path", err))
		return
	}

	pl, err := h.useCase.GetPricelist(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPricelistResponse(pl))
}

func (h *Handler) UpdatePricelist(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid pricelist ID in path", err))
		return
	}

	var req UpdatePricelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdatePricelist(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPricelistResponse(updated))
}

func (h *Handler) DeletePricelist(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid pricelist ID in path", err))
		return
	}

	if err := h.useCase.DeletePricelist(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListPricelists(w http.ResponseWriter, r *http.Request) {
	pricelists, err := h.useCase.ListPricelists(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPricelistResponseList(pricelists))
}

func (h *Handler) AddPricelistItem(w http.ResponseWriter, r *http.Request) {
	pricelistID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid pricelist ID in path", err))
		return
	}

	var req CreatePricelistItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	item, err := h.useCase.AddPricelistItem(r.Context(), pricelistID, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPricelistItemResponse(item))
}

func (h *Handler) DeletePricelistItem(w http.ResponseWriter, r *http.Request) {
	itemID, err := parseID(chi.URLParam(r, "itemId"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid pricelist item ID in path", err))
		return
	}

	if err := h.useCase.DeletePricelistItem(r.Context(), itemID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ComputePrice(w http.ResponseWriter, r *http.Request) {
	pricelistID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid pricelist ID in path", err))
		return
	}

	var req ComputePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	unitPrice, err := h.useCase.ComputePrice(r.Context(), pricelistID, req.ProductID, req.VariantID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}

	qty := req.Quantity
	if qty <= 0 {
		qty = 1.0
	}

	response.JSON(w, http.StatusOK, ComputePriceResponse{
		PricelistID: pricelistID,
		ProductID:   req.ProductID,
		VariantID:   req.VariantID,
		Quantity:    qty,
		UnitPrice:   unitPrice,
		TotalPrice:  unitPrice * qty,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func parseID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}

func parseProductFilterFromQuery(r *http.Request) *filter.Filter {
	q := r.URL.Query()
	f := filter.NewFilter()

	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}

	if pType := q.Get("type"); pType != "" {
		f.Add("type", filter.OpEqual, pType)
	}

	if ref := q.Get("internal_ref"); ref != "" {
		f.Add("internal_ref", filter.OpEqual, ref)
	}

	if barcode := q.Get("barcode"); barcode != "" {
		f.Add("barcode", filter.OpEqual, barcode)
	}

	if catStr := q.Get("category_id"); catStr != "" {
		if catID, err := strconv.ParseInt(catStr, 10, 64); err == nil {
			f.Add("category_id", filter.OpEqual, catID)
		}
	}

	if saleOK := q.Get("sale_ok"); saleOK != "" {
		if b, err := strconv.ParseBool(saleOK); err == nil {
			f.Add("sale_ok", filter.OpEqual, b)
		}
	}

	if len(f.Criteria) == 0 {
		return nil
	}
	return f
}
