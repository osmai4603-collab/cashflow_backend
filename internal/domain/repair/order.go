package repair

import (
	"time"
)

type RepairState string

const (
	StateDraft        RepairState = "draft"
	StateConfirmed    RepairState = "confirmed"
	StateUnderRepair  RepairState = "under_repair"
	StateDone         RepairState = "done"
	StateCancel       RepairState = "cancel"
)

type RepairOrder struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	PartnerID      int64             `json:"partner_id"`
	ProductID      int64             `json:"product_id"`
	ProductLotID   *int64            `json:"product_lot_id,omitempty"`
	WarrantyCheck  bool              `json:"warranty_check"`
	State          RepairState       `json:"state"`
	LocationID     int64             `json:"location_id"`
	LocationDestID int64             `json:"location_dest_id"`
	AmountTotal    float64           `json:"amount_total"`
	AccountMoveID  *int64            `json:"account_move_id,omitempty"`
	CompanyID      int64             `json:"company_id"`
	Lines          []RepairOrderLine `json:"lines,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (r *RepairOrder) Validate() error {
	if r.Name == "" {
		return errInvalid("repair order name cannot be empty")
	}
	if r.PartnerID <= 0 {
		return errInvalid("partner ID is required")
	}
	if r.ProductID <= 0 {
		return errInvalid("product ID is required")
	}
	if r.LocationID <= 0 || r.LocationDestID <= 0 {
		return errInvalid("source and destination locations are required")
	}
	if r.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}

func (r *RepairOrder) CalculateTotal() {
	var total float64
	for _, line := range r.Lines {
		total += line.PriceTotal
	}
	r.AmountTotal = total
}
