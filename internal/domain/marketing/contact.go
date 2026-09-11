package marketing

import "time"

type Contact struct {
	ID          int64     `json:"id"`
	PartnerID   *int64    `json:"partner_id,omitempty"`
	Email       string    `json:"email"`
	Mobile      string    `json:"mobile,omitempty"`
	Name        string    `json:"name"`
	IsOptOut    bool      `json:"is_opt_out"`
	IsBlacklist bool      `json:"is_blacklist"`
	CompanyID   int64     `json:"company_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MailingListContactRel struct {
	ListID    int64 `json:"list_id"`
	ContactID int64 `json:"contact_id"`
}
