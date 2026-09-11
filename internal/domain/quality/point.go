package quality

type QualityPointTrigger string

const (
	TriggerReceipt  QualityPointTrigger = "receipt"
	TriggerDelivery QualityPointTrigger = "delivery"
	TriggerMRP      QualityPointTrigger = "mrp"
)

type QualityControlPoint struct {
	ID           int64               `json:"id"`
	Name         string              `json:"name"`
	ProductID    *int64              `json:"product_id,omitempty"`
	CategoryID   *int64              `json:"category_id,omitempty"`
	Trigger      QualityPointTrigger `json:"trigger"`
	TestType     string              `json:"test_type"` // e.g. "pass_fail", "measure"
	NormMin      *float64            `json:"norm_min,omitempty"`
	NormMax      *float64            `json:"norm_max,omitempty"`
	Instructions string              `json:"instructions,omitempty"`
	CompanyID    int64               `json:"company_id"`
	Active       bool                `json:"active"`
}

func (q *QualityControlPoint) Validate() error {
	if q.Name == "" {
		return errInvalid("quality control point name cannot be empty")
	}
	if q.Trigger == "" {
		return errInvalid("trigger is required")
	}
	if q.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	if q.NormMin != nil && q.NormMax != nil && *q.NormMin > *q.NormMax {
		return errInvalid("norm_min cannot be greater than norm_max")
	}
	return nil
}
