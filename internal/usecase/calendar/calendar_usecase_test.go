package calendarusecase

import (
	"context"
	"testing"
	"time"

	calendarstorage "cashflow_backend/internal/adapters/storage/calendar"
	"cashflow_backend/internal/domain/calendar"
)

func TestBookAppointmentRejectsOverlappingBooking(t *testing.T) {
	repo := calendarstorage.NewMemoryRepo()
	useCase := New(repo)
	appointmentType := &calendar.AppointmentType{Name: "Demo", Slug: "demo", DurationMinutes: 30, MinScheduleHours: 0, MaxScheduleDays: 30, CompanyID: 1, StaffUserIDs: []int64{7}}
	if _, err := useCase.CreateAppointmentType(context.Background(), appointmentType); err != nil {
		t.Fatalf("create appointment type: %v", err)
	}
	start := time.Now().UTC().Add(2 * time.Hour)
	first := &calendar.AppointmentBooking{StaffID: 7, CustomerName: "First", CustomerEmail: "first@example.com", StartTime: start}
	if _, err := useCase.BookAppointment(context.Background(), "demo", first); err != nil {
		t.Fatalf("book first appointment: %v", err)
	}
	second := &calendar.AppointmentBooking{StaffID: 7, CustomerName: "Second", CustomerEmail: "second@example.com", StartTime: start.Add(10 * time.Minute)}
	if _, err := useCase.BookAppointment(context.Background(), "demo", second); err == nil {
		t.Fatal("expected overlapping appointment to fail")
	}
}

func TestGetAvailableSlotsUsesConfiguredWorkingHours(t *testing.T) {
	repo := calendarstorage.NewMemoryRepo()
	useCase := New(repo)
	appointmentType := &calendar.AppointmentType{Name: "Demo", Slug: "demo", DurationMinutes: 30, MinScheduleHours: 0, MaxScheduleDays: 30, CompanyID: 1, StaffUserIDs: []int64{7}}
	if _, err := useCase.CreateAppointmentType(context.Background(), appointmentType); err != nil {
		t.Fatalf("create appointment type: %v", err)
	}
	day := time.Now().UTC().AddDate(0, 0, 2)
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	if _, err := useCase.AddAppointmentSlot(context.Background(), &calendar.AppointmentSlot{AppointmentTypeID: appointmentType.ID, DayOfWeek: int(day.Weekday()), HourFrom: 9, HourTo: 10}); err != nil {
		t.Fatalf("create slot: %v", err)
	}
	slots, err := useCase.GetAvailableSlots(context.Background(), appointmentType.ID, day, day.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("get slots: %v", err)
	}
	if len(slots) != 2 || slots[0].Hour() != 9 || slots[1].Hour() != 9 || slots[1].Minute() != 30 {
		t.Fatalf("unexpected available slots: %v", slots)
	}
}
