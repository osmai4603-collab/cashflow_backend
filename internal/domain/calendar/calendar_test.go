package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestRecurrenceOccurrencesRespectCountAndExceptions(t *testing.T) {
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	exception := start.AddDate(0, 0, 1)
	count := 4
	rule := RecurrenceRule{Freq: "DAILY", Count: &count, Exceptions: []time.Time{exception}}
	occurences, err := rule.Occurrences(start, start, start.AddDate(0, 0, 6))
	if err != nil {
		t.Fatalf("occurrences: %v", err)
	}
	if len(occurences) != 3 || occurences[1].Day() != 3 {
		t.Fatalf("unexpected occurrences: %v", occurences)
	}
}

func TestCalendarEventOverlapAndICSExport(t *testing.T) {
	start := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	event := &CalendarEvent{ID: 7, Name: "Planning", Start: start, Stop: start.Add(time.Hour), UserID: 1, CompanyID: 1}
	if err := event.Validate(); err != nil {
		t.Fatalf("validate event: %v", err)
	}
	if !event.Overlaps(start.Add(30*time.Minute), start.Add(90*time.Minute)) || event.Overlaps(start.Add(time.Hour), start.Add(2*time.Hour)) {
		t.Fatal("unexpected event overlap result")
	}
	ics, err := ExportICS(event)
	if err != nil {
		t.Fatalf("export ics: %v", err)
	}
	icsText := string(ics)
	if !strings.Contains(icsText, "BEGIN:VCALENDAR") || !strings.Contains(icsText, "SUMMARY:Planning") {
		t.Fatalf("invalid ICS output: %s", icsText)
	}
	parsed, err := ParseICS(ics)
	if err != nil || parsed.Name != event.Name || !parsed.Start.Equal(event.Start) {
		t.Fatalf("unexpected parsed ICS event: %+v, error=%v", parsed, err)
	}
}

func TestWeeklyRecurrenceSupportsByDay(t *testing.T) {
	start := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	rule := RecurrenceRule{Freq: "WEEKLY", Interval: 1, ByDay: []string{"MO", "WE"}}
	occurrences, err := rule.Occurrences(start, start, start.AddDate(0, 0, 7))
	if err != nil {
		t.Fatalf("occurrences: %v", err)
	}
	if len(occurrences) != 2 || occurrences[1].Weekday() != time.Wednesday {
		t.Fatalf("unexpected weekly occurrences: %v", occurrences)
	}
}

func TestAppointmentTypeAndBookingValidation(t *testing.T) {
	appointmentType := &AppointmentType{Name: "Consultation", Slug: "consultation", DurationMinutes: 30, MaxScheduleDays: 30, CompanyID: 1, StaffUserIDs: []int64{2}}
	if err := appointmentType.Validate(); err != nil {
		t.Fatalf("validate appointment type: %v", err)
	}
	booking := &AppointmentBooking{AppointmentTypeID: appointmentType.ID + 1, StaffID: 2, CustomerName: "Client", CustomerEmail: "client@example.com", StartTime: time.Now().UTC(), EndTime: time.Now().UTC().Add(30 * time.Minute)}
	if err := booking.Validate(); err != nil {
		t.Fatalf("validate booking: %v", err)
	}
}
