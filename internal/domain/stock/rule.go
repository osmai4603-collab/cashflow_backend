package stock

type ActionType string

const (
	ActionPull     ActionType = "pull"
	ActionPush     ActionType = "push"
	ActionPullPush ActionType = "pull_push"
	ActionBuy      ActionType = "buy"
	ActionManufacture ActionType = "manufacture"
)

type ProcurementMethod string

const (
	ProcureFromStock ProcurementMethod = "make_to_stock"
	ProcureFromRule  ProcurementMethod = "make_to_order"
)

type StockRule struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Action         ActionType        `json:"action"`
	RouteID        int64             `json:"route_id"`
	LocationSrcID  *int64            `json:"location_src_id,omitempty"`
	LocationDestID int64             `json:"location_dest_id"`
	PickingTypeID  int64             `json:"picking_type_id"`
	ProcureMethod  ProcurementMethod `json:"procure_method"`
	WarehouseID    *int64            `json:"warehouse_id,omitempty"`
	CompanyID      int64             `json:"company_id"`
	Sequence       int               `json:"sequence"`
	Active         bool              `json:"active"`
}

type StockRoute struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Sequence  int    `json:"sequence"`
	Active    bool   `json:"active"`
	CompanyID *int64 `json:"company_id,omitempty"`
}
