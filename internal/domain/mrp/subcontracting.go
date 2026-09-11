package mrp

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type SubcontractingBom struct {
	BomID           int64   `json:"bom_id"`
	SubcontractorID int64   `json:"subcontractor_id"`
	LeadTime        int     `json:"lead_time"`
	CostPerUnit     float64 `json:"cost_per_unit"`
}

type SubcontractState string

const (
	SubcontractStateDraft    SubcontractState = "draft"
	SubcontractStatePurchase SubcontractState = "purchase"
	SubcontractStateSent     SubcontractState = "sent"
	SubcontractStateReceived SubcontractState = "received"
	SubcontractStateDone     SubcontractState = "done"
	SubcontractStateCancel   SubcontractState = "cancel"
)

type SubcontractingOrder struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	ProductionID    int64            `json:"production_id"`
	SubcontractorID int64            `json:"subcontractor_id"`
	PurchaseOrderID *int64           `json:"purchase_order_id,omitempty"`
	PickingOutID    *int64           `json:"picking_out_id,omitempty"`
	PickingInID     *int64           `json:"picking_in_id,omitempty"`
	State           SubcontractState `json:"state"`
	CompanyID       int64            `json:"company_id"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (o *SubcontractingOrder) Validate() error {
	if o.ProductionID <= 0 || o.SubcontractorID <= 0 {
		return platformerrors.Validation("production and subcontractor are required", nil)
	}
	if o.State == "" {
		o.State = SubcontractStateDraft
	}
	switch o.State {
	case SubcontractStateDraft, SubcontractStatePurchase, SubcontractStateSent, SubcontractStateReceived, SubcontractStateDone, SubcontractStateCancel:
		return nil
	default:
		return platformerrors.Validation("invalid subcontracting state", nil)
	}
}

func (o *SubcontractingOrder) ConfirmPurchase() error {
	if o.State != SubcontractStateDraft {
		return platformerrors.Validation("subcontracting order cannot be confirmed", nil)
	}
	o.State = SubcontractStatePurchase
	return nil
}

func (o *SubcontractingOrder) MarkSent() error {
	if o.State != SubcontractStatePurchase {
		return platformerrors.Validation("subcontracting materials cannot be sent", nil)
	}
	o.State = SubcontractStateSent
	return nil
}

func (o *SubcontractingOrder) MarkReceived() error {
	if o.State != SubcontractStateSent {
		return platformerrors.Validation("subcontracted product cannot be received", nil)
	}
	o.State = SubcontractStateReceived
	return nil
}

func (o *SubcontractingOrder) Finish() error {
	if o.State != SubcontractStateReceived {
		return platformerrors.Validation("subcontracting order cannot be finished", nil)
	}
	o.State = SubcontractStateDone
	return nil
}
