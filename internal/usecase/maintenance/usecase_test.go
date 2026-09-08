package maintenanceusecase

import (
	"context"
	"testing"
	"time"

	maintenancestorage "cashflow_backend/internal/adapters/storage/maintenance"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/platform/pagination"
)

const companyID = 1

type fakeReminders struct {
	types     []activity.ActivityType
	scheduled []activity.Activity
}

func (f *fakeReminders) ListTypes(_ context.Context, _ *int64) ([]activity.ActivityType, error) {
	return f.types, nil
}

func (f *fakeReminders) Schedule(_ context.Context, a *activity.Activity) error {
	f.scheduled = append(f.scheduled, *a)
	return nil
}

func newTestService(repo *maintenancestorage.MemoryRepo, reminders ReminderScheduler) *Service {
	return New(repo, reminders, nil)
}

func TestCreateRequestSchedulesReminderForTechnician(t *testing.T) {
	ctx := context.Background()
	repo := maintenancestorage.NewMemoryRepo()
	reminders := &fakeReminders{
		types: []activity.ActivityType{{ID: 4, Name: "To-Do"}},
	}
	svc := newTestService(repo, reminders)

	equipment := &maintenance.Equipment{
		Name:             "CNC Unit A",
		TechnicianUserID: i64ptr(5),
		CompanyID:        companyID,
	}
	if err := svc.CreateEquipment(ctx, equipment); err != nil {
		t.Fatalf("failed to create equipment: %v", err)
	}

	schedule := time.Now().UTC().Add(24 * time.Hour)
	request := &maintenance.MaintenanceRequest{
		Name:            "MT/0001",
		EquipmentID:     &equipment.ID,
		MaintenanceType: maintenance.MaintenanceTypeCorrective,
		ScheduleDate:    &schedule,
		CompanyID:       companyID,
	}
	if err := svc.CreateRequest(ctx, request); err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if request.TechnicianUserID == nil || *request.TechnicianUserID != 5 {
		t.Fatalf("expected technician defaulted from equipment to 5, got %v", request.TechnicianUserID)
	}
	if len(reminders.scheduled) != 1 {
		t.Fatalf("expected 1 scheduled reminder, got %d", len(reminders.scheduled))
	}
	scheduled := reminders.scheduled[0]
	if scheduled.ResModel != "maintenance.request" || scheduled.AssignedUserID != 5 {
		t.Fatalf("unexpected reminder: %+v", scheduled)
	}
}

func TestCloseRequestSpawnsRecurringOccurrence(t *testing.T) {
	ctx := context.Background()
	repo := maintenancestorage.NewMemoryRepo()
	reminders := &fakeReminders{types: []activity.ActivityType{{ID: 4, Name: "To-Do"}}}
	svc := newTestService(repo, reminders)

	if err := svc.CreateStage(ctx, &maintenance.EquipmentStage{Name: "To-Do", Done: false}); err != nil {
		t.Fatalf("failed to create stage: %v", err)
	}
	doneStage := &maintenance.EquipmentStage{Name: "Repaired", Done: true}
	if err := svc.CreateStage(ctx, doneStage); err != nil {
		t.Fatalf("failed to create done stage: %v", err)
	}

	base := time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	request := &maintenance.MaintenanceRequest{
		Name:                 "PM/Oil Change",
		MaintenanceType:      maintenance.MaintenanceTypePreventive,
		RecurringMaintenance: true,
		ScheduleDate:         &base,
		RepeatInterval:       1,
		RepeatUnit:           maintenance.RepeatUnitWeek,
		RepeatType:           maintenance.RepeatTypeForever,
		CompanyID:            companyID,
	}
	if err := svc.CreateRequest(ctx, request); err != nil {
		t.Fatalf("failed to create recurring request: %v", err)
	}

	closed, err := svc.CloseRequest(ctx, companyID, request.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to close request: %v", err)
	}
	if closed.CloseDate == nil || closed.StageID == nil || *closed.StageID != doneStage.ID {
		t.Fatalf("expected close date and done stage, got %+v", closed)
	}
	if closed.KanbanState != maintenance.KanbanStateDone {
		t.Fatalf("expected kanban done, got %q", closed.KanbanState)
	}

	result, err := svc.ListRequests(ctx, companyID, nil, pageRequest())
	if err != nil {
		t.Fatalf("failed to list requests: %v", err)
	}
	spawned := findSpawned(result.Items, request.ID)
	if spawned == nil {
		t.Fatalf("expected spawned occurrence for %q", request.Name)
	}
	wantNext := base.AddDate(0, 0, 7)
	if spawned.ScheduleDate == nil || !spawned.ScheduleDate.Equal(wantNext) {
		t.Fatalf("expected next schedule %v, got %v", wantNext, spawned.ScheduleDate)
	}
	if spawned.ID == request.ID {
		t.Fatalf("spawned occurrence must be a new record")
	}
}

func TestCloseRequestArchivesWhenChainExhausted(t *testing.T) {
	ctx := context.Background()
	repo := maintenancestorage.NewMemoryRepo()
	svc := newTestService(repo, nil)

	base := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
	until := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	request := &maintenance.MaintenanceRequest{
		Name:                 "PM/Final",
		MaintenanceType:      maintenance.MaintenanceTypePreventive,
		RecurringMaintenance: true,
		ScheduleDate:         &base,
		RepeatInterval:       1,
		RepeatUnit:           maintenance.RepeatUnitWeek,
		RepeatType:           maintenance.RepeatTypeUntil,
		RepeatUntil:          &until,
		CompanyID:            companyID,
	}
	if err := svc.CreateRequest(ctx, request); err != nil {
		t.Fatalf("failed to create recurring request: %v", err)
	}

	// Base 08-31 + 1 week = 09-07 is past repeat_until 09-05, so the chain
	// is exhausted: the request is archived and nothing new is spawned.
	closed, err := svc.CloseRequest(ctx, companyID, request.ID, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to close: %v", err)
	}
	if !closed.Archived {
		t.Fatalf("expected chain exhausted request to be archived")
	}
	result, err := svc.ListRequests(ctx, companyID, nil, pageRequest())
	if err != nil {
		t.Fatalf("failed to list requests: %v", err)
	}
	if countRecurring(result.Items) != 1 {
		t.Fatalf("expected no spawned occurrence, got %d recurring records", countRecurring(result.Items))
	}
}

func findSpawned(items []maintenance.MaintenanceRequest, parentID int64) *maintenance.MaintenanceRequest {
	for i := range items {
		if items[i].ID != parentID && items[i].RecurringMaintenance {
			return &items[i]
		}
	}
	return nil
}

func countRecurring(items []maintenance.MaintenanceRequest) int {
	count := 0
	for _, item := range items {
		if item.RecurringMaintenance {
			count++
		}
	}
	return count
}

func pageRequest() pagination.PageRequest {
	return pagination.PageRequest{Page: 1, Limit: 100}
}

func i64ptr(v int64) *int64 { return &v }