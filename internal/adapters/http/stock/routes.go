package stockhttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all stock and inventory management endpoints onto the router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(r chi.Router, model string, action auth.Action) chi.Router {
		if authorizer == nil {
			return r
		}
		return r.With(auth.RequireAccess(authorizer, model, action))
	}
	r.Route("/warehouses", func(wr chi.Router) {
		access(wr, "stock.warehouse", auth.ActionCreate).Post("/", h.CreateWarehouse)
		access(wr, "stock.warehouse", auth.ActionRead).Get("/", h.ListWarehouses)
		access(wr, "stock.warehouse", auth.ActionRead).Get("/{id}", h.GetWarehouse)
		access(wr, "stock.warehouse", auth.ActionWrite).Put("/{id}", h.UpdateWarehouse)
		access(wr, "stock.warehouse", auth.ActionUnlink).Delete("/{id}", h.DeleteWarehouse)
	})

	r.Route("/stock-locations", func(lr chi.Router) {
		access(lr, "stock.location", auth.ActionCreate).Post("/", h.CreateLocation)
		access(lr, "stock.location", auth.ActionRead).Get("/", h.ListLocations)
		access(lr, "stock.location", auth.ActionRead).Get("/{id}", h.GetLocation)
		access(lr, "stock.location", auth.ActionWrite).Put("/{id}", h.UpdateLocation)
		access(lr, "stock.location", auth.ActionUnlink).Delete("/{id}", h.DeleteLocation)
	})

	r.Route("/stock-pickings", func(pr chi.Router) {
		access(pr, "stock.picking", auth.ActionCreate).Post("/", h.CreatePicking)
		access(pr, "stock.picking", auth.ActionRead).Get("/", h.ListPickings)
		access(pr, "stock.picking", auth.ActionRead).Get("/{id}", h.GetPicking)
		access(pr, "stock.picking", auth.ActionWrite).Put("/{id}", h.UpdatePicking)
		access(pr, "stock.picking", auth.ActionUnlink).Delete("/{id}", h.DeletePicking)
		access(pr, "stock.picking", auth.ActionWrite).Post("/{id}/confirm", h.ConfirmPicking)
		access(pr, "stock.picking", auth.ActionWrite).Post("/{id}/validate", h.ValidatePicking)
		access(pr, "stock.picking", auth.ActionWrite).Post("/{id}/cancel", h.CancelPicking)
	})

	r.Route("/stock", func(sr chi.Router) {
		access(sr, "stock.quant", auth.ActionRead).Get("/on-hand", h.GetOnHandStock)
		access(sr, "stock.move", auth.ActionRead).Get("/moves", h.ListMoves)
		access(sr, "stock.quant", auth.ActionWrite).Post("/adjust", h.AdjustStock)
		access(sr, "stock.move", auth.ActionWrite).Post("/moves/{id}/value", h.AdjustMoveValue)

		// Phase 12 — stock valuation
		access(sr, "stock.quant", auth.ActionRead).Get("/valuation/summaries", h.GetValuationSummaries)
		access(sr, "stock.move", auth.ActionRead).Get("/valuation/values", h.ListProductValues)
		access(sr, "stock.quant", auth.ActionRead).Get("/valuation/periods", h.ListAccountingPeriods)
		access(sr, "stock.quant", auth.ActionWrite).Post("/valuation/periods", h.CreateAccountingPeriod)
		access(sr, "stock.quant", auth.ActionRead).Get("/valuation/periods/{id}", h.GetAccountingPeriod)
		access(sr, "stock.quant", auth.ActionWrite).Post("/valuation/periods/{id}/close", h.ClosePeriodValuation)
	})

	r.Route("/reorder-rules", func(orr chi.Router) {
		access(orr, "stock.orderpoint", auth.ActionCreate).Post("/", h.CreateOrderpoint)
		access(orr, "stock.orderpoint", auth.ActionRead).Get("/", h.ListOrderpoints)
		access(orr, "stock.orderpoint", auth.ActionRead).Get("/suggestions", h.OrderpointSuggestions)
		access(orr, "stock.orderpoint", auth.ActionWrite).Post("/run", h.RunReplenishment)
		access(orr, "stock.orderpoint", auth.ActionRead).Get("/{id}", h.GetOrderpoint)
		access(orr, "stock.orderpoint", auth.ActionWrite).Put("/{id}", h.UpdateOrderpoint)
		access(orr, "stock.orderpoint", auth.ActionUnlink).Delete("/{id}", h.DeleteOrderpoint)
	})

	r.Route("/landed-costs", func(lcr chi.Router) {
		access(lcr, "stock.landed.cost", auth.ActionCreate).Post("/", h.CreateLandedCost)
		access(lcr, "stock.landed.cost", auth.ActionRead).Get("/", h.ListLandedCosts)
		access(lcr, "stock.landed.cost", auth.ActionRead).Get("/{id}", h.GetLandedCost)
		access(lcr, "stock.landed.cost", auth.ActionWrite).Put("/{id}", h.UpdateLandedCost)
		access(lcr, "stock.landed.cost", auth.ActionUnlink).Delete("/{id}", h.DeleteLandedCost)
		access(lcr, "stock.landed.cost", auth.ActionWrite).Post("/{id}/compute", h.ComputeLandedCost)
		access(lcr, "stock.landed.cost", auth.ActionWrite).Post("/{id}/validate", h.ValidateLandedCost)
		access(lcr, "stock.landed.cost", auth.ActionWrite).Post("/{id}/cancel", h.CancelLandedCost)
	})

	r.Route("/invoices", func(ir chi.Router) {
		access(ir, "stock.landed.cost", auth.ActionCreate).Post("/{id}/create-landed-cost", h.CreateLandedCostFromVendorBill)
	})
}
