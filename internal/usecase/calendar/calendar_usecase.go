package calendarusecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cashflow_backend/internal/domain/calendar"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo calendar.Repository
}

func New(repo calendar.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func (useCase *UseCase) CreateEvent(ctx context.Context, event *calendar.CalendarEvent) (*calendar.CalendarEvent, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.ensureNoConflict(ctx, event.UserID, event.Start, event.Stop, 0); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (useCase *UseCase) GetEvent(ctx context.Context, id int64) (*calendar.CalendarEvent, error) {
	return useCase.repo.GetEvent(ctx, id)
}

func (useCase *UseCase) UpdateEvent(ctx context.Context, event *calendar.CalendarEvent) (*calendar.CalendarEvent, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.ensureNoConflict(ctx, event.UserID, event.Start, event.Stop, event.ID); err != nil {
		return nil, err
	}
	if err := useCase.repo.UpdateEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (useCase *UseCase) DeleteEvent(ctx context.Context, id int64) error {
	return useCase.repo.DeleteEvent(ctx, id)
}

func (useCase *UseCase) ListEvents(ctx context.Context, filter calendar.EventFilter) ([]calendar.CalendarEvent, error) {
	return useCase.repo.ListEvents(ctx, filter)
}

func (useCase *UseCase) RespondAttendee(ctx context.Context, token string, status calendar.AttendeeStatus) (*calendar.Attendee, error) {
	if status != calendar.StatusAccepted && status != calendar.StatusDeclined && status != calendar.StatusTentative {
		return nil, platformerrors.Validation("invalid attendee response status", nil)
	}
	return useCase.repo.RespondAttendee(ctx, token, status)
}

func (useCase *UseCase) CreateAppointmentType(ctx context.Context, appointmentType *calendar.AppointmentType) (*calendar.AppointmentType, error) {
	if err := appointmentType.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateAppointmentType(ctx, appointmentType); err != nil {
		return nil, err
	}
	return appointmentType, nil
}

func (useCase *UseCase) ListAppointmentTypes(ctx context.Context, companyID int64) ([]calendar.AppointmentType, error) {
	return useCase.repo.ListAppointmentTypes(ctx, companyID)
}

func (useCase *UseCase) GetAppointmentTypeBySlug(ctx context.Context, slug string) (*calendar.AppointmentType, error) {
	return useCase.repo.GetAppointmentTypeBySlug(ctx, slug)
}

func (useCase *UseCase) AddAppointmentSlot(ctx context.Context, slot *calendar.AppointmentSlot) (*calendar.AppointmentSlot, error) {
	if err := slot.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateSlot(ctx, slot); err != nil {
		return nil, err
	}
	return slot, nil
}

func (useCase *UseCase) GetAvailableSlots(ctx context.Context, appointmentTypeID int64, from, to time.Time) ([]time.Time, error) {
	appointmentType, err := useCase.repo.GetAppointmentType(ctx, appointmentTypeID)
	if err != nil {
		return nil, err
	}
	slots, err := useCase.repo.ListSlots(ctx, appointmentTypeID)
	if err != nil {
		return nil, err
	}
	available := make([]time.Time, 0)
	for day := startOfDay(from); day.Before(to); day = day.AddDate(0, 0, 1) {
		for _, slot := range slots {
			if int(day.Weekday()) != slot.DayOfWeek {
				continue
			}
			for candidate := hourOnDay(day, slot.HourFrom); candidate.Add(time.Duration(appointmentType.DurationMinutes)*time.Minute).Compare(hourOnDay(day, slot.HourTo)) <= 0; candidate = candidate.Add(time.Duration(appointmentType.DurationMinutes) * time.Minute) {
				if candidate.Before(from) || candidate.Add(time.Duration(appointmentType.DurationMinutes)*time.Minute).After(to) {
					continue
				}
				free := true
				for _, staffID := range appointmentType.StaffUserIDs {
					if useCase.hasConflict(ctx, staffID, candidate, candidate.Add(time.Duration(appointmentType.DurationMinutes)*time.Minute), 0) == nil {
						free = true
						break
					}
					free = false
				}
				if free {
					available = append(available, candidate)
				}
			}
		}
	}
	sort.Slice(available, func(left, right int) bool { return available[left].Before(available[right]) })
	return available, nil
}

func (useCase *UseCase) BookAppointment(ctx context.Context, slug string, booking *calendar.AppointmentBooking) (*calendar.AppointmentBooking, error) {
	appointmentType, err := useCase.repo.GetAppointmentTypeBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	booking.AppointmentTypeID = appointmentType.ID
	booking.EndTime = booking.StartTime.Add(time.Duration(appointmentType.DurationMinutes) * time.Minute)
	if booking.StartTime.Before(time.Now().UTC().Add(time.Duration(appointmentType.MinScheduleHours)*time.Hour)) || booking.StartTime.After(time.Now().UTC().AddDate(0, 0, appointmentType.MaxScheduleDays)) {
		return nil, platformerrors.Validation("booking is outside the allowed scheduling window", nil)
	}
	if booking.StaffID == 0 {
		booking.StaffID = appointmentType.StaffUserIDs[0]
	}
	if !containsID(appointmentType.StaffUserIDs, booking.StaffID) {
		return nil, platformerrors.Validation("selected staff member is not assigned to this appointment type", nil)
	}
	if err := booking.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.ensureNoConflict(ctx, booking.StaffID, booking.StartTime, booking.EndTime, 0); err != nil {
		return nil, err
	}
	event := &calendar.CalendarEvent{Name: appointmentType.Name, Start: booking.StartTime, Stop: booking.EndTime, UserID: booking.StaffID, CompanyID: appointmentType.CompanyID, Location: appointmentType.Location}
	createdEvent, err := useCase.CreateEvent(ctx, event)
	if err != nil {
		return nil, err
	}
	booking.EventID = createdEvent.ID
	if err := useCase.repo.CreateBooking(ctx, booking); err != nil {
		return nil, err
	}
	return booking, nil
}

func (useCase *UseCase) ensureNoConflict(ctx context.Context, userID int64, start, stop time.Time, ignoredID int64) error {
	events, err := useCase.repo.ListEvents(ctx, calendar.EventFilter{UserID: &userID, From: &start, To: &stop})
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.ID != ignoredID && event.Overlaps(start, stop) {
			return platformerrors.Conflict(fmt.Sprintf("user %d already has an event in this time range", userID))
		}
	}
	return nil
}

func (useCase *UseCase) hasConflict(ctx context.Context, userID int64, start, stop time.Time, ignoredID int64) error {
	return useCase.ensureNoConflict(ctx, userID, start, stop, ignoredID)
}

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func hourOnDay(day time.Time, hour float64) time.Time {
	hours := int(hour)
	minutes := int((hour - float64(hours)) * 60)
	return time.Date(day.Year(), day.Month(), day.Day(), hours, minutes, 0, 0, day.Location())
}

func containsID(values []int64, wanted int64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
