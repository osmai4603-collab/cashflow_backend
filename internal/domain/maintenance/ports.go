package maintenance

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines all persistent storage operations for the Maintenance domain.
type Repository interface {
	// Categories
	CreateCategory(ctx context.Context, category *EquipmentCategory) error
	GetCategoryByID(ctx context.Context, companyID *int64, id int64) (*EquipmentCategory, error)
	UpdateCategory(ctx context.Context, category *EquipmentCategory) error
	DeleteCategory(ctx context.Context, companyID *int64, id int64) error
	ListCategories(ctx context.Context, companyID *int64) ([]EquipmentCategory, error)

	// Stages (global, Odoo maintenance.stage)
	CreateStage(ctx context.Context, stage *EquipmentStage) error
	GetStageByID(ctx context.Context, id int64) (*EquipmentStage, error)
	UpdateStage(ctx context.Context, stage *EquipmentStage) error
	DeleteStage(ctx context.Context, id int64) error
	ListStages(ctx context.Context) ([]EquipmentStage, error)

	// Teams
	CreateTeam(ctx context.Context, team *Team) error
	GetTeamByID(ctx context.Context, companyID *int64, id int64) (*Team, error)
	UpdateTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, companyID *int64, id int64) error
	ListTeams(ctx context.Context, companyID *int64) ([]Team, error)
	SetTeamMembers(ctx context.Context, teamID int64, memberIDs []int64) error

	// Equipment
	CreateEquipment(ctx context.Context, equipment *Equipment) error
	GetEquipmentByID(ctx context.Context, companyID, id int64) (*Equipment, error)
	UpdateEquipment(ctx context.Context, equipment *Equipment) error
	DeleteEquipment(ctx context.Context, companyID, id int64) error
	ListEquipments(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Equipment], error)

	// Requests
	CreateRequest(ctx context.Context, request *MaintenanceRequest) error
	GetRequestByID(ctx context.Context, companyID, id int64) (*MaintenanceRequest, error)
	UpdateRequest(ctx context.Context, request *MaintenanceRequest) error
	DeleteRequest(ctx context.Context, companyID, id int64) error
	ListRequests(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[MaintenanceRequest], error)
	// ListRecurringOpenRequests returns non-archived recurring requests used by the scheduler.
	ListRecurringOpenRequests(ctx context.Context) ([]MaintenanceRequest, error)
}