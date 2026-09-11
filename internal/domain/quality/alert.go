package quality

import (
	"time"
)

type AlertStage string

const (
	StageNew        AlertStage = "new"
	StageConfirmed  AlertStage = "confirmed"
	StageInProgress AlertStage = "in_progress"
	StageResolved   AlertStage = "resolved"
)

type QualityAlert struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	ProductID   int64      `json:"product_id"`
	LotID       *int64     `json:"lot_id,omitempty"`
	PickingID   *int64     `json:"picking_id,omitempty"`
	Description string     `json:"description"`
	ActionTaken *string    `json:"action_taken,omitempty"`
	Stage       AlertStage `json:"stage"`
	CompanyID   int64      `json:"company_id"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (a *QualityAlert) Validate() error {
	if a.Name == "" {
		return errInvalid("alert name cannot be empty")
	}
	if a.ProductID <= 0 {
		return errInvalid("product ID is required")
	}
	if a.Description == "" {
		return errInvalid("description is required")
	}
	if a.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}

func (a *QualityAlert) Resolve(action string) {
	a.Stage = StageResolved
	a.ActionTaken = &action
}
