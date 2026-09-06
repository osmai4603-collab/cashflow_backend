package httpadapter

// ServiceInfoResponse provides information about the running ERP backend service.
type ServiceInfoResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Status  string `json:"status"`
}
