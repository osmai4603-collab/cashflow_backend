package calendarstorage

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/calendar"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (repo *PostgresRepo) CreateEvent(ctx context.Context, event *calendar.CalendarEvent) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO calendar_events (name, description, start_date, stop_date, duration, allday, location, video_url, privacy, show_as, user_id, res_model, res_id, recurrence_id, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id, created_at, updated_at`, event.Name, event.Description, event.Start, event.Stop, event.Duration, event.Allday, event.Location, event.VideoURL, event.Privacy, event.ShowAs, event.UserID, event.ResModel, event.ResID, event.RecurrenceID, event.CompanyID).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
}

func (repo *PostgresRepo) GetEvent(ctx context.Context, id int64) (*calendar.CalendarEvent, error) {
	event := &calendar.CalendarEvent{ID: id}
	err := repo.pool.QueryRow(ctx, `SELECT name, description, start_date, stop_date, duration, allday, location, video_url, privacy, show_as, user_id, res_model, res_id, recurrence_id, company_id, created_at, updated_at FROM calendar_events WHERE id=$1`, id).Scan(&event.Name, &event.Description, &event.Start, &event.Stop, &event.Duration, &event.Allday, &event.Location, &event.VideoURL, &event.Privacy, &event.ShowAs, &event.UserID, &event.ResModel, &event.ResID, &event.RecurrenceID, &event.CompanyID, &event.CreatedAt, &event.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound(fmt.Sprintf("calendar event with ID %d not found", id))
	}
	return event, err
}

func (repo *PostgresRepo) UpdateEvent(ctx context.Context, event *calendar.CalendarEvent) error {
	result, err := repo.pool.Exec(ctx, `UPDATE calendar_events SET name=$1, description=$2, start_date=$3, stop_date=$4, duration=$5, allday=$6, location=$7, video_url=$8, privacy=$9, show_as=$10, user_id=$11, res_model=$12, res_id=$13, recurrence_id=$14, updated_at=NOW() WHERE id=$15`, event.Name, event.Description, event.Start, event.Stop, event.Duration, event.Allday, event.Location, event.VideoURL, event.Privacy, event.ShowAs, event.UserID, event.ResModel, event.ResID, event.RecurrenceID, event.ID)
	if err == nil && result.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("calendar event with ID %d not found", event.ID))
	}
	return err
}

func (repo *PostgresRepo) DeleteEvent(ctx context.Context, id int64) error {
	result, err := repo.pool.Exec(ctx, `DELETE FROM calendar_events WHERE id=$1`, id)
	if err == nil && result.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("calendar event with ID %d not found", id))
	}
	return err
}

