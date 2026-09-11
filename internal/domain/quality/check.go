package quality

import (
	"time"
)

type QualityCheckState string

const (
	CheckStateDraft QualityCheckState = "draft"
	CheckStatePass  QualityCheckState = "pass"
	CheckStateFail  QualityCheckState = "fail"
)

type QualityCheck struct {
	ID           int64             `json:"id"`
	PointID      int64             `json:"point_id"`
	ProductID    int64             `json:"product_id"`
	LotID        *int64            `json:"lot_id,omitempty"`
	PickingID    *int64            `json:"picking_id,omitempty"`
	State        QualityCheckState `json:"state"`
	MeasureValue *float64          `json:"measure_value,omitempty"`
	Notes        string            `json:"notes,omitempty"`
	CheckedBy    int64             `json:"checked_by"`
	CompanyID    int64             `json:"company_id"`
	CreatedAt    time.Time         `json:"created_at"`
}

func (c *QualityCheck) PerformCheck(point *QualityControlPoint, value *float64) {
	if point.TestType == "measure" && value != nil {
		c.MeasureValue = value
		pass := true
		if point.NormMin != nil && *value < *point.NormMin {
			pass = false
		}
		if point.NormMax != nil && *value > *point.NormMax {
			pass = false
		}
		if pass {
			c.State = CheckStatePass
		} else {
			c.State = CheckStateFail
		}
	}
}
