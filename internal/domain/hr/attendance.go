package hr

import (
	"time"
)

// Attendance Status and Modes
const (
	OvertimeStatusToApprove = "to_approve"
	OvertimeStatusApproved  = "approved"
	OvertimeStatusRefused   = "refused"

	AttendanceModeKiosk     = "kiosk"
	AttendanceModeSystray   = "systray"
	AttendanceModeManual    = "manual"
	AttendanceModeTechnical = "technical"
	AttendanceModeAuto      = "auto_check_out"
)

// Attendance represents a single check-in/check-out record.
type Attendance struct {
	ID             int64      `json:"id"`
	EmployeeID     int64      `json:"employee_id"`
	CheckIn        time.Time  `json:"check_in"`
	CheckOut       *time.Time `json:"check_out"`
	WorkedHours    float64    `json:"worked_hours"`
	ExpectedHours  float64    `json:"expected_hours"`
	OvertimeHours  float64    `json:"overtime_hours"`
	OvertimeStatus string     `json:"overtime_status"`

	// In tracking
	InLatitude     float64 `json:"in_latitude"`
	InLongitude    float64 `json:"in_longitude"`
	InIPAddress    string  `json:"in_ip_address"`
	InBrowser      string  `json:"in_browser"`
	InMode         string  `json:"in_mode"`

	// Out tracking
	OutLatitude    float64 `json:"out_latitude"`
	OutLongitude   float64 `json:"out_longitude"`
	OutIPAddress   string  `json:"out_ip_address"`
	OutBrowser     string  `json:"out_browser"`
	OutMode        string  `json:"out_mode"`

	CompanyID      int64     `json:"company_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OvertimeLine represents an overtime entry.
type OvertimeLine struct {
	ID             int64     `json:"id"`
	EmployeeID     int64     `json:"employee_id"`
	AttendanceID   *int64    `json:"attendance_id,omitempty"`
	Date           time.Time `json:"date"`
	Duration       float64   `json:"duration"`
	ManualDuration float64   `json:"manual_duration"`
	Status         string    `json:"status"`
	TimeStart      time.Time `json:"time_start"`
	TimeStop       time.Time `json:"time_stop"`
	RuleIDs        []int64   `json:"rule_ids"`
	CompanyID      int64     `json:"company_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OvertimeRule defines how overtime is calculated.
type OvertimeRule struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	BaseOff     string  `json:"base_off"`    // quantity, timing
	TimingType  string  `json:"timing_type"` // work_days, non_work_days, leave, schedule
	TimingStart float64 `json:"timing_start"`
	Multiplier  float64 `json:"multiplier"`
	Active      bool    `json:"active"`
	CompanyID   int64   `json:"company_id"`
}

// WorkedHours calculates the duration between check-in and check-out in hours.
func (a *Attendance) CalculateWorkedHours() float64 {
	if a.CheckOut == nil {
		return 0
	}
	duration := a.CheckOut.Sub(a.CheckIn)
	return duration.Hours()
}
