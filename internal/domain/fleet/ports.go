package fleet

import (
	"context"
	"time"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines all persistent storage operations for the Fleet domain.
type Repository interface {
	// Brands (global)
	CreateBrand(ctx context.Context, brand *VehicleBrand) error
	GetBrandByID(ctx context.Context, id int64) (*VehicleBrand, error)
	UpdateBrand(ctx context.Context, brand *VehicleBrand) error
	DeleteBrand(ctx context.Context, id int64) error
	ListBrands(ctx context.Context) ([]VehicleBrand, error)

	// Model categories (global)
	CreateModelCategory(ctx context.Context, category *VehicleModelCategory) error
	GetModelCategoryByID(ctx context.Context, id int64) (*VehicleModelCategory, error)
	UpdateModelCategory(ctx context.Context, category *VehicleModelCategory) error
	DeleteModelCategory(ctx context.Context, id int64) error
	ListModelCategories(ctx context.Context) ([]VehicleModelCategory, error)

	// Models (global)
	CreateModel(ctx context.Context, model *VehicleModel) error
	GetModelByID(ctx context.Context, id int64) (*VehicleModel, error)
	UpdateModel(ctx context.Context, model *VehicleModel) error
	DeleteModel(ctx context.Context, id int64) error
	ListModels(ctx context.Context, brandID *int64) ([]VehicleModel, error)

	// Tags (global)
	CreateTag(ctx context.Context, tag *VehicleTag) error
	GetTagByID(ctx context.Context, id int64) (*VehicleTag, error)
	UpdateTag(ctx context.Context, tag *VehicleTag) error
	DeleteTag(ctx context.Context, id int64) error
	ListTags(ctx context.Context) ([]VehicleTag, error)

	// Vehicle states (global)
	CreateState(ctx context.Context, state *VehicleState) error
	GetStateByID(ctx context.Context, id int64) (*VehicleState, error)
	UpdateState(ctx context.Context, state *VehicleState) error
	DeleteState(ctx context.Context, id int64) error
	ListStates(ctx context.Context) ([]VehicleState, error)

	// Service types (global)
	CreateServiceType(ctx context.Context, serviceType *ServiceType) error
	GetServiceTypeByID(ctx context.Context, id int64) (*ServiceType, error)
	UpdateServiceType(ctx context.Context, serviceType *ServiceType) error
	DeleteServiceType(ctx context.Context, id int64) error
	ListServiceTypes(ctx context.Context) ([]ServiceType, error)

	// Vehicles (company-scoped)
	CreateVehicle(ctx context.Context, vehicle *Vehicle) error
	GetVehicleByID(ctx context.Context, companyID, id int64) (*Vehicle, error)
	UpdateVehicle(ctx context.Context, vehicle *Vehicle) error
	DeleteVehicle(ctx context.Context, companyID, id int64) error
	ListVehicles(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Vehicle], error)
	SetVehicleTags(ctx context.Context, companyID, vehicleID int64, tagIDs []int64) error
	ListVehicleTagIDs(ctx context.Context, companyID, vehicleID int64) ([]int64, error)

	// Assignation logs (company-scoped by vehicle)
	CreateAssignationLog(ctx context.Context, log *VehicleAssignationLog) error
	GetAssignationLogByID(ctx context.Context, id int64) (*VehicleAssignationLog, error)
	UpdateAssignationLog(ctx context.Context, log *VehicleAssignationLog) error
	DeleteAssignationLog(ctx context.Context, id int64) error
	ListAssignationLogs(ctx context.Context, companyID, vehicleID int64) ([]VehicleAssignationLog, error)
	HasOverlappingAssignation(ctx context.Context, vehicleID int64, start, end *time.Time, excludeID int64) (bool, error)

	// Odometers (company-scoped by vehicle)
	CreateOdometer(ctx context.Context, odometer *VehicleOdometer) error
	GetOdometerByID(ctx context.Context, id int64) (*VehicleOdometer, error)
	UpdateOdometer(ctx context.Context, odometer *VehicleOdometer) error
	DeleteOdometer(ctx context.Context, id int64) error
	ListOdometers(ctx context.Context, companyID, vehicleID int64) ([]VehicleOdometer, error)
	GetLatestOdometer(ctx context.Context, companyID, vehicleID int64) (*VehicleOdometer, error)

	// Service logs (company-scoped)
	CreateLogService(ctx context.Context, service *VehicleLogService) error
	GetLogServiceByID(ctx context.Context, companyID, id int64) (*VehicleLogService, error)
	UpdateLogService(ctx context.Context, service *VehicleLogService) error
	DeleteLogService(ctx context.Context, companyID, id int64) error
	ListLogServices(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[VehicleLogService], error)

	// Contracts (company-scoped)
	CreateLogContract(ctx context.Context, contract *VehicleLogContract) error
	GetLogContractByID(ctx context.Context, companyID, id int64) (*VehicleLogContract, error)
	UpdateLogContract(ctx context.Context, contract *VehicleLogContract) error
	DeleteLogContract(ctx context.Context, companyID, id int64) error
	ListLogContracts(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[VehicleLogContract], error)
	ListContractsForRefresh(ctx context.Context) ([]VehicleLogContract, error)

	// Reports
	CostByVehicle(ctx context.Context, companyID int64) ([]VehicleCost, error)
}