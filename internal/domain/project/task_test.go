package project

import (
	"testing"
	"time"
)

func TestTaskValidateSetsDefaults(t *testing.T) {
	task := Task{Name: "  Prepare report  ", ProjectID: 1, StageID: 2, CompanyID: 3}
	if err := task.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if task.Name != "Prepare report" || task.Priority != PriorityNormal || task.State != TaskInProgress {
		t.Fatalf("unexpected defaults: %#v", task)
	}
}

func TestDependencyCreatesCycle(t *testing.T) {
	dependencies := map[int64][]int64{2: {1}, 3: {2}}
	if !DependencyCreatesCycle(1, 3, dependencies) {
		t.Fatal("expected dependency cycle to be detected")
	}
	if DependencyCreatesCycle(4, 3, dependencies) {
		t.Fatal("did not expect a cycle")
	}
}

func TestMilestoneCannotBeReachedWithOpenTasks(t *testing.T) {
	milestone := Milestone{Name: "Release", ProjectID: 1}
	if err := milestone.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := milestone.Reach(time.Now(), true); err == nil {
		t.Fatal("expected open tasks to block milestone")
	}
}
