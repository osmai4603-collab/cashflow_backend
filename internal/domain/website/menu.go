package website

// Menu represents a navigation item.
type Menu struct {
	ID        int64  `json:"id"`
	WebsiteID int64  `json:"website_id"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Sequence  int    `json:"sequence"`
	NewWindow bool   `json:"new_window"`
	Children  []Menu `json:"children,omitempty"`
}
