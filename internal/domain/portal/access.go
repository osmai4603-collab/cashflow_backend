package portal

// AccessRule defines which documents a portal user can see.
type AccessRule struct {
	Model      string `json:"model"`       // e.g., "sale.order", "account.move"
	FieldLink  string `json:"field_link"`  // e.g., "partner_id"
	AllowRead  bool   `json:"allow_read"`
	AllowWrite bool   `json:"allow_write"`
}

// CheckAccess verifies if the portal user has rights to a specific record.
func CheckAccess(user *User, model string, recordPartnerID int64) bool {
	// Simple rule: portal users only see records linked to their PartnerID
	return user.PartnerID == recordPartnerID
}
