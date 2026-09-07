package hrusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// AttendanceUseCase implements hr.AttendanceUseCase.
type AttendanceUseCase struct {
	repo        hr.Repository
	companyRepo company.Repository
	logger      *slog.Logger
}

// NewAttendanceUseCase creates a new AttendanceUseCase.
func NewAttendanceUseCase(repo hr.Repository, companyRepo company.Repository, logger *slog.Logger) *AttendanceUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &AttendanceUseCase{
		repo:        repo,
		companyRepo: companyRepo,
		logger:      logger,
	}
}

func (u *AttendanceUseCase) CheckIn(ctx context.Context, employeeID int64, info hr.Attendance) (*hr.Attendance, error) {
	// 1. Validate employee exists
	emp, err := u.repo.GetEmployeeByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	// 2. Check for open attendance
	lastAtt, err := u.repo.GetLastAttendance(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if lastAtt != nil && lastAtt.CheckOut == nil {
		return nil, platformerrors.Conflict("employee already checked in", nil)
	}

	// 3. Prepare attendance
	att := &hr.Attendance{
		EmployeeID:  employeeID,
		CheckIn:     time.Now().UTC(),
		InLatitude:  info.InLatitude,
		InLongitude: info.InLongitude,
		InIPAddress: info.InIPAddress,
		InBrowser:   info.InBrowser,
		InMode:      info.InMode,
		CompanyID:   *emp.CompanyID,
	}
	if att.InMode == "" {
		att.InMode = hr.AttendanceModeManual
	}

	// 4. Save
	if err := u.repo.CreateAttendance(ctx, att); err != nil {
		return nil, err
	}

	u.logger.Info("employee checked in", "employee_id", employeeID, "attendance_id", att.ID)
	return att, nil
}

func (u *AttendanceUseCase) CheckOut(ctx context.Context, employeeID int64, info hr.Attendance) (*hr.Attendance, error) {
	// 1. Find last open attendance
	att, err := u.repo.GetLastAttendance(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if att == nil || att.CheckOut != nil {
		return nil, platformerrors.Conflict("employee is not checked in", nil)
	}

	// 2. Prepare check-out
	now := time.Now().UTC()
	att.CheckOut = &now
	att.OutLatitude = info.OutLatitude
	att.OutLongitude = info.OutLongitude
	att.OutIPAddress = info.OutIPAddress
	att.OutBrowser = info.OutBrowser
	att.OutMode = info.OutMode
	if att.OutMode == "" {
		att.OutMode = hr.AttendanceModeManual
	}

	// 3. Calculate worked hours
	att.WorkedHours = att.CalculateWorkedHours()

	// 4. Overtime calculation (Odoo 19.0 style)
	comp, err := u.companyRepo.GetByID(ctx, att.CompanyID)
	if err == nil {
		emp, _ := u.repo.GetEmployeeByID(ctx, employeeID)

		// Expected hours could come from a schedule/calendar (Phase 19 maybe)
		// For now, assume 8 hours if not defined
		att.ExpectedHours = 8.0

		overtime := att.WorkedHours - att.ExpectedHours
		if overtime > 0 {
			// Apply thresholds (Tolerance)
			threshold := comp.OvertimeCompanyThreshold
			if emp != nil && emp.OvertimeEmployeeThreshold > 0 {
				threshold = emp.OvertimeEmployeeThreshold
			}

			// threshold is in minutes
			if overtime*60 >= float64(threshold) {
				att.OvertimeHours = overtime
				att.OvertimeStatus = hr.OvertimeStatusToApprove

				// Create Overtime Line
				otLine := &hr.OvertimeLine{
					EmployeeID:   employeeID,
					AttendanceID: &att.ID,
					Date:         att.CheckIn,
					Duration:     overtime,
					Status:       hr.OvertimeStatusToApprove,
					TimeStart:    att.CheckIn.Add(time.Duration(att.ExpectedHours) * time.Hour),
					TimeStop:     *att.CheckOut,
					CompanyID:    att.CompanyID,
				}
				_ = u.repo.CreateOvertimeLine(ctx, otLine)
			}
		}
	}

	// 5. Update
	if err := u.repo.UpdateAttendance(ctx, att); err != nil {
		return nil, err
	}

	u.logger.Info("employee checked out", "employee_id", employeeID, "attendance_id", att.ID, "worked_hours", att.WorkedHours)
	return att, nil
}

func (u *AttendanceUseCase) GetKioskConfig(ctx context.Context, companyID int64) (map[string]any, error) {
	comp, err := u.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"kiosk_mode":  comp.AttendanceKioskMode,
		"kiosk_delay": comp.AttendanceKioskDelay,
	}, nil
}

func (u *AttendanceUseCase) ApproveOvertime(ctx context.Context, lineID int64, managerID int64) error {
	line, err := u.repo.GetOvertimeLineByID(ctx, lineID)
	if err != nil {
		return err
	}

	line.Status = hr.OvertimeStatusApproved
	line.ManualDuration = line.Duration // Use calculated as manual by default
	if err := u.repo.UpdateOvertimeLine(ctx, line); err != nil {
		return err
	}

	// Update associated attendance status
	if line.AttendanceID != nil {
		att, err := u.repo.GetAttendanceByID(ctx, *line.AttendanceID)
		if err == nil {
			att.OvertimeStatus = hr.OvertimeStatusApproved
			_ = u.repo.UpdateAttendance(ctx, att)
		}
	}

	return nil
}

func (u *AttendanceUseCase) RefuseOvertime(ctx context.Context, lineID int64, managerID int64) error {
	line, err := u.repo.GetOvertimeLineByID(ctx, lineID)
	if err != nil {
		return err
	}

	line.Status = hr.OvertimeStatusRefused
	if err := u.repo.UpdateOvertimeLine(ctx, line); err != nil {
		return err
	}

	if line.AttendanceID != nil {
		att, err := u.repo.GetAttendanceByID(ctx, *line.AttendanceID)
		if err == nil {
			att.OvertimeStatus = hr.OvertimeStatusRefused
			_ = u.repo.UpdateAttendance(ctx, att)
		}
	}

	return nil
}

func (u *AttendanceUseCase) GetAttendanceReport(ctx context.Context, employeeID int64, from, to time.Time) ([]hr.Attendance, error) {
	// Implement filtering logic
	// For simplicity, using ListAttendance with filters
	return nil, fmt.Errorf("not implemented")
}
