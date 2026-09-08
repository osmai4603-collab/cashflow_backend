package purchase

import (
	"fmt"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// RequisitionType indicates the type of blanket or template purchase agreement.
type RequisitionType string

const (
	RequisitionBlanketOrder RequisitionType = "blanket_order"
	RequisitionTemplate     RequisitionType = "purchase_template"
)

// RequisitionState represents the lifecycle of a purchase requisition.
type RequisitionState string

const (
	RequisitionDraft     RequisitionState = "draft"
	RequisitionConfirmed RequisitionState = "confirmed"
	RequisitionDone      RequisitionState = "done"
	RequisitionCancel    RequisitionState = "cancel"
)

// PurchaseRequisitionLine represents a single line inside a purchase requisition.
type PurchaseRequisitionLine struct {
	ID                         int64      `json:"id"`
	RequisitionID              int64      `json:"requisition_id"`
	ProductID                  int64      `json:"product_id"`
	ProductQty                 float64    `json:"product_qty"`
	ProductUOMID               *int64     `json:"product_uom_id,omitempty"`
	PriceUnit                  float64    `json:"price_unit"`
	QtyOrdered                 float64    `json:"qty_ordered"`
	ScheduleDate               *time.Time `json:"schedule_date,omitempty"`
	SupplierID                 *int64     `json:"supplier_id,omitempty"`
	SupplierInfoID             *int64     `json:"supplier_info_id,omitempty"`
	ProductDescriptionVariants string     `json:"product_description_variants,omitempty"`
	Description                string     `json:"description,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

// SupplierInfo is the vendor price created by a confirmed blanket order.
type SupplierInfo struct {
	ID            int64   `json:"id"`
	RequisitionID int64   `json:"requisition_id"`
	LineID        int64   `json:"line_id"`
	ProductID     int64   `json:"product_id"`
	VendorID      int64   `json:"vendor_id"`
	UOMID         *int64  `json:"uom_id,omitempty"`
	Price         float64 `json:"price"`
	CurrencyID    int64   `json:"currency_id"`
	Active        bool    `json:"active"`
}

// PurchaseRequisition models the Odoo purchase.requisition concept.
type PurchaseRequisition struct {
	ID               int64                     `json:"id"`
	Name             string                    `json:"name"`
	Active           bool                      `json:"active"`
	Reference        string                    `json:"reference,omitempty"`
	Type             RequisitionType           `json:"type"`
	VendorID         *int64                    `json:"vendor_id,omitempty"`
	UserID           int64                     `json:"user_id"`
	DateStart        *time.Time                `json:"date_start,omitempty"`
	DateEnd          *time.Time                `json:"date_end,omitempty"`
	State            RequisitionState          `json:"state"`
	CurrencyID       int64                     `json:"currency_id"`
	CompanyID        int64                     `json:"company_id"`
	Description      string                    `json:"description,omitempty"`
	OrderCount       int                       `json:"order_count"`
	PurchaseOrderIDs []int64                   `json:"purchase_order_ids,omitempty"`
	Lines            []PurchaseRequisitionLine `json:"lines,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

// Validate ensures the requisition has the required data to exist and transition.
func (r *PurchaseRequisition) Validate() error {
	if r.Type == "" {
		r.Type = RequisitionBlanketOrder
	}
	if r.Type != RequisitionBlanketOrder && r.Type != RequisitionTemplate {
		return platformerrors.Validation("invalid requisition type", map[string]string{
			"type": "must be blanket_order or purchase_template",
		})
	}
	if r.State == "" {
		r.State = RequisitionDraft
	}
	if r.State != RequisitionDraft && r.State != RequisitionConfirmed && r.State != RequisitionDone && r.State != RequisitionCancel {
		return platformerrors.Validation("invalid requisition state", map[string]string{
			"state": "must be draft, confirmed, done, or cancel",
		})
	}
	if len(r.Lines) == 0 {
		return platformerrors.Validation("requisition requires at least one line", map[string]string{
			"lines": "must contain at least one item",
		})
	}
	if r.DateStart != nil && r.DateEnd != nil && r.DateEnd.Before(*r.DateStart) {
		return platformerrors.Validation("requisition dates are invalid", map[string]string{
			"date_end": "must be greater than or equal to date_start",
		})
	}
	for i, line := range r.Lines {
		if line.ProductID <= 0 {
			return platformerrors.Validation("product is required for requisition line", map[string]string{
				fmt.Sprintf("lines[%d].product_id", i): "must reference a valid product",
			})
		}
		if line.ProductQty <= 0 {
			return platformerrors.Validation("quantity must be strictly positive", map[string]string{
				fmt.Sprintf("lines[%d].product_qty", i): "must be greater than 0",
			})
		}
		if line.PriceUnit < 0 {
			return platformerrors.Validation("unit price cannot be negative", map[string]string{
				fmt.Sprintf("lines[%d].price_unit", i): "must be non-negative",
			})
		}
	}
	return nil
}

// Confirm transitions the requisition to the confirmed state.
func (r *PurchaseRequisition) Confirm() error {
	if r.State != RequisitionDraft {
		return platformerrors.Conflict(fmt.Sprintf("cannot confirm requisition from state '%s'; must be draft", r.State))
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Type == RequisitionBlanketOrder {
		if r.VendorID == nil || *r.VendorID <= 0 {
			return platformerrors.Validation("blanket order vendor is required", map[string]string{
				"vendor_id": "must reference a valid supplier",
			})
		}
		for i, line := range r.Lines {
			if line.PriceUnit <= 0 {
				return platformerrors.Validation("blanket order price is required", map[string]string{
					fmt.Sprintf("lines[%d].price_unit", i): "must be greater than 0",
				})
			}
		}
	}
	r.State = RequisitionConfirmed
	return nil
}

// CanChangeDefinition reports whether fields that define the agreement may change.
func (r *PurchaseRequisition) CanChangeDefinition() error {
	if r.State != RequisitionDraft {
		return platformerrors.Conflict("requisition type and company cannot change after confirmation")
	}
	return nil
}

// Close transitions the requisition to done.
func (r *PurchaseRequisition) Close() error {
	if r.State != RequisitionConfirmed {
		return platformerrors.Conflict(fmt.Sprintf("cannot close requisition from state '%s'; must be confirmed", r.State))
	}
	r.State = RequisitionDone
	return nil
}

// Cancel transitions the requisition to cancelled.
func (r *PurchaseRequisition) Cancel() error {
	if r.State == RequisitionCancel {
		return nil
	}
	if r.State != RequisitionDraft && r.State != RequisitionConfirmed {
		return platformerrors.Conflict(fmt.Sprintf("cannot cancel requisition from state '%s'", r.State))
	}
	r.State = RequisitionCancel
	return nil
}
