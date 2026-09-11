package mrp

import (
	"context"
	"time"
)

// Repository defines the persistence contract for MRP entities.
type Repository interface {
	// Workcenter
	CreateWorkcenter(ctx context.Context, wc *Workcenter) error
	UpdateWorkcenter(ctx context.Context, wc *Workcenter) error
	GetWorkcenterByID(ctx context.Context, id int64) (*Workcenter, error)
	ListWorkcenters(ctx context.Context, filter WorkcenterFilter) ([]*Workcenter, int64, error)
	DeleteWorkcenter(ctx context.Context, id int64) error

	// Bill of Materials
	CreateBoM(ctx context.Context, bom *BillOfMaterials) error
	UpdateBoM(ctx context.Context, bom *BillOfMaterials) error
	GetBoMByID(ctx context.Context, id int64) (*BillOfMaterials, error)
	ListBoMs(ctx context.Context, filter BoMFilter) ([]*BillOfMaterials, int64, error)
	DeleteBoM(ctx context.Context, id int64) error

	// BoM Lines & Operations
	AddBomLine(ctx context.Context, line *BomLine) error
	RemoveBomLine(ctx context.Context, id int64) error
	AddRoutingOperation(ctx context.Context, op *RoutingOperation) error
	RemoveRoutingOperation(ctx context.Context, id int64) error

	// Production Orders
	CreateProduction(ctx context.Context, mo *ProductionOrder) error
	UpdateProduction(ctx context.Context, mo *ProductionOrder) error
	GetProductionByID(ctx context.Context, id int64) (*ProductionOrder, error)
	ListProductions(ctx context.Context, filter ProductionFilter) ([]*ProductionOrder, int64, error)

	// Workorders
	CreateWorkorder(ctx context.Context, wo *Workorder) error
	UpdateWorkorder(ctx context.Context, wo *Workorder) error
	GetWorkorderByID(ctx context.Context, id int64) (*Workorder, error)
	ListWorkorders(ctx context.Context, productionID int64) ([]*Workorder, error)

	// Unbuild Orders
	CreateUnbuild(ctx context.Context, uo *UnbuildOrder) error
	GetUnbuildByID(ctx context.Context, id int64) (*UnbuildOrder, error)
}

// Phase2Repository persists the advanced MRP scheduling and time-tracking data.
// It is intentionally separate so existing adapters can adopt Phase 2 incrementally.
type Phase2Repository interface {
	CreateWorkorderTimeLog(context.Context, *WorkorderTimeLog) error
	ListWorkorderTimeLogs(context.Context, int64) ([]WorkorderTimeLog, error)
	CreateWorkcenterCalendar(context.Context, *WorkcenterCalendar) error
	ListWorkcenterCalendars(context.Context, int64) ([]WorkcenterCalendar, error)
}

// WorkcenterFilter defines criteria for listing workcenters.
type WorkcenterFilter struct {
	CompanyID *int64
	Active    *bool
	Search    string
	Limit     int
	Offset    int
}

// BoMFilter defines criteria for listing BoMs.
type BoMFilter struct {
	ProductID *int64
	CompanyID *int64
	Active    *bool
	Type      BomType
	Limit     int
	Offset    int
}

// ProductionFilter defines criteria for listing MOs.
type ProductionFilter struct {
	ProductID *int64
	State     ProductionState
	CompanyID *int64
	DateFrom  *time.Time
	DateTo    *time.Time
	Limit     int
	Offset    int
}
