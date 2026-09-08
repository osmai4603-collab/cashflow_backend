package fleetusecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/fleet"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// dedupe removes duplicate tag IDs preserving order.
func dedupe(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ReminderScheduler schedules activity entries for fleet contracts.
// It is satisfied by *activityusecase.UseCase.
type ReminderScheduler interface {
	Schedule(ctx context.Context, a *activity.Activity) error
	ListTypes(ctx context.Context, companyID *int64) ([]activity.ActivityType, error)
}

// Service implements the use cases for the Fleet domain.
type Service struct {
	repo      fleet.Repository
	reminders ReminderScheduler
	logger    *slog.Logger

	mu         sync.Mutex
	todoTypeID *int64
}

// New creates a new Fleet use case service.
func New(repo fleet.Repository, reminders ReminderScheduler, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:      repo,
		reminders: reminders,
		logger:    logger,
	}
}

// --- Brands ---

func (s *Service) CreateBrand(ctx context.Context, brand *fleet.VehicleBrand) error {
	brand.Name = strings.TrimSpace(brand.Name)
	if brand.Name == "" {
		return platformerrors.Validation("brand name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.CreateBrand(ctx, brand)
}

func (s *Service) GetBrand(ctx context.Context, id int64) (*fleet.VehicleBrand, error) {
	return s.repo.GetBrandByID(ctx, id)
}

func (s *Service) UpdateBrand(ctx context.Context, brand *fleet.VehicleBrand) error {
	if brand.Name == "" {
		return platformerrors.Validation("brand name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.UpdateBrand(ctx, brand)
}

func (s *Service) DeleteBrand(ctx context.Context, id int64) error {
	return s.repo.DeleteBrand(ctx, id)
}

func (s *Service) ListBrands(ctx context.Context) ([]fleet.VehicleBrand, error) {
	return s.repo.ListBrands(ctx)
}

// --- Model categories ---

func (s *Service) CreateModelCategory(ctx context.Context, category *fleet.VehicleModelCategory) error {
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return platformerrors.Validation("model category name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.CreateModelCategory(ctx, category)
}

func (s *Service) GetModelCategory(ctx context.Context, id int64) (*fleet.VehicleModelCategory, error) {
	return s.repo.GetModelCategoryByID(ctx, id)
}

func (s *Service) UpdateModelCategory(ctx context.Context, category *fleet.VehicleModelCategory) error {
	if category.Name == "" {
		return platformerrors.Validation("model category name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.UpdateModelCategory(ctx, category)
}

func (s *Service) DeleteModelCategory(ctx context.Context, id int64) error {
	return s.repo.DeleteModelCategory(ctx, id)
}

func (s *Service) ListModelCategories(ctx context.Context) ([]fleet.VehicleModelCategory, error) {
	return s.repo.ListModelCategories(ctx)
}

// --- Models ---

func (s *Service) CreateModel(ctx context.Context, model *fleet.VehicleModel) error {
	model.Name = strings.TrimSpace(model.Name)
	if model.Name == "" {
		return platformerrors.Validation("model name is required", map[string]string{"name": "cannot be empty"})
	}
	if model.BrandID <= 0 {
		return platformerrors.Validation("model brand is required", map[string]string{"brand_id": "must be positive"})
	}
	return s.repo.CreateModel(ctx, model)
}

func (s *Service) GetModel(ctx context.Context, id int64) (*fleet.VehicleModel, error) {
	return s.repo.GetModelByID(ctx, id)
}

func (s *Service) UpdateModel(ctx context.Context, model *fleet.VehicleModel) error {
	if model.Name == "" || model.BrandID <= 0 {
		return platformerrors.Validation("model name and brand are required", nil)
	}
	return s.repo.UpdateModel(ctx, model)
}

func (s *Service) DeleteModel(ctx context.Context, id int64) error {
	return s.repo.DeleteModel(ctx, id)
}

func (s *Service) ListModels(ctx context.Context, brandID *int64) ([]fleet.VehicleModel, error) {
	return s.repo.ListModels(ctx, brandID)
}

// --- Tags ---

func (s *Service) CreateTag(ctx context.Context, tag *fleet.VehicleTag) error {
	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" {
		return platformerrors.Validation("tag name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.CreateTag(ctx, tag)
}

func (s *Service) GetTag(ctx context.Context, id int64) (*fleet.VehicleTag, error) {
	return s.repo.GetTagByID(ctx, id)
}

func (s *Service) UpdateTag(ctx context.Context, tag *fleet.VehicleTag) error {
	if tag.Name == "" {
		return platformerrors.Validation("tag name is required", map[string]string{"name": "cannot be empty"})
	}
	return s.repo.UpdateTag(ctx, tag)
}

func (s *Service) DeleteTag(ctx context.Context, id int64) error {
	return s.repo.DeleteTag(ctx, id)
}

func (s *Service) ListTags(ctx context.Context) ([]fleet.VehicleTag, error) {
	return s.repo.ListTags(ctx)
}

// --- States ---

func (s *Service) CreateState(ctx context.Context, state *fleet.VehicleState) error {
	if err := state.Validate(); err != nil {
		return err
	}
	return s.repo.CreateState(ctx, state)
}

func (s *Service) GetState(ctx context.Context, id int64) (*fleet.VehicleState, error) {
	return s.repo.GetStateByID(ctx, id)
}

func (s *Service) UpdateState(ctx context.Context, state *fleet.VehicleState) error {
	if err := state.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateState(ctx, state)
}

func (s *Service) DeleteState(ctx context.Context, id int64) error {
	return s.repo.DeleteState(ctx, id)
}

func (s *Service) ListStates(ctx context.Context) ([]fleet.VehicleState, error) {
	return s.repo.ListStates(ctx)
}

// --- Service types ---

func (s *Service) CreateServiceType(ctx context.Context, serviceType *fleet.ServiceType) error {
	if err := serviceType.Validate(); err != nil {
		return err
	}
	return s.repo.CreateServiceType(ctx, serviceType)
}

func (s *Service) GetServiceType(ctx context.Context, id int64) (*fleet.ServiceType, error) {
	return s.repo.GetServiceTypeByID(ctx, id)
}

func (s *Service) UpdateServiceType(ctx context.Context, serviceType *fleet.ServiceType) error {
	if err := serviceType.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateServiceType(ctx, serviceType)
}

func (s *Service) DeleteServiceType(ctx context.Context, id int64) error {
	return s.repo.DeleteServiceType(ctx, id)
}

func (s *Service) ListServiceTypes(ctx context.Context) ([]fleet.ServiceType, error) {
	return s.repo.ListServiceTypes(ctx)
}

// --- Vehicles ---

func (s *Service) CreateVehicle(ctx context.Context, vehicle *fleet.Vehicle) error {
	if vehicle.Name == "" {
		vehicle.Name = vehicle.LicensePlate
	}
	vehicle.Name = strings.TrimSpace(vehicle.Name)
	if err := vehicle.Validate(); err != nil {
		return err
	}
	vehicle.State = fleet.StateActive
	if vehicle.StateID != nil {
		state, err := s.repo.GetStateByID(ctx, *vehicle.StateID)
		if err != nil {
			return platformerrors.Validation("vehicle state does not exist", map[string]string{"state_id": "invalid"})
		}
		switch state.Name {
		case fleet.PresetStateRegistered:
			vehicle.State = fleet.StateActive
		case fleet.PresetStateDowngraded:
			vehicle.State = fleet.StateRetired
		default:
			vehicle.State = fleet.StateActive
		}
	}
	if err := s.repo.CreateVehicle(ctx, vehicle); err != nil {
		return err
	}
	if vehicle.Tags != nil {
		if err := s.repo.SetVehicleTags(ctx, vehicle.CompanyID, vehicle.ID, dedupe(vehicle.Tags)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetVehicle(ctx context.Context, companyID, id int64) (*fleet.Vehicle, error) {
	vehicle, err := s.repo.GetVehicleByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	tags, err := s.repo.ListVehicleTagIDs(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	vehicle.Tags = tags
	return vehicle, nil
}

func (s *Service) UpdateVehicle(ctx context.Context, vehicle *fleet.Vehicle) error {
	if err := vehicle.Validate(); err != nil {
		return err
	}
	if err := s.repo.UpdateVehicle(ctx, vehicle); err != nil {
		return err
	}
	if vehicle.Tags != nil {
		if err := s.repo.SetVehicleTags(ctx, vehicle.CompanyID, vehicle.ID, dedupe(vehicle.Tags)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteVehicle(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteVehicle(ctx, companyID, id)
}

func (s *Service) ListVehicles(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.Vehicle], error) {
	return s.repo.ListVehicles(ctx, companyID, f, page)
}

// --- Assignation logs ---

func (s *Service) CreateAssignationLog(ctx context.Context, log *fleet.VehicleAssignationLog) error {
	if err := s.validateAssignationLog(ctx, log.VehicleID, log.DateStart, log.DateEnd, 0); err != nil {
		return err
	}
	return s.repo.CreateAssignationLog(ctx, log)
}

func (s *Service) GetAssignationLog(ctx context.Context, id int64) (*fleet.VehicleAssignationLog, error) {
	return s.repo.GetAssignationLogByID(ctx, id)
}

func (s *Service) UpdateAssignationLog(ctx context.Context, log *fleet.VehicleAssignationLog) error {
	if err := s.validateAssignationLog(ctx, log.VehicleID, log.DateStart, log.DateEnd, log.ID); err != nil {
		return err
	}
	return s.repo.UpdateAssignationLog(ctx, log)
}

func (s *Service) DeleteAssignationLog(ctx context.Context, id int64) error {
	return s.repo.DeleteAssignationLog(ctx, id)
}

func (s *Service) ListAssignationLogs(ctx context.Context, companyID, vehicleID int64) ([]fleet.VehicleAssignationLog, error) {
	return s.repo.ListAssignationLogs(ctx, companyID, vehicleID)
}

// validateAssignationLog rejects driver assignments whose windows overlap an
// existing assignment on the same vehicle.
func (s *Service) validateAssignationLog(ctx context.Context, vehicleID int64, start, end *time.Time, excludeID int64) error {
	overlap, err := s.repo.HasOverlappingAssignation(ctx, vehicleID, start, end, excludeID)
	if err != nil {
		return err
	}
	if overlap {
		return platformerrors.Conflict("assignation overlaps an existing assignment for this vehicle", nil)
	}
	return nil
}

// --- Odometers ---

func (s *Service) CreateOdometer(ctx context.Context, companyID int64, odometer *fleet.VehicleOdometer) error {
	if err := s.validateOdometer(ctx, companyID, odometer.VehicleID, odometer); err != nil {
		return err
	}
	return s.repo.CreateOdometer(ctx, odometer)
}

func (s *Service) GetOdometer(ctx context.Context, id int64) (*fleet.VehicleOdometer, error) {
	return s.repo.GetOdometerByID(ctx, id)
}

func (s *Service) UpdateOdometer(ctx context.Context, companyID int64, odometer *fleet.VehicleOdometer) error {
	if err := s.validateOdometer(ctx, companyID, odometer.VehicleID, odometer); err != nil {
		return err
	}
	return s.repo.UpdateOdometer(ctx, odometer)
}

func (s *Service) DeleteOdometer(ctx context.Context, id int64) error {
	return s.repo.DeleteOdometer(ctx, id)
}

func (s *Service) ListOdometers(ctx context.Context, companyID, vehicleID int64) ([]fleet.VehicleOdometer, error) {
	return s.repo.ListOdometers(ctx, companyID, vehicleID)
}

func (s *Service) GetLatestOdometer(ctx context.Context, companyID, vehicleID int64) (*fleet.VehicleOdometer, error) {
	return s.repo.GetLatestOdometer(ctx, companyID, vehicleID)
}

// validateOdometer blocks readings that decrease the vehicle's odometer.
func (s *Service) validateOdometer(ctx context.Context, companyID, vehicleID int64, odometer *fleet.VehicleOdometer) error {
	latest, err := s.repo.GetLatestOdometer(ctx, companyID, vehicleID)
	if err != nil {
		var appErr *platformerrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != platformerrors.CodeNotFound {
			return err
		}
	}
	if latest != nil && odometer.Value < latest.Value {
		return platformerrors.Validation("odometer reading cannot decrease", nil)
	}
	return nil
}

// --- Service logs ---

func (s *Service) CreateLogService(ctx context.Context, service *fleet.VehicleLogService) error {
	if err := service.Validate(); err != nil {
		return err
	}
	return s.repo.CreateLogService(ctx, service)
}

func (s *Service) GetLogService(ctx context.Context, companyID, id int64) (*fleet.VehicleLogService, error) {
	return s.repo.GetLogServiceByID(ctx, companyID, id)
}

func (s *Service) UpdateLogService(ctx context.Context, service *fleet.VehicleLogService) error {
	if err := service.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateLogService(ctx, service)
}

func (s *Service) DeleteLogService(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteLogService(ctx, companyID, id)
}

func (s *Service) ListLogServices(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.VehicleLogService], error) {
	return s.repo.ListLogServices(ctx, companyID, f, page)
}

// CostByVehicle aggregates service spend per vehicle for the company.
func (s *Service) CostByVehicle(ctx context.Context, companyID int64) ([]fleet.VehicleCost, error) {
	return s.repo.CostByVehicle(ctx, companyID)
}

// --- Contracts ---

func (s *Service) CreateLogContract(ctx context.Context, contract *fleet.VehicleLogContract) error {
	if err := contract.Validate(); err != nil {
		return err
	}
	return s.repo.CreateLogContract(ctx, contract)
}

func (s *Service) GetLogContract(ctx context.Context, companyID, id int64) (*fleet.VehicleLogContract, error) {
	return s.repo.GetLogContractByID(ctx, companyID, id)
}

func (s *Service) UpdateLogContract(ctx context.Context, contract *fleet.VehicleLogContract) error {
	if err := contract.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateLogContract(ctx, contract)
}

func (s *Service) DeleteLogContract(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteLogContract(ctx, companyID, id)
}

func (s *Service) ListLogContracts(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.VehicleLogContract], error) {
	return s.repo.ListLogContracts(ctx, companyID, f, page)
}

// RefreshExpiredContracts transitions open contracts whose expiration date
// passed into the expired state. It returns the number of refreshed contracts.
func (s *Service) RefreshExpiredContracts(ctx context.Context, now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	contracts, err := s.repo.ListContractsForRefresh(ctx)
	if err != nil {
		return 0, err
	}
	refreshed := 0
	for i := range contracts {
		if contracts[i].RefreshState(now) {
			if err := s.repo.UpdateLogContract(ctx, &contracts[i]); err != nil {
				s.logger.Warn("failed to refresh contract state", "contract_id", contracts[i].ID, "error", err)
				continue
			}
			refreshed++
		}
	}
	return refreshed, nil
}

// ScheduleContractReminders schedules a To-Do reminder whenever an open
// contract is expiring (within 30 days or today). Reminders are best-effort.
func (s *Service) ScheduleContractReminders(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if s.reminders == nil {
		return nil
	}
	contracts, err := s.repo.ListContractsForRefresh(ctx)
	if err != nil {
		return err
	}
	for i := range contracts {
		contract := &contracts[i]
		if contract.UserID == nil || *contract.UserID <= 0 {
			continue
		}
		if contract.State != fleet.ContractStateOpen || contract.ExpirationDate == nil {
			continue
		}
		daysLeft := contract.DaysLeft(now)
		if daysLeft == nil || *daysLeft < 0 {
			continue
		}
		if *daysLeft > 30 && !contract.ExpiresToday(now) {
			continue
		}
		typeID, err := s.resolveTodoTypeID(ctx, contract.CompanyID)
		if err != nil {
			s.logger.Warn("could not resolve to-do activity type", "error", err)
			continue
		}
		userID := *contract.UserID
		resID := contract.ID
		summary := "Contract expiring: " + contract.Name
		if contract.Name == "" {
			summary = "Vehicle contract expiring"
		}
		act := &activity.Activity{
			ActivityTypeID: typeID,
			Summary:        summary,
			DateDeadline:   *contract.ExpirationDate,
			AssignedUserID: userID,
			ResModel:       "fleet.vehicle.log.contract",
			ResID:          &resID,
			CompanyID:      contract.CompanyID,
			CreatedBy:      &userID,
		}
		if err := s.reminders.Schedule(ctx, act); err != nil {
			s.logger.Warn("could not schedule contract expiry reminder", "contract_id", contract.ID, "error", err)
		}
	}
	return nil
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