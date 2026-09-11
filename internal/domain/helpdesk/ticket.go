package helpdesk

import (
	"fmt"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// Ticket represents a customer support request (helpdesk.ticket).
type Ticket struct {
	ID              int64      `json:"id"`
	Number          string     `json:"number"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	TeamID          int64      `json:"team_id"`
	StageID         int64      `json:"stage_id"`
	Priority        string     `json:"priority"` // "0", "1", "2", "3"
	PartnerID       *int64     `json:"partner_id,omitempty"`
	PartnerEmail    string     `json:"partner_email"`
	PartnerPhone    string     `json:"partner_phone,omitempty"`
	AssignedUserID  *int64     `json:"assigned_user_id,omitempty"`
	SaleOrderID     *int64     `json:"sale_order_id,omitempty"`
	StockPickingID  *int64     `json:"stock_picking_id,omitempty"`
	RepairOrderID   *int64     `json:"repair_order_id,omitempty"`
	FirstResponseAt *time.Time `json:"first_response_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	SLABreach       bool       `json:"sla_breach"`
	CompanyID       int64      `json:"company_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (t *Ticket) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("ticket subject is required", nil)
	}
	t.PartnerEmail = strings.TrimSpace(t.PartnerEmail)
	if t.PartnerEmail == "" {
		return platformerrors.Validation("partner email is required", nil)
	}
	if t.TeamID <= 0 {
		return platformerrors.Validation("team_id is required", nil)
	}
	if t.StageID <= 0 {
		return platformerrors.Validation("stage_id is required", nil)
	}
	return nil
}

// ActionAssign assigns the ticket to a user.
func (t *Ticket) ActionAssign(userID int64) {
	t.AssignedUserID = &userID
	t.UpdatedAt = time.Now().UTC()
}

// ActionRespond marks the first response time.
func (t *Ticket) ActionRespond() {
	if t.FirstResponseAt == nil {
		now := time.Now().UTC()
		t.FirstResponseAt = &now
	}
	t.UpdatedAt = time.Now().UTC()
}

// ActionClose marks the ticket as closed.
func (t *Ticket) ActionClose(closedStageID int64) {
	now := time.Now().UTC()
	t.StageID = closedStageID
	t.ClosedAt = &now
	t.UpdatedAt = now
}

// ThreadModel satisfies activity.Threadable.
func (t *Ticket) ThreadModel() string { return "helpdesk.ticket" }

// ThreadID satisfies activity.Threadable.
func (t *Ticket) ThreadID() int64 { return t.ID }

// ThreadCompanyID satisfies activity.Threadable.
func (t *Ticket) ThreadCompanyID() int64 { return t.CompanyID }

// GenerateNumber creates a unique ticket number.
func (t *Ticket) GenerateNumber(id int64) {
	t.Number = fmt.Sprintf("TK-%06d", id)
}
