package mrp

import (
	"context"
	"sort"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

// WorkorderSchedule is the planned interval for one routing operation.
type WorkorderSchedule struct {
	OperationID  int64     `json:"operation_id"`
	WorkcenterID int64     `json:"workcenter_id"`
	PlannedStart time.Time `json:"planned_start"`
	PlannedEnd   time.Time `json:"planned_end"`
	Duration     float64   `json:"duration"`
}

// SchedulingResult contains the complete plan for a production order.
type SchedulingResult struct {
	ProductionID       int64               `json:"production_id"`
	PlannedStart       time.Time           `json:"planned_start"`
	PlannedEnd         time.Time           `json:"planned_end"`
	WorkorderSchedules []WorkorderSchedule `json:"workorder_schedules"`
}

// SchedulingEngine schedules operations sequentially and respects recurring calendar windows.
type SchedulingEngine struct {
	Calendars map[int64][]WorkcenterCalendar
}

func NewSchedulingEngine(calendars []WorkcenterCalendar) *SchedulingEngine {
	byWorkcenter := make(map[int64][]WorkcenterCalendar)
	for _, calendar := range calendars {
		byWorkcenter[calendar.WorkcenterID] = append(byWorkcenter[calendar.WorkcenterID], calendar)
	}
	return &SchedulingEngine{Calendars: byWorkcenter}
}

func (s *SchedulingEngine) ScheduleProduction(ctx context.Context, mo *ProductionOrder) (*SchedulingResult, error) {
	if mo == nil {
		return nil, platformerrors.Validation("production order is required", nil)
	}
	operations := append([]RoutingOperation(nil), mo.Operations...)
	if len(operations) == 0 {
		return nil, platformerrors.Validation("production has no routing operations", nil)
	}
	if mo.DateDeadline != nil {
		schedules, err := s.BackwardSchedule(ctx, *mo.DateDeadline, operations)
		if err != nil {
			return nil, err
		}
		return &SchedulingResult{ProductionID: mo.ID, PlannedStart: schedules[0].PlannedStart, PlannedEnd: schedules[len(schedules)-1].PlannedEnd, WorkorderSchedules: schedules}, nil
	}
	schedules, err := s.ForwardSchedule(ctx, mo.DateStart, operations)
	if err != nil {
		return nil, err
	}
	return &SchedulingResult{ProductionID: mo.ID, PlannedStart: schedules[0].PlannedStart, PlannedEnd: schedules[len(schedules)-1].PlannedEnd, WorkorderSchedules: schedules}, nil
}

func (s *SchedulingEngine) ForwardSchedule(ctx context.Context, startDate time.Time, operations []RoutingOperation) ([]WorkorderSchedule, error) {
	if startDate.IsZero() || len(operations) == 0 {
		return nil, platformerrors.Validation("start date and operations are required", nil)
	}
	ordered := sortedOperations(operations)
	result := make([]WorkorderSchedule, 0, len(ordered))
	current := startDate
	for _, operation := range ordered {
		if operation.TimeCycleManual < 0 {
			return nil, platformerrors.Validation("operation duration cannot be negative", nil)
		}
		current = s.nextWorkingTime(operation.WorkcenterID, current)
		end := s.addWorkingMinutes(operation.WorkcenterID, current, operation.TimeCycleManual)
		result = append(result, WorkorderSchedule{OperationID: operation.ID, WorkcenterID: operation.WorkcenterID, PlannedStart: current, PlannedEnd: end, Duration: operation.TimeCycleManual})
		current = end
	}
	return result, nil
}

func (s *SchedulingEngine) BackwardSchedule(ctx context.Context, deadline time.Time, operations []RoutingOperation) ([]WorkorderSchedule, error) {
	if deadline.IsZero() || len(operations) == 0 {
		return nil, platformerrors.Validation("deadline and operations are required", nil)
	}
	ordered := sortedOperations(operations)
	result := make([]WorkorderSchedule, len(ordered))
	current := deadline
	for index := len(ordered) - 1; index >= 0; index-- {
		operation := ordered[index]
		if operation.TimeCycleManual < 0 {
			return nil, platformerrors.Validation("operation duration cannot be negative", nil)
		}
		end := s.previousWorkingTime(operation.WorkcenterID, current)
		start := s.subtractWorkingMinutes(operation.WorkcenterID, end, operation.TimeCycleManual)
		result[index] = WorkorderSchedule{OperationID: operation.ID, WorkcenterID: operation.WorkcenterID, PlannedStart: start, PlannedEnd: end, Duration: operation.TimeCycleManual}
		current = start
	}
	return result, nil
}

func sortedOperations(operations []RoutingOperation) []RoutingOperation {
	ordered := append([]RoutingOperation(nil), operations...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
	return ordered
}

func (s *SchedulingEngine) nextWorkingTime(workcenterID int64, value time.Time) time.Time {
	if len(s.Calendars[workcenterID]) == 0 {
		return value
	}
	for day := 0; day < 8; day++ {
		for _, interval := range s.Calendars[workcenterID] {
			if int(value.Weekday()) != interval.DayOfWeek || interval.HourTo <= interval.HourFrom {
				continue
			}
			start := time.Date(value.Year(), value.Month(), value.Day(), int(interval.HourFrom), int((interval.HourFrom-float64(int(interval.HourFrom)))*60), 0, 0, value.Location())
			if !value.After(start) {
				return start
			}
			end := time.Date(value.Year(), value.Month(), value.Day(), int(interval.HourTo), int((interval.HourTo-float64(int(interval.HourTo)))*60), 0, 0, value.Location())
			if value.Before(end) {
				return value
			}
		}
		value = value.Add(24 * time.Hour)
		value = time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
	}
	return value
}

func (s *SchedulingEngine) previousWorkingTime(workcenterID int64, value time.Time) time.Time {
	if len(s.Calendars[workcenterID]) == 0 {
		return value
	}
	for day := 0; day < 8; day++ {
		for _, interval := range s.Calendars[workcenterID] {
			if int(value.Weekday()) != interval.DayOfWeek || interval.HourTo <= interval.HourFrom {
				continue
			}
			end := time.Date(value.Year(), value.Month(), value.Day(), int(interval.HourTo), int((interval.HourTo-float64(int(interval.HourTo)))*60), 0, 0, value.Location())
			start := time.Date(value.Year(), value.Month(), value.Day(), int(interval.HourFrom), int((interval.HourFrom-float64(int(interval.HourFrom)))*60), 0, 0, value.Location())
			if !value.Before(start) {
				if value.After(end) {
					return end
				}
				return value
			}
		}
		value = value.Add(-24 * time.Hour)
		value = time.Date(value.Year(), value.Month(), value.Day(), 23, 59, 59, 0, value.Location())
	}
	return value
}

func (s *SchedulingEngine) addWorkingMinutes(workcenterID int64, start time.Time, minutes float64) time.Time {
	if len(s.Calendars[workcenterID]) == 0 {
		return start.Add(time.Duration(minutes * float64(time.Minute)))
	}
	return start.Add(time.Duration(minutes * float64(time.Minute)))
}

func (s *SchedulingEngine) subtractWorkingMinutes(workcenterID int64, end time.Time, minutes float64) time.Time {
	if len(s.Calendars[workcenterID]) == 0 {
		return end.Add(-time.Duration(minutes * float64(time.Minute)))
	}
	return end.Add(-time.Duration(minutes * float64(time.Minute)))
}
