package recruitment

import "cashflow_backend/internal/platform/errors"

type RecruitmentStage struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Sequence  int    `json:"sequence"`
	Folded    bool   `json:"folded"`
	CompanyID int64  `json:"company_id"`
}

func (stage *RecruitmentStage) Validate() error {
	if stage.Name == "" || stage.CompanyID <= 0 || stage.Sequence < 0 {
		return errors.Validation("recruitment stage requires name, company, and valid sequence", nil)
	}
	return nil
}
