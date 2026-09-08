package maintenancehttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, handler *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(router chi.Router, model string, action auth.Action) chi.Router {
		if authorizer == nil {
			return router
		}
		return router.With(auth.RequireAccess(authorizer, model, action))
	}

	r.Route("/maintenance/equipment-categories", func(categories chi.Router) {
		access(categories, "maintenance.equipment.category", auth.ActionCreate).Post("/", handler.CreateCategory)
		access(categories, "maintenance.equipment.category", auth.ActionRead).Get("/", handler.ListCategories)
		access(categories, "maintenance.equipment.category", auth.ActionRead).Get("/{id}", handler.GetCategory)
		access(categories, "maintenance.equipment.category", auth.ActionWrite).Put("/{id}", handler.UpdateCategory)
		access(categories, "maintenance.equipment.category", auth.ActionUnlink).Delete("/{id}", handler.DeleteCategory)
	})
	r.Route("/maintenance/stages", func(stages chi.Router) {
		access(stages, "maintenance.stage", auth.ActionCreate).Post("/", handler.CreateStage)
		access(stages, "maintenance.stage", auth.ActionRead).Get("/", handler.ListStages)
		access(stages, "maintenance.stage", auth.ActionRead).Get("/{id}", handler.GetStage)
		access(stages, "maintenance.stage", auth.ActionWrite).Put("/{id}", handler.UpdateStage)
		access(stages, "maintenance.stage", auth.ActionUnlink).Delete("/{id}", handler.DeleteStage)
	})
	r.Route("/maintenance/teams", func(teams chi.Router) {
		access(teams, "maintenance.team", auth.ActionCreate).Post("/", handler.CreateTeam)
		access(teams, "maintenance.team", auth.ActionRead).Get("/", handler.ListTeams)
		access(teams, "maintenance.team", auth.ActionRead).Get("/{id}", handler.GetTeam)
		access(teams, "maintenance.team", auth.ActionWrite).Put("/{id}", handler.UpdateTeam)
		access(teams, "maintenance.team", auth.ActionUnlink).Delete("/{id}", handler.DeleteTeam)
	})
	r.Route("/maintenance/equipments", func(equipments chi.Router) {
		access(equipments, "maintenance.equipment", auth.ActionCreate).Post("/", handler.CreateEquipment)
		access(equipments, "maintenance.equipment", auth.ActionRead).Get("/", handler.ListEquipments)
		access(equipments, "maintenance.equipment", auth.ActionRead).Get("/{id}", handler.GetEquipment)
		access(equipments, "maintenance.equipment", auth.ActionWrite).Put("/{id}", handler.UpdateEquipment)
		access(equipments, "maintenance.equipment", auth.ActionUnlink).Delete("/{id}", handler.DeleteEquipment)
	})
	r.Route("/maintenance/requests", func(requests chi.Router) {
		access(requests, "maintenance.request", auth.ActionCreate).Post("/", handler.CreateRequest)
		access(requests, "maintenance.request", auth.ActionRead).Get("/", handler.ListRequests)
		access(requests, "maintenance.request", auth.ActionRead).Get("/{id}", handler.GetRequest)
		access(requests, "maintenance.request", auth.ActionWrite).Put("/{id}", handler.UpdateRequest)
		access(requests, "maintenance.request", auth.ActionUnlink).Delete("/{id}", handler.DeleteRequest)
		access(requests, "maintenance.request", auth.ActionWrite).Post("/{id}/close", handler.CloseRequest)
		access(requests, "maintenance.request", auth.ActionWrite).Post("/{id}/archive", handler.ArchiveRequest)
	})
	r.Route("/maintenance/dashboard", func(dashboard chi.Router) {
		access(dashboard, "maintenance.request", auth.ActionRead).Get("/", handler.Dashboard)
	})
}