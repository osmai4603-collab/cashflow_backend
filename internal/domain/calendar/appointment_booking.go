package calendar

import (
	"strings"
	"time"
)

type AppointmentBooking struct {
	ID                int64     `json:"id"`
	AppointmentTypeID int64     `json:"appointment_type_id"`
	EventID           int64     `json:"event_id"`
	StaffID           int64     `json:"staff_id"`
	CustomerName      string    `json:"customer_name"`
	CustomerEmail     string    `json:"customer_email"`
	CustomerPhone     string    `json:"customer_phone"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Notes             string    `json:"notes,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

func (booking *AppointmentBooking) Validate() error {
	booking.CustomerName = strings.TrimSpace(booking.CustomerName)
	booking.CustomerEmail = strings.TrimSpace(booking.CustomerEmail)
	if booking.AppointmentTypeID <= 0 || booking.StaffID <= 0 || booking.CustomerName == "" || booking.CustomerEmail == "" || booking.StartTime.IsZero() || !booking.EndTime.After(booking.StartTime) {
		return invalidCalendar("booking has missing or invalid fields")
	}
	if booking.Status == "" {
		booking.Status = "confirmed"
	}
	return nil
}
