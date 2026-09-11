package marketing

import "time"

type MailingList struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	IsPublic  bool      `json:"is_public"`
	CompanyID int64     `json:"company_id"`
	CreatedAt time.Time `json:"created_at"`
}
