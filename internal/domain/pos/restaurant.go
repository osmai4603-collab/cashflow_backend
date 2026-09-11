package pos

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type RestaurantFloor struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	PosConfigID int64             `json:"pos_config_id"`
	Sequence    int               `json:"sequence"`
	Tables      []RestaurantTable `json:"tables,omitempty"`
	CompanyID   int64             `json:"company_id"`
	Active      bool              `json:"active"`
}

type RestaurantTable struct {
	ID        int64   `json:"id"`
	FloorID   int64   `json:"floor_id"`
	Name      string  `json:"name"`
	Seats     int     `json:"seats"`
	Shape     string  `json:"shape"`
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Active    bool    `json:"active"`
	CompanyID int64   `json:"company_id"`
}

func (table *RestaurantTable) Validate() error {
	table.Name = strings.TrimSpace(table.Name)
	if table.FloorID <= 0 || table.Name == "" || table.Seats <= 0 || table.CompanyID <= 0 {
		return platformerrors.Validation("restaurant table requires floor, name, seats, and company", nil)
	}
	if table.Shape == "" {
		table.Shape = "square"
	}
	return nil
}

type KitchenOrderStatus string

const (
	KitchenOrderPending   KitchenOrderStatus = "pending"
	KitchenOrderPreparing KitchenOrderStatus = "preparing"
	KitchenOrderReady     KitchenOrderStatus = "ready"
	KitchenOrderServed    KitchenOrderStatus = "served"
)

type KitchenOrderLine struct {
	ID          int64   `json:"id"`
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Qty         float64 `json:"qty"`
	Notes       string  `json:"notes,omitempty"`
}

type KitchenOrderTicket struct {
	ID      int64              `json:"id"`
	OrderID int64              `json:"order_id"`
	TableID *int64             `json:"table_id,omitempty"`
	Status  KitchenOrderStatus `json:"status"`
	Course  string             `json:"course"`
	Notes   string             `json:"notes,omitempty"`
	Lines   []KitchenOrderLine `json:"lines"`
}
