package stock

import (
	"context"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type ProcurementRequest struct {
	ProductID   int64          `json:"product_id"`
	Quantity    float64        `json:"quantity"`
	UoMID       int64          `json:"uom_id"`
	LocationID  int64          `json:"location_id"`
	Origin      string         `json:"origin"`
	GroupID     *int64         `json:"group_id,omitempty"`
	RouteIDs    []int64        `json:"route_ids,omitempty"`
	Values      map[string]any `json:"values,omitempty"`
	DatePlanned time.Time      `json:"date_planned"`
	CompanyID   int64          `json:"company_id"`
	WarehouseID *int64         `json:"warehouse_id,omitempty"`
}

func (r *ProcurementRequest) Validate() error {
	if r.ProductID <= 0 || r.Quantity <= 0 || r.LocationID <= 0 || r.CompanyID <= 0 {
		return platformerrors.Validation("procurement product, quantity, location and company are required", nil)
	}
	return nil
}

type ProcurementRuleRepository interface {
	ListRulesByRoute(ctx context.Context, routeID int64) ([]StockRule, error)
}
type PendingProcurementRepository interface {
	ListPendingProcurements(ctx context.Context, now time.Time) ([]ProcurementRequest, error)
}
type ProcurementAction func(context.Context, *ProcurementRequest, *StockRule) error

type ProcurementEngine interface {
	RunProcurement(context.Context, *ProcurementRequest) error
	RunScheduler(context.Context) error
	FindApplicableRule(context.Context, *ProcurementRequest) (*StockRule, error)
}

type DefaultProcurementEngine struct {
	Rules   ProcurementRuleRepository
	Pending PendingProcurementRepository
	Execute ProcurementAction
}

func (e *DefaultProcurementEngine) FindApplicableRule(ctx context.Context, request *ProcurementRequest) (*StockRule, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	for _, routeID := range request.RouteIDs {
		rules, err := e.Rules.ListRulesByRoute(ctx, routeID)
		if err != nil {
			return nil, err
		}
		for index := range rules {
			if rules[index].Active && (rules[index].WarehouseID == nil || request.WarehouseID == nil || *rules[index].WarehouseID == *request.WarehouseID) {
				return &rules[index], nil
			}
		}
	}
	return nil, platformerrors.NotFound("no applicable procurement rule", nil)
}

func (e *DefaultProcurementEngine) RunProcurement(ctx context.Context, request *ProcurementRequest) error {
	rule, err := e.FindApplicableRule(ctx, request)
	if err != nil {
		return err
	}
	if e.Execute == nil {
		return platformerrors.Internal("procurement action executor is not configured", nil)
	}
	return e.Execute(ctx, request, rule)
}

func (e *DefaultProcurementEngine) RunScheduler(ctx context.Context) error {
	if e.Pending == nil {
		return platformerrors.Internal("pending procurement repository is not configured", nil)
	}
	requests, err := e.Pending.ListPendingProcurements(ctx, time.Now())
	if err != nil {
		return err
	}
	for index := range requests {
		if err := e.RunProcurement(ctx, &requests[index]); err != nil {
			return err
		}
	}
	return nil
}
