package planning

type PlanningRole struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CompanyID int64  `json:"company_id"`
}

func (r *PlanningRole) Validate() error {
	if r.Name == "" {
		return errInvalid("planning role name cannot be empty")
	}
	if r.CompanyID <= 0 {
		return errInvalid("company ID is required")
	}
	return nil
}
