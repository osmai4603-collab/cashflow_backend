package maintenanceusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ReminderScheduler schedules activity entries for maintenance requests.
// It is satisfied by *activityusecase.UseCase.
type ReminderScheduler interface {
	Schedule(ctx context.Context, a *activity.Activity) error
	ListTypes(ctx context.Context, companyID *int64) ([]activity.ActivityType, error)
}

// Service implements the use cases for the Maintenance domain.
type Service struct {
	repo      maintenance.Repository
	reminders ReminderScheduler
	logger    *slog.Logger

	mu         sync.Mutex
	todoTypeID *int64
}

// New creates a new Maintenance use case service.
func New(repo maintenance.Repository, reminders ReminderScheduler, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:      repo,
		reminders: reminders,
		logger:    logger,
	}
}

// --- Equipment Categories ---

func (s *Service) CreateCategory(ctx context.Context, category *maintenance.EquipmentCategory) error {
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return platformerrors.Validation("category name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.CreateCategory(ctx, category)
}

func (s *Service) GetCategory(ctx context.Context, companyID, id int64) (*maintenance.EquipmentCategory, error) {
	return s.repo.GetCategoryByID(ctx, &companyID, id)
}

func (s *Service) UpdateCategory(ctx context.Context, category *maintenance.EquipmentCategory) error {
	if category.Name == "" {
		return platformerrors.Validation("category name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.UpdateCategory(ctx, category)
}

func (s *Service) DeleteCategory(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteCategory(ctx, &companyID, id)
}

func (s *Service) ListCategories(ctx context.Context, companyID *int64) ([]maintenance.EquipmentCategory, error) {
	return s.repo.ListCategories(ctx, companyID)
}

// --- Stages ---

func (s *Service) CreateStage(ctx context.Context, stage *maintenance.EquipmentStage) error {
	stage.Name = strings.TrimSpace(stage.Name)
	if stage.Name == "" {
		return platformerrors.Validation("stage name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.CreateStage(ctx, stage)
}

func (s *Service) GetStage(ctx context.Context, id int64) (*maintenance.EquipmentStage, error) {
	return s.repo.GetStageByID(ctx, id)
}

func (s *Service) UpdateStage(ctx context.Context, stage *maintenance.EquipmentStage) error {
	if stage.Name == "" {
		return platformerrors.Validation("stage name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.UpdateStage(ctx, stage)
}

func (s *Service) DeleteStage(ctx context.Context, id int64) error {
	return s.repo.DeleteStage(ctx, id)
}

func (s *Service) ListStages(ctx context.Context) ([]maintenance.EquipmentStage, error) {
	return s.repo.ListStages(ctx)
}

// --- Teams ---

func (s *Service) CreateTeam(ctx context.Context, team *maintenance.Team) error {
	if err := team.Validate(); err != nil {
		return err
	}
	return s.repo.CreateTeam(ctx, team)
}

func (s *Service) GetTeam(ctx context.Context, companyID *int64, id int64) (*maintenance.Team, error) {
	return s.repo.GetTeamByID(ctx, companyID, id)
}

func (s *Service) UpdateTeam(ctx context.Context, team *maintenance.Team) error {
	if err := team.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateTeam(ctx, team)
}

func (s *Service) DeleteTeam(ctx context.Context, companyID *int64, id int64) error {
	return s.repo.DeleteTeam(ctx, companyID, id)
}

func (s *Service) ListTeams(ctx context.Context, companyID *int64) ([]maintenance.Team, error) {
	return s.repo.ListTeams(ctx, companyID)
}

func (s *Service) SetTeamMembers(ctx context.Context, teamID int64, memberIDs []int64) error {
	return s.repo.SetTeamMembers(ctx, teamID, memberIDs)
}

// --- Equipment ---

func (s *Service) CreateEquipment(ctx context.Context, equipment *maintenance.Equipment) error {
	if err := equipment.Validate(); err != nil {
		return err
	}
	return s.repo.CreateEquipment(ctx, equipment)
}

func (s *Service) GetEquipment(ctx context.Context, companyID, id int64) (*maintenance.Equipment, error) {
	return s.repo.GetEquipmentByID(ctx, companyID, id)
}

func (s *Service) UpdateEquipment(ctx context.Context, equipment *maintenance.Equipment) error {
	if err := equipment.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateEquipment(ctx, equipment)
}

func (s *Service) DeleteEquipment(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteEquipment(ctx, companyID, id)
}

func (s *Service) ListEquipments(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[maintenance.Equipment], error) {
	return s.repo.ListEquipments(ctx, companyID, f, page)
}

// --- Requests ---

func (s *Service) CreateRequest(ctx context.Context, request *maintenance.MaintenanceRequest) error {
	if request.RequestDate.IsZero() {
		request.RequestDate = time.Now().UTC()
	}
	if request.StageID == nil {
		if stages, err := s.repo.ListStages(ctx); err == nil {
			for _, stage := range stages {
				if !stage.Fold {
					id := stage.ID
					request.StageID = &id
					break
				}
			}
		}
	}
	if request.EquipmentID != nil {
		equipment, err := s.repo.GetEquipmentByID(ctx, request.CompanyID, *request.EquipmentID)
		if err != nil {
			return platformerrors.Validation("equipment does not exist", map[string]string{"equipment_id": "invalid"})
		}
		if request.TeamID == nil {
			request.TeamID = equipment.TeamID
		}
		if request.TechnicianUserID == nil {
			request.TechnicianUserID = equipment.TechnicianUserID
		}
		if request.OwnerUserID == nil {
			request.OwnerUserID = equipment.OwnerUserID
		}
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if err := s.repo.CreateRequest(ctx, request); err != nil {
		return err
	}
	s.scheduleRequestReminder(ctx, request)
	return nil
}

func (s *Service) GetRequest(ctx context.Context, companyID, id int64) (*maintenance.MaintenanceRequest, error) {
	return s.repo.GetRequestByID(ctx, companyID, id)
}

func (s *Service) UpdateRequest(ctx context.Context, request *maintenance.MaintenanceRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateRequest(ctx, request)
}

func (s *Service) DeleteRequest(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteRequest(ctx, companyID, id)
}

func (s *Service) ListRequests(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[maintenance.MaintenanceRequest], error) {
	return s.repo.ListRequests(ctx, companyID, f, page)
}

// CloseRequest closes a request and, when recurring, spawns the next occurrence.
func (s *Service) CloseRequest(ctx context.Context, companyID, id int64, now time.Time) (*maintenance.MaintenanceRequest, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	request, err := s.repo.GetRequestByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if request.CloseDate != nil {
		return nil, platformerrors.Conflict("request is already closed", nil)
	}
	done := now
	request.CloseDate = &done
	request.KanbanState = maintenance.KanbanStateDone
	if stageID, err := s.doneStage(ctx); err == nil && stageID != nil {
		request.StageID = stageID
	}

	if request.RecurringMaintenance {
		next, err := request.NextOccurrence()
		if err != nil {
			return nil, err
		}
		if next != nil {
			spawned := *request
			spawned.ID = 0
			spawned.ScheduleDate = next
			spawned.CloseDate = nil
			spawned.KanbanState = maintenance.KanbanStateNormal
			spawned.StageID = nil
			spawned.Archived = false
			spawned.Name = fmt.Sprintf("%s - %s", request.Name, next.Format("2006-01-02"))
			spawned.Audit = audit.Fields{}
			if err := s.repo.CreateRequest(ctx, &spawned); err != nil {
				return nil, err
			}
			s.logger.Info("spawned recurring maintenance request", "parent_id", request.ID, "child_id", spawned.ID, "schedule_date", next)
		} else {
			request.Archived = true
		}
	}

	if err := s.repo.UpdateRequest(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

// ArchiveRequest archives a maintenance request.
func (s *Service) ArchiveRequest(ctx context.Context, companyID, id int64) (*maintenance.MaintenanceRequest, error) {
	request, err := s.repo.GetRequestByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if request.Archived {
		return nil, platformerrors.Conflict("request is already archived", nil)
	}
	request.Archived = true
	if err := s.repo.UpdateRequest(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

// Dashboard groups open requests by stage for the kanban view.
type Dashboard struct {
	Requests []maintenance.MaintenanceRequest `json:"requests"`
	Kanban   []KanbanColumn                   `json:"kanban"`
}

// KanbanColumn represents one stage column in the maintenance kanban.
type KanbanColumn struct {
	StageID int64  `json:"stage_id"`
	Name    string `json:"name"`
	Count   int    `json:"count"`
}

func (s *Service) Dashboard(ctx context.Context, companyID int64) (*Dashboard, error) {
	stages, err := s.repo.ListStages(ctx)
	if err != nil {
		return nil, err
	}
	page, err := s.repo.ListRequests(ctx, companyID, nil, pagination.PageRequest{Page: 1, Limit: pagination.MaxLimit})
	if err != nil {
		return nil, err
	}
	requests := page.Items
	counts := map[int64]int{}
	for _, r := range requests {
		if r.StageID != nil {
			counts[*r.StageID]++
		}
	}
	columns := make([]KanbanColumn, 0, len(stages))
	for _, stage := range stages {
		columns = append(columns, KanbanColumn{
			StageID: stage.ID,
			Name:    stage.Name,
			Count:   counts[stage.ID],
		})
	}
	return &Dashboard{Requests: requests, Kanban: columns}, nil
}

// doneStage returns the ID of the first "done" stage, if any.
func (s *Service) doneStage(ctx context.Context) (*int64, error) {
	stages, err := s.repo.ListStages(ctx)
	if err != nil {
		return nil, err
	}
	for _, stage := range stages {
		if stage.Done {
			id := stage.ID
			return &id, nil
		}
	}
	return nil, nil
}

// RemindRecurringDue schedules reminder activities for open recurring requests
// whose schedule date has arrived. It is invoked periodically by the scheduler.
func (s *Service) RemindRecurringDue(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	requests, err := s.repo.ListRecurringOpenRequests(ctx)
	if err != nil {
		return err
	}
	for i := range requests {
		request := &requests[i]
		if request.ScheduleDate == nil || request.ScheduleDate.After(now) {
			continue
		}
		if request.CloseDate != nil {
			continue
		}
		s.scheduleRequestReminder(ctx, request)
	}
	return nil
}

// scheduleRequestReminder schedules a best-effort To-Do activity for the
// technician (or owner) when the request is due, mirroring Odoo's follow-up
// activities. Failures are logged and never fail the request lifecycle.
func (s *Service) scheduleRequestReminder(ctx context.Context, request *maintenance.MaintenanceRequest) {
	if s.reminders == nil {
		return
	}
	var userID int64
	switch {
	case request.TechnicianUserID != nil:
		userID = *request.TechnicianUserID
	case request.OwnerUserID != nil:
		userID = *request.OwnerUserID
	default:
		return
	}
	if userID <= 0 {
		return
	}
	typeID, err := s.resolveTodoTypeID(ctx, request.CompanyID)
	if err != nil {
		s.logger.Warn("could not resolve to-do activity type", "error", err)
		return
	}
	deadline := time.Now().UTC().Add(24 * time.Hour)
	if request.ScheduleDate != nil {
		deadline = *request.ScheduleDate
	}
	resID := request.ID
	act := &activity.Activity{
		ActivityTypeID: typeID,
		Summary:        "Maintenance: " + request.Name,
		Note:           request.Description,
		DateDeadline:   deadline,
		AssignedUserID: userID,
		ResModel:       "maintenance.request",
		ResID:          &resID,
		CompanyID:      request.CompanyID,
		CreatedBy:      &userID,
	}
	if err := s.reminders.Schedule(ctx, act); err != nil {
		s.logger.Warn("could not schedule maintenance reminder", "request_id", request.ID, "error", err)
	}
}

// resolveTodoTypeID finds the seeded global "To-Do" activity type, caching it.
func (s *Service) resolveTodoTypeID(ctx context.Context, companyID int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.todoTypeID != nil {
		return *s.todoTypeID, nil
	}
	types, err := s.reminders.ListTypes(ctx, &companyID)
	if err != nil {
		return 0, err
	}
	for _, t := range types {
		if strings.EqualFold(strings.TrimSpace(t.Name), "To-Do") {
			id := t.ID
			s.todoTypeID = &id
			return id, nil
		}
	}
	return 0, platformerrors.NotFound("to-do activity type not found")
}