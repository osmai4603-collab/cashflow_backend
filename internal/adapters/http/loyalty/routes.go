package loyaltyhttp

import (
	"github.com/go-chi/chi/v5"
)

// Routes builds the /api/v1/loyalty sub-router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/programs", func(r chi.Router) {
		r.Post("/", h.CreateProgram)
		r.Get("/", h.ListPrograms)
		r.Get("/{id}", h.GetProgram)
		r.Put("/{id}", h.UpdateProgram)
		r.Put("/{id}/type", h.SetProgramType)
		r.Delete("/{id}", h.DeleteProgram)

		r.Post("/{id}/rules", h.CreateRule)
		r.Post("/{id}/rewards", h.CreateReward)
		r.Post("/{id}/mails", h.CreateMail)
	})

	r.Route("/rules", func(r chi.Router) {
		r.Put("/{id}", h.UpdateRule)
		r.Delete("/{id}", h.DeleteRule)
	})

	r.Route("/rewards", func(r chi.Router) {
		r.Put("/{id}", h.UpdateReward)
		r.Delete("/{id}", h.DeleteReward)
	})

	r.Route("/mails", func(r chi.Router) {
		r.Put("/{id}", h.UpdateMail)
		r.Delete("/{id}", h.DeleteMail)
	})

	r.Route("/cards", func(r chi.Router) {
		r.Post("/generate", h.GenerateCards)
		r.Get("/", h.ListCards)
		r.Get("/{id}", h.GetCard)
		r.Get("/{id}/history", h.CardHistory)
		r.Post("/{id}/archive", h.ArchiveCard)
	})

	r.Get("/check/{code}", h.CheckCoupon)

	r.Route("/orders/{id}", func(r chi.Router) {
		r.Get("/preview", h.PreviewOrder)
		r.Post("/earn", h.EarnCoupons)
		r.Post("/claim", h.ClaimCoupon)
		r.Post("/apply-code", h.ApplyCode)
		r.Post("/redeem", h.RedeemCoupon)
		r.Delete("/coupons/{coupon_id}", h.RemoveCoupon)
	})

	return r
}