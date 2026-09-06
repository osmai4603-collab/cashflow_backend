package stock

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines storage ports for the complete stock management domain.
type Repository interface {
	// Locations
	CreateLocation(ctx context.Context, loc *StockLocation) error
	GetLocationByID(ctx context.Context, id int64) (*StockLocation, error)
	GetLocationByName(ctx context.Context, name string) (*StockLocation, error)
	GetLocationByUsage(ctx context.Context, usage LocationUsage) (*StockLocation, error)
	UpdateLocation(ctx context.Context, loc *StockLocation) error
	DeleteLocation(ctx context.Context, id int64) error
	ListLocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[StockLocation], error)

	// Warehouses
	CreateWarehouse(ctx context.Context, wh *Warehouse) error
	GetWarehouseByID(ctx context.Context, id int64) (*Warehouse, error)
	GetWarehouseByCode(ctx context.Context, code string) (*Warehouse, error)
	UpdateWarehouse(ctx context.Context, wh *Warehouse) error
	DeleteWarehouse(ctx context.Context, id int64) error
	ListWarehouses(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Warehouse], error)
	GetDefaultWarehouse(ctx context.Context) (*Warehouse, error)

	// Pickings (Receipts, Deliveries, Internal Transfers)
	CreatePicking(ctx context.Context, picking *StockPicking) error
	GetPickingByID(ctx context.Context, id int64) (*StockPicking, error)
	GetPickingByName(ctx context.Context, name string) (*StockPicking, error)
	UpdatePicking(ctx context.Context, picking *StockPicking) error
	DeletePicking(ctx context.Context, id int64) error
	ListPickings(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[StockPicking], error)
	NextSequence(ctx context.Context, pickingType PickingType, year int) (string, error)

	// Moves
	CreateMove(ctx context.Context, move *StockMove) error
	GetMoveByID(ctx context.Context, id int64) (*StockMove, error)
	UpdateMove(ctx context.Context, move *StockMove) error
	ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[StockMove], error)
	GetMovesByPickingID(ctx context.Context, pickingID int64) ([]StockMove, error)

	// Quants & Balances
	GetQuant(ctx context.Context, productID, locationID int64) (*StockQuant, error)
	UpdateQuantQuantity(ctx context.Context, productID, locationID int64, deltaQty float64) error
	SetQuantQuantity(ctx context.Context, productID, locationID int64, newQty float64) error
	ListQuants(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[StockQuant], error)
	GetOnHandStock(ctx context.Context, productID *int64, locationID *int64, warehouseID *int64) ([]StockOnHandItem, error)

	// Atomic Execution
	ValidatePickingTx(ctx context.Context, picking *StockPicking) error
}
