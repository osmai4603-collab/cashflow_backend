package stock

import (
	"fmt"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// PickingType categorizes stock operations (incoming receipt, outgoing delivery, or internal transfer).
type PickingType string

const (
	PickingTypeIncoming PickingType = "incoming" // Vendor Receipt (WH/IN)
	PickingTypeOutgoing PickingType = "outgoing" // Customer Delivery (WH/OUT)
	PickingTypeInternal PickingType = "internal" // Internal Transfer (WH/INT)
)

// PickingState represents the lifecycle status of a stock picking document (stock.picking in Odoo).
type PickingState string

const (
	PickingStateDraft     PickingState = "draft"     // Draft picking
	PickingStateWaiting   PickingState = "waiting"   // Waiting another operation
	PickingStateConfirmed PickingState = "confirmed" // Waiting availability
	PickingStateAssigned  PickingState = "assigned"  // Ready to transfer (reserved)
	PickingStateDone      PickingState = "done"      // Successfully validated / transferred
	PickingStateCancel    PickingState = "cancel"    // Cancelled
)

// StockPicking represents a delivery order, receipt, or internal transfer document.
type StockPicking struct {
	ID                 int64        `json:"id"`
	Name               string       `json:"name"` // e.g. "WH/IN/2026/00001"
	PickingType        PickingType  `json:"picking_type"`
	State              PickingState `json:"state"`
	PartnerID          *int64       `json:"partner_id,omitempty"`
	LocationID         int64        `json:"location_id"`
	LocationDestID     int64        `json:"location_dest_id"`
	ScheduledDate      time.Time    `json:"scheduled_date"`
	DateDone           *time.Time   `json:"date_done,omitempty"`
	Origin             string       `json:"origin,omitempty"` // e.g. "SO/2026/00001", "PO/2026/00001"
	SourceOrderID      *int64       `json:"source_order_id,omitempty"`
	ProcurementGroupID *int64       `json:"procurement_group_id,omitempty"`
	CompanyID          *int64       `json:"company_id,omitempty"`
	Note               string       `json:"note,omitempty"`
	Active             bool         `json:"active"`
	Moves              []StockMove  `json:"moves,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
	CreatedBy          *int64       `json:"created_by,omitempty"`
	UpdatedBy          *int64       `json:"updated_by,omitempty"`
}

// Validate checks business invariants for StockPicking.
func (p *StockPicking) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = "/"
	}

	switch p.PickingType {
	case PickingTypeIncoming, PickingTypeOutgoing, PickingTypeInternal:
		// Valid type
	default:
		return platformerrors.Validation("invalid picking type", map[string]string{
			"picking_type": fmt.Sprintf("must be one of [%s, %s, %s]", PickingTypeIncoming, PickingTypeOutgoing, PickingTypeInternal),
		})
	}

	if p.LocationID <= 0 {
		return platformerrors.Validation("source location is required", map[string]string{
			"location_id": "must reference a valid source location",
		})
	}

	if p.LocationDestID <= 0 {
		return platformerrors.Validation("destination location is required", map[string]string{
			"location_dest_id": "must reference a valid destination location",
		})
	}

	if p.LocationID == p.LocationDestID {
		return platformerrors.Validation("source and destination locations cannot be identical", map[string]string{
			"location_dest_id": "must be different from source location",
		})
	}

	if p.State == "" {
		p.State = PickingStateDraft
	}

	switch p.State {
	case PickingStateDraft, PickingStateWaiting, PickingStateConfirmed, PickingStateAssigned, PickingStateDone, PickingStateCancel:
		// Valid state
	default:
		return platformerrors.Validation("invalid picking state", map[string]string{
			"state": fmt.Sprintf("invalid picking state '%s'", p.State),
		})
	}

	for i, m := range p.Moves {
		if err := m.Validate(); err != nil {
			return platformerrors.Validation(fmt.Sprintf("move line #%d is invalid: %v", i+1, err), nil)
		}
	}

	return nil
}

// ActionConfirm confirms the picking, ensures there are moves, and assigns the sequence number.
func (p *StockPicking) ActionConfirm(sequence string) error {
	if p.State != PickingStateDraft {
		return platformerrors.Conflict(fmt.Sprintf("cannot confirm picking in state '%s'; must be draft", p.State))
	}
	if len(p.Moves) == 0 {
		return platformerrors.Conflict("cannot confirm stock picking with no move lines")
	}

	if strings.TrimSpace(sequence) != "" {
		p.Name = strings.TrimSpace(sequence)
	}

	p.State = PickingStateConfirmed
	for i := range p.Moves {
		_ = p.Moves[i].ActionConfirm()
	}
	return nil
}

// ActionAssign marks stock as reserved and ready for transfer.
func (p *StockPicking) ActionAssign() error {
	if p.State != PickingStateConfirmed && p.State != PickingStateWaiting {
		return platformerrors.Conflict(fmt.Sprintf("cannot assign stock for picking in state '%s'", p.State))
	}
	p.State = PickingStateAssigned
	for i := range p.Moves {
		p.Moves[i].State = MoveStateAssigned
	}
	return nil
}

// ActionValidate validates and executes the stock transfer.
func (p *StockPicking) ActionValidate(effectiveDate time.Time) error {
	if p.State == PickingStateDone {
		return platformerrors.Conflict("stock picking is already validated")
	}
	if p.State == PickingStateCancel {
		return platformerrors.Conflict("cannot validate a cancelled stock picking")
	}
	if len(p.Moves) == 0 {
		return platformerrors.Conflict("cannot validate picking with no move lines")
	}

	if effectiveDate.IsZero() {
		effectiveDate = time.Now().UTC()
	}

	p.State = PickingStateDone
	p.DateDone = &effectiveDate

	for i := range p.Moves {
		qty := p.Moves[i].QuantityDone
		if qty <= 0 {
			qty = p.Moves[i].ProductQty
		}
		if err := p.Moves[i].ActionDone(qty); err != nil {
			return err
		}
	}

	return nil
}

// ActionCancel cancels the picking.
func (p *StockPicking) ActionCancel() error {
	if p.State == PickingStateDone {
		return platformerrors.Conflict("cannot cancel an already validated stock picking")
	}
	p.State = PickingStateCancel
	for i := range p.Moves {
		_ = p.Moves[i].ActionCancel()
	}
	return nil
}
