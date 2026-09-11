package mrp

import platformerrors "cashflow_backend/internal/platform/errors"

type QualityCheckType string

const (
	QualityCheckPassFail QualityCheckType = "pass_fail"
	QualityCheckMeasure  QualityCheckType = "measure"
	QualityCheckText     QualityCheckType = "text"
	QualityCheckPicture  QualityCheckType = "picture"
)

type QualityPoint struct {
	ID           int64            `json:"id"`
	Name         string           `json:"name"`
	ProductID    *int64           `json:"product_id,omitempty"`
	OperationID  *int64           `json:"operation_id,omitempty"`
	WorkcenterID *int64           `json:"workcenter_id,omitempty"`
	CheckType    QualityCheckType `json:"check_type"`
	NormMin      *float64         `json:"norm_min,omitempty"`
	NormMax      *float64         `json:"norm_max,omitempty"`
	Instructions string           `json:"instructions,omitempty"`
	CompanyID    int64            `json:"company_id"`
	Active       bool             `json:"active"`
}

type QualityCheck struct {
	ID           int64   `json:"id"`
	PointID      int64   `json:"point_id"`
	WorkorderID  *int64  `json:"workorder_id,omitempty"`
	ProductionID int64   `json:"production_id"`
	ProductID    int64   `json:"product_id"`
	Result       string  `json:"result"`
	MeasureValue *float64 `json:"measure_value,omitempty"`
	Note         string  `json:"note,omitempty"`
	State        string  `json:"state"`
	CompanyID    int64   `json:"company_id"`
}

func (p *QualityPoint) Validate() error {
	if p.Name == "" { return platformerrors.Validation("quality point name is required", nil) }
	switch p.CheckType { case QualityCheckPassFail, QualityCheckMeasure, QualityCheckText, QualityCheckPicture: default: return platformerrors.Validation("invalid quality check type", nil) }
	if p.NormMin != nil && p.NormMax != nil && *p.NormMin > *p.NormMax { return platformerrors.Validation("minimum norm cannot exceed maximum norm", nil) }
	return nil
}

func (c *QualityCheck) Evaluate() error {
	if c.State != "none" && c.State != "pass" && c.State != "fail" { return platformerrors.Validation("invalid quality check state", nil) }
	if c.State == "none" { return platformerrors.Validation("quality check has no result", nil) }
	if c.Result == "" { return platformerrors.Validation("quality check result is required", nil) }
	return nil
}