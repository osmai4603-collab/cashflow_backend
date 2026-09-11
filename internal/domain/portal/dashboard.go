package portal

// DashboardSummary provides an overview for the customer self-service portal.
type DashboardSummary struct {
	OpenOrdersCount    int     `json:"open_orders_count"`
	PendingInvoices    int     `json:"pending_invoices"`
	TotalOutstanding   float64 `json:"total_outstanding"`
	LoyaltyPoints      float64 `json:"loyalty_points"`
	RecentOrders       []OrderSnippet `json:"recent_orders"`
	AvailableDocuments []DocSnippet   `json:"available_documents"`
}

type OrderSnippet struct {
	ID          int64   `json:"id"`
	Number      string  `json:"number"`
	Date        string  `json:"date"`
	Total       float64 `json:"total"`
	Status      string  `json:"status"`
}

type DocSnippet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}
