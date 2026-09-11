package calendar

import (
	"context"
	"time"
)

type EventFilter struct {
	UserID *int64
	From   *time.Time
	To     *time.Time
}

type Repository interface {
	CreateEvent(ctx context.Context, event *CalendarEvent) error
	GetEvent(ctx context.Context, id int64) (*CalendarEvent, error)
	UpdateEvent(ctx context.Context, event *CalendarEvent) error
	DeleteEvent(ctx context.Context, id int64) error
	ListEvents(ctx context.Context, filter EventFilter) ([]CalendarEvent, error)
	RespondAttendee(ctx context.Context, token string, status AttendeeStatus) (*Attendee, error)
	CreateAppointmentType(ctx context.Context, appointmentType *AppointmentType) error
	GetAppointmentType(ctx context.Context, id int64) (*AppointmentType, error)
	GetAppointmentTypeBySlug(ctx context.Context, slug string) (*AppointmentType, error)
	ListAppointmentTypes(ctx context.Context, companyID int64) ([]AppointmentType, error)
	CreateBooking(ctx context.Context, booking *AppointmentBooking) error
	ListBookings(ctx context.Context, staffID int64, from, to time.Time) ([]AppointmentBooking, error)
	CreateSlot(ctx context.Context, slot *AppointmentSlot) error
	ListSlots(ctx context.Context, appointmentTypeID int64) ([]AppointmentSlot, error)
}

type AvailabilityEngine interface {
	GetAvailableSlots(ctx context.Context, appointmentTypeID int64, from, to time.Time) ([]time.Time, error)
}
