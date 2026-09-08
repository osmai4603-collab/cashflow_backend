package stock

import (
	"context"
	"time"

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
	ReserveMove(ctx context.Context, move *StockMove) error
	CreateMoveLine(ctx context.Context, line *StockMoveLine) error
	GetMoveLineByID(ctx context.Context, id int64) (*StockMoveLine, error)
	ListMoveLinesByMoveID(ctx context.Context, moveID int64) ([]StockMoveLine, error)
	UpdateMoveLine(ctx context.Context, line *StockMoveLine) error
	DeleteMoveLine(ctx context.Context, id int64) error
	CreateLot(ctx context.Context, lot *StockLot) error
	GetLotByID(ctx context.Context, id int64) (*StockLot, error)
	ListLotsByProduct(ctx context.Context, productID int64) ([]StockLot, error)
	UpdateLot(ctx context.Context, lot *StockLot) error

	// Quants & Balances
	GetQuant(ctx context.Context, productID, locationID int64) (*StockQuant, error)
	// ReserveQuantity atomically reserves available stock and returns the quantity reserved.
	ReserveQuantity(ctx context.Context, productID, locationID int64, quantity float64) (float64, error)
	UpdateQuantQuantity(ctx context.Context, productID, locationID int64, deltaQty float64) error
	SetQuantQuantity(ctx context.Context, productID, locationID int64, newQty float64) error
	ListQuants(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[StockQuant], error)
	GetOnHandStock(ctx context.Context, productID *int64, locationID *int64, warehouseID *int64) ([]StockOnHandItem, error)

	// Atomic Execution
	ValidateMovesTx(ctx context.Context, moves []StockMove) error
	ValidatePickingTx(ctx context.Context, picking *StockPicking) error

	// Valuations (Phase 12 — stock-account integration)
	UpdateMoveValue(ctx context.Context, move *StockMove) error
	GetFIFOStack(ctx context.Context, productID, companyID int64) (FIFOStack, error)
	CreateProductValue(ctx context.Context, pv *ProductValue) error
	ListProductValues(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[ProductValue], error)
	ComputeTotalValuation(ctx context.Context, productID *int64, locationID *int64) ([]ValuationSummary, error)

	// Accounting Periods (periodic closing valuation)
	CreateAccountingPeriod(ctx context.Context, p *AccountingPeriod) error
	GetAccountingPeriodByID(ctx context.Context, id int64) (*AccountingPeriod, error)
	ListAccountingPeriods(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[AccountingPeriod], error)
	CloseAccountingPeriod(ctx context.Context, p *AccountingPeriod) error

	// Reorder rules (Phase 13 — stock.orderpoint)
	CreateOrderpoint(ctx context.Context, op *Orderpoint) error
	GetOrderpointByID(ctx context.Context, id int64) (*Orderpoint, error)
	UpdateOrderpoint(ctx context.Context, op *Orderpoint) error
	DeleteOrderpoint(ctx context.Context, id int64) error
	ListOrderpoints(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Orderpoint], error)
	ListOrderpointsForReplenishment(ctx context.Context, now time.Time) ([]*Orderpoint, error)
	StockForecast(ctx context.Context, productID, locationID int64, at time.Time) (onHand, incoming, outgoing float64, err error)

	// Landed costs (Phase 13 — stock.landed.cost)
	CreateLandedCost(ctx context.Context, lc *LandedCost) error
	GetLandedCostByID(ctx context.Context, id int64) (*LandedCost, error)
	UpdateLandedCost(ctx context.Context, lc *LandedCost) error
	ListLandedCosts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[LandedCost], error)
	DeleteLandedCost(ctx context.Context, id int64) error

	// Procurement Groups (Phase 14 — sale_stock/purchase_stock)
	CreateProcurementGroup(ctx context.Context, pg *ProcurementGroup) error
	GetProcurementGroupByID(ctx context.Context, id int64) (*ProcurementGroup, error)
	GetProcurementGroupByName(ctx context.Context, name string) (*ProcurementGroup, error)

	// Routes & Rules (Odoo 19 Parity)
	CreateRoute(ctx context.Context, r *StockRoute) error
	GetRouteByID(ctx context.Context, id int64) (*StockRoute, error)
	ListRoutes(ctx context.Context, companyID *int64) ([]StockRoute, error)
	CreateRule(ctx context.Context, r *StockRule) error
	GetRuleByID(ctx context.Context, id int64) (*StockRule, error)
	ListRulesByRoute(ctx context.Context, routeID int64) ([]StockRule, error)
	FindRule(ctx context.Context, routeID int64, locationDestID int64) (*StockRule, error)

	// Scrap
	CreateScrap(ctx context.Context, s *StockScrap) error
	GetScrapByID(ctx context.Context, id int64) (*StockScrap, error)
	UpdateScrap(ctx context.Context, s *StockScrap) error
}
