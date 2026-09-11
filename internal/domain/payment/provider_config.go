package payment

type ProviderConfig struct {
	ID          int64  `json:"id"`
	ProviderID  int64  `json:"provider_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsSecret    bool   `json:"is_secret"`
	Environment string `json:"environment"`
	CompanyID   int64  `json:"company_id"`
}
