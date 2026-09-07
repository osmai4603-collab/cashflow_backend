package projectusecase

import (
	"context"
	"testing"

	projectstorage "cashflow_backend/internal/adapters/storage/project"
	projectdomain "cashflow_backend/internal/domain/project"
)

func TestServiceDependencyLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := projectstorage.NewMemoryRepo()
	service := New(repo)

	value, err := service.CreateProject(ctx, CreateProjectInput{
		Name: "Delivery", CompanyID: 10, AllowDependencies: true, AllowSubtasks: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	stage := &projectdomain.TaskStage{Name: "Todo", ProjectIDs: []int64{value.ID}}
	if err := repo.CreateTaskStage(ctx, stage); err != nil {
		t.Fatal(err)
	}
	blocked, err := service.CreateTask(ctx, CreateTaskInput{Name: "Blocked", ProjectID: value.ID, StageID: stage.ID, CompanyID: 10})
	if err != nil {
		t.Fatal(err)
	}
	dependency, err := service.CreateTask(ctx, CreateTaskInput{Name: "Dependency", ProjectID: value.ID, StageID: stage.ID, CompanyID: 10})
	if err != nil {
		t.Fatal(err)
	}

	if err := service.AddDependency(ctx, 10, blocked.ID, dependency.ID); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetTaskByID(ctx, 10, blocked.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != projectdomain.TaskWaiting {
		t.Fatalf("expected waiting state, got %s", stored.State)
	}

	dependency.State = projectdomain.TaskDone
	if err := repo.UpdateTask(ctx, dependency); err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveDependency(ctx, 10, blocked.ID, dependency.ID); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.GetTaskByID(ctx, 10, blocked.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != projectdomain.TaskInProgress {
		t.Fatalf("expected in-progress state, got %s", stored.State)
	}
}

func TestServiceRejectsMilestoneWhenDisabled(t *testing.T) {
	ctx := context.Background()
	repo := projectstorage.NewMemoryRepo()
	service := New(repo)
	value, err := service.CreateProject(ctx, CreateProjectInput{Name: "No milestones", CompanyID: 10})
	if err != nil {
		t.Fatal(err)
	}
	err = service.CreateMilestone(ctx, 10, &projectdomain.Milestone{Name: "Blocked", ProjectID: value.ID})
	if err == nil {
		t.Fatal("expected disabled milestones to be rejected")
	}
}