func (repo *PostgresRepo) ListEvents(ctx context.Context, filter calendar.EventFilter) ([]calendar.CalendarEvent, error) {
	query := `SELECT id, name, description, start_date, stop_date, duration, allday, location, video_url, privacy, show_as, user_id, res_model, res_id, recurrence_id, company_id, created_at, updated_at FROM calendar_events WHERE 1=1`
	args := make([]any, 0, 3)
	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		query += fmt.Sprintf(" AND user_id=$%d", len(args))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		query += fmt.Sprintf(" AND stop_date>$%d", len(args))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		query += fmt.Sprintf(" AND start_date<$%d", len(args))
	}
	query += " ORDER BY start_date"
	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]calendar.CalendarEvent, 0)
	for rows.Next() {
		var event calendar.CalendarEvent
		if err := rows.Scan(&event.ID, &event.Name, &event.Description, &event.Start, &event.Stop, &event.Duration, &event.Allday, &event.Location, &event.VideoURL, &event.Privacy, &event.ShowAs, &event.UserID, &event.ResModel, &event.ResID, &event.RecurrenceID, &event.CompanyID, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (repo *PostgresRepo) RespondAttendee(ctx context.Context, token string, status calendar.AttendeeStatus) (*calendar.Attendee, error) {
	attendee := &calendar.Attendee{Token: token, Status: status}
	err := repo.pool.QueryRow(ctx, `UPDATE calendar_attendees SET status=$1 WHERE token=$2 RETURNING id, event_id, partner_id, email, name, is_owner`, status, token).Scan(&attendee.ID, &attendee.EventID, &attendee.PartnerID, &attendee.Email, &attendee.Name, &attendee.IsOwner)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound("attendee token not found")
	}
	return attendee, err
}

func (repo *PostgresRepo) CreateAppointmentType(ctx context.Context, appointmentType *calendar.AppointmentType) error {
	if err := repo.pool.QueryRow(ctx, `INSERT INTO appointment_types (name, slug, duration_minutes, min_schedule_hours, max_schedule_days, assignation_method, location, active, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, appointmentType.Name, appointmentType.Slug, appointmentType.DurationMinutes, appointmentType.MinScheduleHours, appointmentType.MaxScheduleDays, appointmentType.AssignationMethod, appointmentType.Location, appointmentType.Active, appointmentType.CompanyID).Scan(&appointmentType.ID); err != nil {
		return err
	}
	for _, staffID := range appointmentType.StaffUserIDs {
		if _, err := repo.pool.Exec(ctx, `INSERT INTO appointment_type_users (appointment_type_id, user_id) VALUES ($1,$2)`, appointmentType.ID, staffID); err != nil {
			return err
		}
	}
	return nil
}

func (repo *PostgresRepo) scanAppointmentType(ctx context.Context, row pgx.Row) (*calendar.AppointmentType, error) {
	appointmentType := &calendar.AppointmentType{}
	if err := row.Scan(&appointmentType.ID, &appointmentType.Name, &appointmentType.Slug, &appointmentType.DurationMinutes, &appointmentType.MinScheduleHours, &appointmentType.MaxScheduleDays, &appointmentType.AssignationMethod, &appointmentType.Location, &appointmentType.Active, &appointmentType.CompanyID); err != nil {
		return nil, err
	}
	rows, err := repo.pool.Query(ctx, `SELECT user_id FROM appointment_type_users WHERE appointment_type_id=$1 ORDER BY user_id`, appointmentType.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var staffID int64
		if err := rows.Scan(&staffID); err != nil {
			return nil, err
		}
		appointmentType.StaffUserIDs = append(appointmentType.StaffUserIDs, staffID)
	}
	return appointmentType, rows.Err()
}

func (repo *PostgresRepo) GetAppointmentType(ctx context.Context, id int64) (*calendar.AppointmentType, error) {
	typeValue, err := repo.scanAppointmentType(ctx, repo.pool.QueryRow(ctx, `SELECT id, name, slug, duration_minutes, min_schedule_hours, max_schedule_days, assignation_method, location, active, company_id FROM appointment_types WHERE id=$1`, id))
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound("appointment type not found")
	}
	return typeValue, err
}

func (repo *PostgresRepo) GetAppointmentTypeBySlug(ctx context.Context, slug string) (*calendar.AppointmentType, error) {
	typeValue, err := repo.scanAppointmentType(ctx, repo.pool.QueryRow(ctx, `SELECT id, name, slug, duration_minutes, min_schedule_hours, max_schedule_days, assignation_method, location, active, company_id FROM appointment_types WHERE slug=$1`, slug))
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound("appointment type not found")
	}
	return typeValue, err
}

func (repo *PostgresRepo) ListAppointmentTypes(ctx context.Context, companyID int64) ([]calendar.AppointmentType, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id, name, slug, duration_minutes, min_schedule_hours, max_schedule_days, assignation_method, location, active, company_id FROM appointment_types WHERE company_id=$1 ORDER BY name`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]calendar.AppointmentType, 0)
	for rows.Next() {
		var appointmentType calendar.AppointmentType
		if err := rows.Scan(&appointmentType.ID, &appointmentType.Name, &appointmentType.Slug, &appointmentType.DurationMinutes, &appointmentType.MinScheduleHours, &appointmentType.MaxScheduleDays, &appointmentType.AssignationMethod, &appointmentType.Location, &appointmentType.Active, &appointmentType.CompanyID); err != nil {
			return nil, err
		}
		result = append(result, appointmentType)
	}
	return result, rows.Err()
}

func (repo *PostgresRepo) CreateBooking(ctx context.Context, booking *calendar.AppointmentBooking) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO appointment_bookings (appointment_type_id, event_id, staff_id, customer_name, customer_email, customer_phone, start_time, end_time, notes, status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at`, booking.AppointmentTypeID, booking.EventID, booking.StaffID, booking.CustomerName, booking.CustomerEmail, booking.CustomerPhone, booking.StartTime, booking.EndTime, booking.Notes, booking.Status).Scan(&booking.ID, &booking.CreatedAt)
}

func (repo *PostgresRepo) ListBookings(ctx context.Context, staffID int64, from, to time.Time) ([]calendar.AppointmentBooking, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id, appointment_type_id, event_id, staff_id, customer_name, customer_email, customer_phone, start_time, end_time, notes, status, created_at FROM appointment_bookings WHERE staff_id=$1 AND end_time>$2 AND start_time<$3 ORDER BY start_time`, staffID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]calendar.AppointmentBooking, 0)
	for rows.Next() {
		var booking calendar.AppointmentBooking
		if err := rows.Scan(&booking.ID, &booking.AppointmentTypeID, &booking.EventID, &booking.StaffID, &booking.CustomerName, &booking.CustomerEmail, &booking.CustomerPhone, &booking.StartTime, &booking.EndTime, &booking.Notes, &booking.Status, &booking.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, booking)
	}
	return result, rows.Err()
}

func (repo *PostgresRepo) CreateSlot(ctx context.Context, slot *calendar.AppointmentSlot) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO appointment_slots (appointment_type_id, day_of_week, hour_from, hour_to) VALUES ($1,$2,$3,$4) RETURNING id`, slot.AppointmentTypeID, slot.DayOfWeek, slot.HourFrom, slot.HourTo).Scan(&slot.ID)
}

func (repo *PostgresRepo) ListSlots(ctx context.Context, appointmentTypeID int64) ([]calendar.AppointmentSlot, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id, appointment_type_id, day_of_week, hour_from, hour_to FROM appointment_slots WHERE appointment_type_id=$1 ORDER BY day_of_week, hour_from`, appointmentTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]calendar.AppointmentSlot, 0)
	for rows.Next() {
		var slot calendar.AppointmentSlot
		if err := rows.Scan(&slot.ID, &slot.AppointmentTypeID, &slot.DayOfWeek, &slot.HourFrom, &slot.HourTo); err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	return result, rows.Err()
}
