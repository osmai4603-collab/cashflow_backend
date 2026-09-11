package repair

import (
	"context"
)

type Repository interface {
	CreateOrder(ctx context.Context, order *RepairOrder) error
	GetOrderByID(ctx context.Context, id int64) (*RepairOrder, error)
	GetOrderByName(ctx context.Context, name string) (*RepairOrder, error)
	UpdateOrder(ctx context.Context, order *RepairOrder) error
	DeleteOrder(ctx context.Context, id int64) error
	ListOrders(ctx context.Context, companyID int64) ([]RepairOrder, error)

	CreateOrderLine(ctx context.Context, line *RepairOrderLine) error
	GetLinesByRepairID(ctx context.Context, repairID int64) ([]RepairOrderLine, error)
	DeleteOrderLine(ctx context.Context, id int64) error
}

type StockPort interface {
	ConsumeStockForRepair(ctx context.Context, locationID int64, productID int64, qty float64) error
}

type InvoicePort interface {
	CreateRepairInvoice(ctx context.Context, order *RepairOrder) (int64, error)
}
