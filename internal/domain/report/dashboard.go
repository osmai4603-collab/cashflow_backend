package report

type DashboardKPI struct {
	Key    string      `json:"key"`
	Label  string      `json:"label"`
	Value  interface{} `json:"value"`
	Change float64     `json:"change"` // % change vs previous period
	Trend  string      `json:"trend"`  // up, down, stable
	Unit   string      `json:"unit"`   // currency, percentage, count
}

type DashboardData struct {
	Title  string                 `json:"title"`
	KPIs   []DashboardKPI         `json:"kpis"`
	Charts map[string]interface{} `json:"charts,omitempty"`
}
