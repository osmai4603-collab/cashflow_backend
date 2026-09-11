package calendarstorage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cashflow_backend/internal/domain/calendar"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepo struct {
	mu               sync.RWMutex
	events           map[int64]*calendar.CalendarEvent
	appointmentTypes map[int64]*calendar.AppointmentType
	slots            map[int64]*calendar.AppointmentSlot
	bookings         map[int64]*calendar.AppointmentBooking
	nextEventID      int64
	nextTypeID       int64
	nextSlotID       int64
	nextBookingID    int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		events:           make(map[int64]*calendar.CalendarEvent),
		appointmentTypes: make(map[int64]*calendar.AppointmentType),
		slots:            make(map[int64]*calendar.AppointmentSlot),
		bookings:         make(map[int64]*calendar.AppointmentBooking),
	}
}

func (repo *MemoryRepo) CreateEvent(ctx context.Context, event *calendar.CalendarEvent) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextEventID++
	event.ID = repo.nextEventID
	clone := *event
	repo.events[event.ID] = &clone
	return nil
}

func (repo *MemoryRepo) GetEvent(ctx context.Context, id int64) (*calendar.CalendarEvent, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	event, ok := repo.events[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("calendar event with ID %d not found", id))
	}
	clone := *event
	return &clone, nil
}

func (repo *MemoryRepo) UpdateEvent(ctx context.Context, event *calendar.CalendarEvent) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, ok := repo.events[event.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("calendar event with ID %d not found", event.ID))
	}
	clone := *event
	repo.events[event.ID] = &clone
	return nil
}

func (repo *MemoryRepo) DeleteEvent(ctx context.Context, id int64) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, ok := repo.events[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("calendar event with ID %d not found", id))
	}
	delete(repo.events, id)
	return nil
}

func (repo *MemoryRepo) ListEvents(ctx context.Context, filter calendar.EventFilter) ([]calendar.CalendarEvent, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]calendar.CalendarEvent, 0)
	for _, event := range repo.events {
		if filter.UserID != nil && event.UserID != *filter.UserID {
			continue
		}
		if filter.From != nil && !event.Stop.After(*filter.From) {
			continue
		}
		if filter.To != nil && !event.Start.Before(*filter.To) {
			continue
		}
		result = append(result, *event)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Start.Before(result[right].Start) })
	return result, nil
}

func (repo *MemoryRepo) RespondAttendee(ctx context.Context, token string, status calendar.AttendeeStatus) (*calendar.Attendee, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, event := range repo.events {
		for index := range event.Attendees {
			if event.Attendees[index].Token == token {
				event.Attendees[index].Status = status
				attendee := event.Attendees[index]
				return &attendee, nil
			}
		}
	}
	return nil, platformerrors.NotFound("attendee token not found")
}

func (repo *MemoryRepo) CreateAppointmentType(ctx context.Context, appointmentType *calendar.AppointmentType) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, existing := range repo.appointmentTypes {
		if existing.Slug == appointmentType.Slug {
			return platformerrors.Conflict("appointment type slug already exists")
		}
	}
	repo.nextTypeID++
	appointmentType.ID = repo.nextTypeID
	clone := *appointmentType
	repo.appointmentTypes[appointmentType.ID] = &clone
	return nil
}

func (repo *MemoryRepo) GetAppointmentTypeBySlug(ctx context.Context, slug string) (*calendar.AppointmentType, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	for _, appointmentType := range repo.appointmentTypes {
		if appointmentType.Slug == slug {
			clone := *appointmentType
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("appointment type not found")
}

func (repo *MemoryRepo) GetAppointmentType(ctx context.Context, id int64) (*calendar.AppointmentType, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	appointmentType, ok := repo.appointmentTypes[id]
	if !ok {
		return nil, platformerrors.NotFound("appointment type not found")
	}
	clone := *appointmentType
	return &clone, nil
}

func (repo *MemoryRepo) ListAppointmentTypes(ctx context.Context, companyID int64) ([]calendar.AppointmentType, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]calendar.AppointmentType, 0)
	for _, appointmentType := range repo.appointmentTypes {
		if appointmentType.CompanyID == companyID {
			result = append(result, *appointmentType)
		}
	}
	return result, nil
}

func (repo *MemoryRepo) CreateBooking(ctx context.Context, booking *calendar.AppointmentBooking) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, existing := range repo.bookings {
		if existing.StaffID == booking.StaffID && existing.Status != "cancelled" && existing.StartTime.Before(booking.EndTime) && booking.StartTime.Before(existing.EndTime) {
			return platformerrors.Conflict("staff member is already booked for this time")
		}
	}
	repo.nextBookingID++
	booking.ID = repo.nextBookingID
	clone := *booking
	repo.bookings[booking.ID] = &clone
	return nil
}

func (repo *MemoryRepo) ListBookings(ctx context.Context, staffID int64, from, to time.Time) ([]calendar.AppointmentBooking, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]calendar.AppointmentBooking, 0)
	for _, booking := range repo.bookings {
		if booking.StaffID == staffID && booking.EndTime.After(from) && booking.StartTime.Before(to) {
			result = append(result, *booking)
		}
	}
	return result, nil
}

func (repo *MemoryRepo) CreateSlot(ctx context.Context, slot *calendar.AppointmentSlot) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextSlotID++
	slot.ID = repo.nextSlotID
	clone := *slot
	repo.slots[slot.ID] = &clone
	return nil
}

func (repo *MemoryRepo) ListSlots(ctx context.Context, appointmentTypeID int64) ([]calendar.AppointmentSlot, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]calendar.AppointmentSlot, 0)
	for _, slot := range repo.slots {
		if slot.AppointmentTypeID == appointmentTypeID {
			result = append(result, *slot)
		}
	}
	return result, nil
}
