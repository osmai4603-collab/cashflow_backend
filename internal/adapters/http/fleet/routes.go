package fleethttp

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

	r.Route("/fleet/brands", func(brands chi.Router) {
		access(brands, "fleet.vehicle.model.brand", auth.ActionCreate).Post("/", handler.CreateBrand)
		access(brands, "fleet.vehicle.model.brand", auth.ActionRead).Get("/", handler.ListBrands)
		access(brands, "fleet.vehicle.model.brand", auth.ActionRead).Get("/{id}", handler.GetBrand)
		access(brands, "fleet.vehicle.model.brand", auth.ActionWrite).Put("/{id}", handler.UpdateBrand)
		access(brands, "fleet.vehicle.model.brand", auth.ActionUnlink).Delete("/{id}", handler.DeleteBrand)
	})
	r.Route("/fleet/model-categories", func(categories chi.Router) {
		access(categories, "fleet.vehicle.model.category", auth.ActionCreate).Post("/", handler.CreateModelCategory)
		access(categories, "fleet.vehicle.model.category", auth.ActionRead).Get("/", handler.ListModelCategories)
		access(categories, "fleet.vehicle.model.category", auth.ActionRead).Get("/{id}", handler.GetModelCategory)
		access(categories, "fleet.vehicle.model.category", auth.ActionWrite).Put("/{id}", handler.UpdateModelCategory)
		access(categories, "fleet.vehicle.model.category", auth.ActionUnlink).Delete("/{id}", handler.DeleteModelCategory)
	})
	r.Route("/fleet/models", func(models chi.Router) {
		access(models, "fleet.vehicle.model", auth.ActionCreate).Post("/", handler.CreateModel)
		access(models, "fleet.vehicle.model", auth.ActionRead).Get("/", handler.ListModels)
		access(models, "fleet.vehicle.model", auth.ActionRead).Get("/{id}", handler.GetModel)
		access(models, "fleet.vehicle.model", auth.ActionWrite).Put("/{id}", handler.UpdateModel)
		access(models, "fleet.vehicle.model", auth.ActionUnlink).Delete("/{id}", handler.DeleteModel)
	})
	r.Route("/fleet/tags", func(tags chi.Router) {
		access(tags, "fleet.vehicle.tag", auth.ActionCreate).Post("/", handler.CreateTag)
		access(tags, "fleet.vehicle.tag", auth.ActionRead).Get("/", handler.ListTags)
		access(tags, "fleet.vehicle.tag", auth.ActionRead).Get("/{id}", handler.GetTag)
		access(tags, "fleet.vehicle.tag", auth.ActionWrite).Put("/{id}", handler.UpdateTag)
		access(tags, "fleet.vehicle.tag", auth.ActionUnlink).Delete("/{id}", handler.DeleteTag)
	})
	r.Route("/fleet/service-types", func(serviceTypes chi.Router) {
		access(serviceTypes, "fleet.service.type", auth.ActionCreate).Post("/", handler.CreateServiceType)
		access(serviceTypes, "fleet.service.type", auth.ActionRead).Get("/", handler.ListServiceTypes)
		access(serviceTypes, "fleet.service.type", auth.ActionRead).Get("/{id}", handler.GetServiceType)
		access(serviceTypes, "fleet.service.type", auth.ActionWrite).Put("/{id}", handler.UpdateServiceType)
		access(serviceTypes, "fleet.service.type", auth.ActionUnlink).Delete("/{id}", handler.DeleteServiceType)
	})
	r.Route("/fleet/vehicle-states", func(states chi.Router) {
		access(states, "fleet.vehicle.state", auth.ActionCreate).Post("/", handler.CreateState)
		access(states, "fleet.vehicle.state", auth.ActionRead).Get("/", handler.ListStates)
		access(states, "fleet.vehicle.state", auth.ActionRead).Get("/{id}", handler.GetState)
		access(states, "fleet.vehicle.state", auth.ActionWrite).Put("/{id}", handler.UpdateState)
		access(states, "fleet.vehicle.state", auth.ActionUnlink).Delete("/{id}", handler.DeleteState)
	})

	r.Route("/fleet/vehicles", func(vehicles chi.Router) {
		access(vehicles, "fleet.vehicle", auth.ActionCreate).Post("/", handler.CreateVehicle)
		access(vehicles, "fleet.vehicle", auth.ActionRead).Get("/", handler.ListVehicles)
		access(vehicles, "fleet.vehicle", auth.ActionRead).Get("/{id}", handler.GetVehicle)
		access(vehicles, "fleet.vehicle", auth.ActionWrite).Put("/{id}", handler.UpdateVehicle)
		access(vehicles, "fleet.vehicle", auth.ActionUnlink).Delete("/{id}", handler.DeleteVehicle)

		access(vehicles, "fleet.vehicle", auth.ActionCreate).Post("/{id}/assignations", handler.CreateAssignation)
		access(vehicles, "fleet.vehicle", auth.ActionRead).Get("/{id}/assignations", handler.ListAssignations)
		access(vehicles, "fleet.vehicle", auth.ActionWrite).Put("/{id}/assignations/{log_id}", handler.UpdateAssignation)
		access(vehicles, "fleet.vehicle", auth.ActionUnlink).Delete("/{id}/assignations/{log_id}", handler.DeleteAssignation)

		access(vehicles, "fleet.vehicle.odometer", auth.ActionCreate).Post("/{id}/odometers", handler.CreateOdometer)
		access(vehicles, "fleet.vehicle.odometer", auth.ActionRead).Get("/{id}/odometers", handler.ListOdometers)
		access(vehicles, "fleet.vehicle.odometer", auth.ActionWrite).Put("/{id}/odometers/{odometer_id}", handler.UpdateOdometer)
		access(vehicles, "fleet.vehicle.odometer", auth.ActionUnlink).Delete("/{id}/odometers/{odometer_id}", handler.DeleteOdometer)

		access(vehicles, "fleet.vehicle.log.services", auth.ActionCreate).Post("/{id}/services", handler.CreateService)
		access(vehicles, "fleet.vehicle.log.services", auth.ActionRead).Get("/{id}/services", handler.ListServices)
		access(vehicles, "fleet.vehicle.log.services", auth.ActionWrite).Put("/{id}/services/{service_id}", handler.UpdateService)
		access(vehicles, "fleet.vehicle.log.services", auth.ActionUnlink).Delete("/{id}/services/{service_id}", handler.DeleteService)

		access(vehicles, "fleet.vehicle.log.contract", auth.ActionCreate).Post("/{id}/contracts", handler.CreateContract)
		access(vehicles, "fleet.vehicle.log.contract", auth.ActionRead).Get("/{id}/contracts", handler.ListContracts)
		access(vehicles, "fleet.vehicle.log.contract", auth.ActionWrite).Put("/{id}/contracts/{contract_id}", handler.UpdateContract)
		access(vehicles, "fleet.vehicle.log.contract", auth.ActionUnlink).Delete("/{id}/contracts/{contract_id}", handler.DeleteContract)
	})

	r.Route("/fleet/cost-by-vehicle", func(report chi.Router) {
		access(report, "fleet.vehicle", auth.ActionRead).Get("/", handler.CostByVehicle)
	})
}