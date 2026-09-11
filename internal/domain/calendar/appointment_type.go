package calendar

import (
	"regexp"
	"strings"
)

type AssignMethod string

const (
	AssignMethodRandom     AssignMethod = "random"
	AssignMethodRoundRobin AssignMethod = "round_robin"
	AssignMethodChosen     AssignMethod = "chosen"
)

type AppointmentType struct {
	ID                int64        `json:"id"`
	Name              string       `json:"name"`
	Slug              string       `json:"slug"`
	DurationMinutes   int          `json:"duration_minutes"`
	MinScheduleHours  int          `json:"min_schedule_hours"`
	MaxScheduleDays   int          `json:"max_schedule_days"`
	AssignationMethod AssignMethod `json:"assignation_method"`
	StaffUserIDs      []int64      `json:"staff_user_ids"`
	Location          string       `json:"location,omitempty"`
	ReminderMinutes   []int        `json:"reminder_minutes"`
	Active            bool         `json:"active"`
	CompanyID         int64        `json:"company_id"`
}

func (appointmentType *AppointmentType) Validate() error {
	appointmentType.Name = strings.TrimSpace(appointmentType.Name)
	appointmentType.Slug = strings.TrimSpace(strings.ToLower(appointmentType.Slug))
	if appointmentType.Name == "" || !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(appointmentType.Slug) {
		return invalidCalendar("appointment type requires a name and URL-safe slug")
	}
	if appointmentType.DurationMinutes <= 0 || appointmentType.MaxScheduleDays <= 0 {
		return invalidCalendar("appointment duration and max schedule days must be positive")
	}
	if appointmentType.MinScheduleHours < 0 || appointmentType.CompanyID <= 0 || len(appointmentType.StaffUserIDs) == 0 {
		return invalidCalendar("appointment type requires staff, company, and non-negative lead time")
	}
	if appointmentType.AssignationMethod == "" {
		appointmentType.AssignationMethod = AssignMethodRoundRobin
	}
	return nil
}
