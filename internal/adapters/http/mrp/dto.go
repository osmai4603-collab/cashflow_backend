package mrphttp

import (
	"cashflow_backend/internal/domain/mrp"
	mrpusecase "cashflow_backend/internal/usecase/mrp"
	"time"
)

type CreateWorkcenterRequest struct {
	Name           string  `json:"name"`
	Code           string  `json:"code"`
	Sequence       int     `json:"sequence"`
	CompanyID      int64   `json:"company_id"`
	TimeStart      float64 `json:"time_start"`
	TimeStop       float64 `json:"time_stop"`
	TimeEfficiency float64 `json:"time_efficiency"`
	Capacity       float64 `json:"capacity"`
	CostPerHour    float64 `json:"cost_per_hour"`
}

func (r CreateWorkcenterRequest) ToInput() mrpusecase.CreateWorkcenterInput {
	return mrpusecase.CreateWorkcenterInput{
		Name:           r.Name,
		Code:           r.Code,
		Sequence:       r.Sequence,
		CompanyID:      r.CompanyID,
		TimeStart:      r.TimeStart,
		TimeStop:       r.TimeStop,
		TimeEfficiency: r.TimeEfficiency,
		Capacity:       r.Capacity,
		CostPerHour:    r.CostPerHour,
	}
}

type CreateBoMRequest struct {
	Code           string                 `json:"code"`
	ProductID      int64                  `json:"product_id"`
	ProductQty     float64                `json:"product_qty"`
	UoMID          int64                  `json:"uom_id"`
	Type           mrp.BomType            `json:"type"`
	ReadyToProduce string                 `json:"ready_to_produce"`
	Consumption    string                 `json:"consumption"`
	CompanyID      int64                  `json:"company_id"`
	Lines          []CreateBomLineRequest `json:"lines"`
	Operations     []CreateOpRequest      `json:"operations"`
}

type CreateBomLineRequest struct {
	ProductID   int64   `json:"product_id"`
	Quantity    float64 `json:"quantity"`
	UoMID       int64   `json:"uom_id"`
	OperationID *int64  `json:"operation_id"`
	Sequence    int     `json:"sequence"`
}

type CreateOpRequest struct {
	WorkcenterID    int64   `json:"workcenter_id"`
	Name            string  `json:"name"`
	Sequence        int     `json:"sequence"`
	TimeMode        string  `json:"time_mode"`
	TimeCycleManual float64 `json:"time_cycle_manual"`
}

func (r CreateBoMRequest) ToInput() mrpusecase.CreateBoMInput {
	lines := make([]mrpusecase.CreateBomLineInput, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = mrpusecase.CreateBomLineInput{
			ProductID:   l.ProductID,
			Quantity:    l.Quantity,
			UoMID:       l.UoMID,
			OperationID: l.OperationID,
			Sequence:    l.Sequence,
		}
	}
	ops := make([]mrpusecase.CreateOperationInput, len(r.Operations))
	for i, o := range r.Operations {
		ops[i] = mrpusecase.CreateOperationInput{
			WorkcenterID:    o.WorkcenterID,
			Name:            o.Name,
			Sequence:        o.Sequence,
			TimeMode:        o.TimeMode,
			TimeCycleManual: o.TimeCycleManual,
		}
	}
	return mrpusecase.CreateBoMInput{
		Code:           r.Code,
		ProductID:      r.ProductID,
		ProductQty:     r.ProductQty,
		UoMID:          r.UoMID,
		Type:           r.Type,
		ReadyToProduce: r.ReadyToProduce,
		Consumption:    r.Consumption,
		CompanyID:      r.CompanyID,
		Lines:          lines,
		Operations:     ops,
	}
}

type CreateProductionRequest struct {
	ProductID      int64   `json:"product_id"`
	ProductQty     float64 `json:"product_qty"`
	BomID          int64   `json:"bom_id"`
	DateDeadline   *string `json:"date_deadline"`
	DateStart      *string `json:"date_start"`
	CompanyID      int64   `json:"company_id"`
	PickingTypeID  int64   `json:"picking_type_id"`
	LocationSrcID  int64   `json:"location_src_id"`
	LocationDestID int64   `json:"location_dest_id"`
	Origin         string  `json:"origin"`
}

func (r CreateProductionRequest) ToInput() mrpusecase.CreateProductionInput {
	var deadline *time.Time
	if r.DateDeadline != nil {
		t, _ := time.Parse(time.RFC3339, *r.DateDeadline)
		deadline = &t
	}
	start := time.Now()
	if r.DateStart != nil {
		start, _ = time.Parse(time.RFC3339, *r.DateStart)
	}

	return mrpusecase.CreateProductionInput{
		ProductID:      r.ProductID,
		ProductQty:     r.ProductQty,
		BomID:          r.BomID,
		DateDeadline:   deadline,
		DateStart:      start,
		CompanyID:      r.CompanyID,
		PickingTypeID:  r.PickingTypeID,
		LocationSrcID:  r.LocationSrcID,
		LocationDestID: r.LocationDestID,
		Origin:         r.Origin,
	}
}

type CreateUnbuildRequest struct {
	ProductID      int64   `json:"product_id"`
	BomID          int64   `json:"bom_id"`
	MOID           *int64  `json:"mo_id"`
	Quantity       float64 `json:"quantity"`
	UoMID          int64   `json:"uom_id"`
	LocationID     int64   `json:"location_id"`
	DestLocationID int64   `json:"dest_location_id"`
	CompanyID      int64   `json:"company_id"`
}

func (r CreateUnbuildRequest) ToInput() mrpusecase.CreateUnbuildInput {
	return mrpusecase.CreateUnbuildInput{
		ProductID:      r.ProductID,
		BomID:          r.BomID,
		MOID:           r.MOID,
		Quantity:       r.Quantity,
		UoMID:          r.UoMID,
		LocationID:     r.LocationID,
		DestLocationID: r.DestLocationID,
		CompanyID:      r.CompanyID,
	}
}
