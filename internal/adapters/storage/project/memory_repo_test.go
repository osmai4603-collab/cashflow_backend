package projectstorage

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/project"
)

func TestMemoryRepoEnforcesCompanyAndDependencyBoundaries(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	first := &project.Project{Name: "First", CompanyID: 10, AllowDependencies: true}
	second := &project.Project{Name: "Second", CompanyID: 10, AllowDependencies: true}
	otherCompany := &project.Project{Name: "Other", CompanyID: 20}
	for _, value := range []*project.Project{first, second, otherCompany} {
		if err := repo.CreateProject(ctx, value); err != nil {
			t.Fatal(err)
		}
	}

	stage := &project.TaskStage{Name: "Todo", ProjectIDs: []int64{first.ID, second.ID}}
	if err := repo.CreateTaskStage(ctx, stage); err != nil {
		t.Fatal(err)
	}

	firstTask := &project.Task{Name: "First task", ProjectID: first.ID, StageID: stage.ID, CompanyID: 10}
	secondTask := &project.Task{Name: "Second task", ProjectID: first.ID, StageID: stage.ID, CompanyID: 10}
	for _, value := range []*project.Task{firstTask, secondTask} {
		if err := repo.CreateTask(ctx, value); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.AddTaskDependency(ctx, 10, secondTask.ID, firstTask.ID); err != nil {
		t.Fatalf("expected same-company dependency: %v", err)
	}
	if err := repo.AddTaskDependency(ctx, 10, firstTask.ID, secondTask.ID); err == nil {
		t.Fatal("expected cyclic dependency to be rejected")
	}

	thirdProject := &project.Project{Name: "Third", CompanyID: 10, AllowDependencies: true}
	if err := repo.CreateProject(ctx, thirdProject); err != nil {
		t.Fatal(err)
	}
	thirdStage := &project.TaskStage{Name: "Third todo", ProjectIDs: []int64{thirdProject.ID}}
	if err := repo.CreateTaskStage(ctx, thirdStage); err != nil {
		t.Fatal(err)
	}
	thirdTask := &project.Task{Name: "Third task", ProjectID: thirdProject.ID, StageID: thirdStage.ID, CompanyID: 10}
	if err := repo.CreateTask(ctx, thirdTask); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTaskDependency(ctx, 10, thirdTask.ID, firstTask.ID); err == nil {
		t.Fatal("expected cross-project dependency to be rejected")
	}

	if _, err := repo.GetProjectByID(ctx, 20, first.ID); err == nil {
		t.Fatal("expected another company to be unable to read the project")
	}
}
