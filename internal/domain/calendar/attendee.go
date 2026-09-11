package calendar

import (
	"net/mail"
	"strings"
)

type AttendeeStatus string

const (
	StatusNeedsAction AttendeeStatus = "needs_action"
	StatusAccepted    AttendeeStatus = "accepted"
	StatusDeclined    AttendeeStatus = "declined"
	StatusTentative   AttendeeStatus = "tentative"
)

type Attendee struct {
	ID        int64          `json:"id"`
	EventID   int64          `json:"event_id"`
	PartnerID *int64         `json:"partner_id,omitempty"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Status    AttendeeStatus `json:"status"`
	IsOwner   bool           `json:"is_owner"`
	Token     string         `json:"token"`
}

func (attendee *Attendee) Validate() error {
	attendee.Email = strings.TrimSpace(attendee.Email)
	attendee.Name = strings.TrimSpace(attendee.Name)
	if attendee.Name == "" || attendee.Email == "" {
		return invalidCalendar("attendee name and email are required")
	}
	if _, err := mail.ParseAddress(attendee.Email); err != nil {
		return invalidCalendar("attendee email is invalid")
	}
	if attendee.Status == "" {
		attendee.Status = StatusNeedsAction
	}
	return nil
}
