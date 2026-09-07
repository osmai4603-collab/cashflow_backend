package projecthttp

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

	r.Route("/projects", func(projects chi.Router) {
		access(projects, "project.project", auth.ActionCreate).Post("/", handler.CreateProject)
		access(projects, "project.project", auth.ActionRead).Get("/", handler.ListProjects)
		access(projects, "project.project", auth.ActionRead).Get("/{id}", handler.GetProject)
		access(projects, "project.project", auth.ActionWrite).Put("/{id}", handler.UpdateProject)
		access(projects, "project.project", auth.ActionUnlink).Delete("/{id}", handler.DeleteProject)
		access(projects, "project.task", auth.ActionCreate).Post("/tasks", handler.CreateTask)
		access(projects, "project.task", auth.ActionRead).Get("/{id}/tasks", handler.ListTasks)
		access(projects, "project.milestone", auth.ActionCreate).Post("/{id}/milestones", handler.CreateMilestone)
		access(projects, "project.milestone", auth.ActionRead).Get("/{id}/milestones", handler.ListMilestones)
	})
	r.Route("/project-stages", func(stages chi.Router) {
		access(stages, "project.project.stage", auth.ActionCreate).Post("/", handler.CreateProjectStage)
		access(stages, "project.project.stage", auth.ActionRead).Get("/", handler.ListProjectStages)
		access(stages, "project.project.stage", auth.ActionWrite).Put("/{id}", handler.UpdateProjectStage)
		access(stages, "project.project.stage", auth.ActionUnlink).Delete("/{id}", handler.DeleteProjectStage)
	})
	r.Route("/task-stages", func(stages chi.Router) {
		access(stages, "project.task.type", auth.ActionCreate).Post("/", handler.CreateTaskStage)
		access(stages, "project.task.type", auth.ActionRead).Get("/", handler.ListTaskStages)
		access(stages, "project.task.type", auth.ActionWrite).Put("/{id}", handler.UpdateTaskStage)
		access(stages, "project.task.type", auth.ActionUnlink).Delete("/{id}", handler.DeleteTaskStage)
	})
	r.Route("/task-tags", func(tags chi.Router) {
		access(tags, "project.tags", auth.ActionCreate).Post("/", handler.CreateTaskTag)
		access(tags, "project.tags", auth.ActionRead).Get("/", handler.ListTaskTags)
		access(tags, "project.tags", auth.ActionWrite).Put("/{id}", handler.UpdateTaskTag)
		access(tags, "project.tags", auth.ActionUnlink).Delete("/{id}", handler.DeleteTaskTag)
	})
	access(r, "project.milestone", auth.ActionWrite).Post("/milestones/{id}/reach", handler.ReachMilestone)

	r.Route("/tasks", func(tasks chi.Router) {
		access(tasks, "project.task", auth.ActionRead).Get("/{id}", handler.GetTask)
		access(tasks, "project.task", auth.ActionWrite).Put("/{id}", handler.UpdateTask)
		access(tasks, "project.task", auth.ActionUnlink).Delete("/{id}", handler.DeleteTask)
		access(tasks, "project.task", auth.ActionWrite).Post("/{id}/dependencies", handler.AddDependency)
		access(tasks, "project.task", auth.ActionWrite).Delete("/{id}/dependencies", handler.RemoveDependency)
	})
}
