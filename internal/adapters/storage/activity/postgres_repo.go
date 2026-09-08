package activitystorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/activity"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// --- ActivityTypeRepository ---

func (r *PostgresRepo) CreateType(ctx context.Context, t *activity.ActivityType) error {
	query := `
		INSERT INTO mail_activity_types (
			name, summary, res_model, category, delay_count, delay_unit,
			icon, sequence, default_note, active, system_type, company_id,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		t.Name, t.Summary, t.ResModel, t.Category, t.DelayCount, t.DelayUnit,
		t.Icon, t.Sequence, t.DefaultNote, t.Active, t.SystemType, t.CompanyID,
		t.CreatedBy, t.UpdatedBy,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create activity type", err)
	}
	return nil
}

func (r *PostgresRepo) GetTypeByID(ctx context.Context, id int64) (*activity.ActivityType, error) {
	query := `
		SELECT id, name, summary, res_model, category, delay_count, delay_unit,
		       icon, sequence, default_note, active, system_type, company_id
		FROM mail_activity_types WHERE id = $1
	`
	var t activity.ActivityType
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Summary, &t.ResModel, &t.Category, &t.DelayCount, &t.DelayUnit,
		&t.Icon, &t.Sequence, &t.DefaultNote, &t.Active, &t.SystemType, &t.CompanyID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("activity type %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get activity type", err)
	}
	return &t, nil
}

func (r *PostgresRepo) UpdateType(ctx context.Context, t *activity.ActivityType) error {
	query := `
		UPDATE mail_activity_types SET
			name = $1, summary = $2, res_model = $3, category = $4,
			delay_count = $5, delay_unit = $6, icon = $7, sequence = $8,
			default_note = $9, active = $10, updated_by = $11, updated_at = NOW()
		WHERE id = $12
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		t.Name, t.Summary, t.ResModel, t.Category, t.DelayCount, t.DelayUnit,
		t.Icon, t.Sequence, t.DefaultNote, t.Active, t.UpdatedBy, t.ID,
	).Scan(&t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("activity type %d not found", t.ID))
		}
		return platformerrors.Internal("failed to update activity type", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteType(ctx context.Context, id int64) error {
	query := `UPDATE mail_activity_types SET active = false, updated_at = NOW() WHERE id = $1 AND system_type = false`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete activity type", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("activity type not found or is a system type")
	}
	return nil
}

func (r *PostgresRepo) ListTypes(ctx context.Context, companyID *int64) ([]activity.ActivityType, error) {
	query := `
		SELECT id, name, summary, res_model, category, delay_count, delay_unit,
		       icon, sequence, default_note, active, system_type, company_id
		FROM mail_activity_types
		WHERE active = true AND (company_id IS NULL OR company_id = $1)
		ORDER BY sequence, name
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list activity types", err)
	}
	defer rows.Close()

	var items []activity.ActivityType
	for rows.Next() {
		var t activity.ActivityType
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Summary, &t.ResModel, &t.Category, &t.DelayCount, &t.DelayUnit,
			&t.Icon, &t.Sequence, &t.DefaultNote, &t.Active, &t.SystemType, &t.CompanyID,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan activity type", err)
		}
		items = append(items, t)
	}
	return items, nil
}

// --- ActivityRepository ---

const selectActivityFields = `
	id, activity_type_id, summary, note, date_deadline, assigned_user_id,
	res_model, res_id, active, date_done, feedback, company_id,
	created_at, updated_at, created_by, updated_by
`

func (r *PostgresRepo) CreateActivity(ctx context.Context, a *activity.Activity) error {
	query := `
		INSERT INTO mail_activities (
			activity_type_id, summary, note, date_deadline, assigned_user_id,
			res_model, res_id, active, company_id, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		a.ActivityTypeID, a.Summary, a.Note, a.DateDeadline, a.AssignedUserID,
		a.ResModel, a.ResID, a.Active, a.CompanyID, a.CreatedBy, a.UpdatedBy,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create activity", err)
	}
	return nil
}

func (r *PostgresRepo) GetActivityByID(ctx context.Context, id int64) (*activity.Activity, error) {
	query := fmt.Sprintf("SELECT %s FROM mail_activities WHERE id = $1", selectActivityFields)
	var a activity.Activity
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.ActivityTypeID, &a.Summary, &a.Note, &a.DateDeadline, &a.AssignedUserID,
		&a.ResModel, &a.ResID, &a.Active, &a.DateDone, &a.Feedback, &a.CompanyID,
		&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("activity %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get activity", err)
	}
	return &a, nil
}

func (r *PostgresRepo) UpdateActivity(ctx context.Context, a *activity.Activity) error {
	query := `
		UPDATE mail_activities SET
			activity_type_id = $1, summary = $2, note = $3, date_deadline = $4,
			assigned_user_id = $5, res_model = $6, res_id = $7, active = $8,
			updated_by = $9, updated_at = NOW()
		WHERE id = $10
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		a.ActivityTypeID, a.Summary, a.Note, a.DateDeadline, a.AssignedUserID,
		a.ResModel, a.ResID, a.Active, a.UpdatedBy, a.ID,
	).Scan(&a.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("activity %d not found", a.ID))
		}
		return platformerrors.Internal("failed to update activity", err)
	}
	return nil
}

func (r *PostgresRepo) CompleteActivity(ctx context.Context, id int64, now time.Time, feedback string) (*activity.Activity, error) {
	query := `
		UPDATE mail_activities SET
			active = false, date_done = $1, feedback = $2, updated_at = NOW()
		WHERE id = $3 AND active = true
		RETURNING ` + selectActivityFields

	var a activity.Activity
	err := r.pool.QueryRow(ctx, query, now, feedback, id).Scan(
		&a.ID, &a.ActivityTypeID, &a.Summary, &a.Note, &a.DateDeadline, &a.AssignedUserID,
		&a.ResModel, &a.ResID, &a.Active, &a.DateDone, &a.Feedback, &a.CompanyID,
		&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.Conflict("activity already completed or not found")
		}
		return nil, platformerrors.Internal("failed to complete activity", err)
	}
	return &a, nil
}

func (r *PostgresRepo) ListActivities(ctx context.Context, f activity.ActivityFilter) (pagination.PageResult[activity.Activity], error) {
	query := "SELECT " + selectActivityFields + " FROM mail_activities WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM mail_activities WHERE 1=1"
	args := []any{}
	idx := 1

	if f.AssignedUserID != nil {
		query += fmt.Sprintf(" AND assigned_user_id = $%d", idx)
		countQuery += fmt.Sprintf(" AND assigned_user_id = $%d", idx)
		args = append(args, *f.AssignedUserID)
		idx++
	}
	if f.ResModel != "" {
		query += fmt.Sprintf(" AND res_model = $%d", idx)
		countQuery += fmt.Sprintf(" AND res_model = $%d", idx)
		args = append(args, f.ResModel)
		idx++
	}
	if f.ResID != nil {
		query += fmt.Sprintf(" AND res_id = $%d", idx)
		countQuery += fmt.Sprintf(" AND res_id = $%d", idx)
		args = append(args, *f.ResID)
		idx++
	}
	if f.Active != nil {
		query += fmt.Sprintf(" AND active = $%d", idx)
		countQuery += fmt.Sprintf(" AND active = $%d", idx)
		args = append(args, *f.Active)
		idx++
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[activity.Activity]{}, platformerrors.Internal("failed to count activities", err)
	}

	query += fmt.Sprintf(" ORDER BY date_deadline ASC, id ASC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, f.Page.LimitClamped(), f.Page.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[activity.Activity]{}, platformerrors.Internal("failed to list activities", err)
	}
	defer rows.Close()

	items := []activity.Activity{}
	for rows.Next() {
		var a activity.Activity
		if err := rows.Scan(
			&a.ID, &a.ActivityTypeID, &a.Summary, &a.Note, &a.DateDeadline, &a.AssignedUserID,
			&a.ResModel, &a.ResID, &a.Active, &a.DateDone, &a.Feedback, &a.CompanyID,
			&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy,
		); err != nil {
			return pagination.PageResult[activity.Activity]{}, platformerrors.Internal("failed to scan activity", err)
		}
		items = append(items, a)
	}

	return pagination.NewPageResult(items, total, f.Page), nil
}

// --- MessageRepository ---

func (r *PostgresRepo) CreateMessage(ctx context.Context, m *activity.Message) error {
	query := `
		INSERT INTO mail_messages (
			subject, body, message_type, res_model, res_id, author_id,
			activity_id, subtype_id, parent_id, company_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		m.Subject, m.Body, m.MessageType, m.ResModel, m.ResID, m.AuthorID,
		m.ActivityID, m.SubtypeID, m.ParentID, m.CompanyID,
	).Scan(&m.ID, &m.CreatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create message", err)
	}
	return nil
}

func (r *PostgresRepo) GetMessageByID(ctx context.Context, id int64) (*activity.Message, error) {
	query := `
		SELECT id, subject, body, message_type, res_model, res_id, author_id,
		       activity_id, subtype_id, parent_id, company_id, created_at
		FROM mail_messages WHERE id = $1
	`
	var m activity.Message
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Subject, &m.Body, &m.MessageType, &m.ResModel, &m.ResID, &m.AuthorID,
		&m.ActivityID, &m.SubtypeID, &m.ParentID, &m.CompanyID, &m.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("message %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get message", err)
	}
	return &m, nil
}

func (r *PostgresRepo) ListMessagesByResource(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[activity.Message], error) {
	countQuery := `SELECT COUNT(*) FROM mail_messages WHERE res_model = $1 AND res_id = $2`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, resModel, resID).Scan(&total); err != nil {
		return pagination.PageResult[activity.Message]{}, platformerrors.Internal("failed to count messages", err)
	}

	query := `
		SELECT id, subject, body, message_type, res_model, res_id, author_id,
		       activity_id, subtype_id, parent_id, company_id, created_at
		FROM mail_messages
		WHERE res_model = $1 AND res_id = $2
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, resModel, resID, page.LimitClamped(), page.Offset())
	if err != nil {
		return pagination.PageResult[activity.Message]{}, platformerrors.Internal("failed to list messages", err)
	}
	defer rows.Close()

	items := []activity.Message{}
	for rows.Next() {
		var m activity.Message
		if err := rows.Scan(
			&m.ID, &m.Subject, &m.Body, &m.MessageType, &m.ResModel, &m.ResID, &m.AuthorID,
			&m.ActivityID, &m.SubtypeID, &m.ParentID, &m.CompanyID, &m.CreatedAt,
		); err != nil {
			return pagination.PageResult[activity.Message]{}, platformerrors.Internal("failed to scan message", err)
		}
		items = append(items, m)
	}
	return pagination.NewPageResult(items, total, page), nil
}

// --- NotificationRepository ---

func (r *PostgresRepo) CreateNotification(ctx context.Context, n *activity.Notification) error {
	query := `
		INSERT INTO mail_notifications (
			message_id, activity_id, recipient_user_id, notification_type,
			status, read_at, email, company_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		n.MessageID, n.ActivityID, n.RecipientUserID, n.NotificationType,
		n.Status, n.ReadAt, n.Email, n.CompanyID,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create notification", err)
	}
	return nil
}

func (r *PostgresRepo) GetNotificationByID(ctx context.Context, id int64) (*activity.Notification, error) {
	query := `
		SELECT id, message_id, activity_id, recipient_user_id, notification_type,
		       status, read_at, email, created_at, updated_at, company_id
		FROM mail_notifications WHERE id = $1
	`
	var n activity.Notification
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.MessageID, &n.ActivityID, &n.RecipientUserID, &n.NotificationType,
		&n.Status, &n.ReadAt, &n.Email, &n.CreatedAt, &n.UpdatedAt, &n.CompanyID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("notification %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get notification", err)
	}
	return &n, nil
}

func (r *PostgresRepo) UpdateNotification(ctx context.Context, n *activity.Notification) error {
	query := `
		UPDATE mail_notifications SET
			status = $1, read_at = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, n.Status, n.ReadAt, n.ID).Scan(&n.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("notification %d not found", n.ID))
		}
		return platformerrors.Internal("failed to update notification", err)
	}
	return nil
}

func (r *PostgresRepo) ListNotificationsForUser(ctx context.Context, userID int64, status *activity.NotificationStatus, page pagination.PageRequest) (pagination.PageResult[activity.Notification], error) {
	query := `SELECT COUNT(*) FROM mail_notifications WHERE recipient_user_id = $1`
	args := []any{userID}
	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return pagination.PageResult[activity.Notification]{}, platformerrors.Internal("failed to count notifications", err)
	}

	query = `
		SELECT id, message_id, activity_id, recipient_user_id, notification_type,
		       status, read_at, email, created_at, updated_at, company_id
		FROM mail_notifications
		WHERE recipient_user_id = $1
	`
	idx := 2
	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", idx)
		idx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, page.LimitClamped(), page.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[activity.Notification]{}, platformerrors.Internal("failed to list notifications", err)
	}
	defer rows.Close()

	items := []activity.Notification{}
	for rows.Next() {
		var n activity.Notification
		if err := rows.Scan(
			&n.ID, &n.MessageID, &n.ActivityID, &n.RecipientUserID, &n.NotificationType,
			&n.Status, &n.ReadAt, &n.Email, &n.CreatedAt, &n.UpdatedAt, &n.CompanyID,
		); err != nil {
			return pagination.PageResult[activity.Notification]{}, platformerrors.Internal("failed to scan notification", err)
		}
		items = append(items, n)
	}
	return pagination.NewPageResult(items, total, page), nil
}

func (r *PostgresRepo) MarkAllRead(ctx context.Context, userID int64, companyID int64) error {
	query := `
		UPDATE mail_notifications SET status = 'read', read_at = NOW(), updated_at = NOW()
		WHERE recipient_user_id = $1 AND company_id = $2 AND status = 'unread'
	`
	_, err := r.pool.Exec(ctx, query, userID, companyID)
	if err != nil {
		return platformerrors.Internal("failed to mark all notifications as read", err)
	}
	return nil
}

// --- EmailQueueRepository ---

func (r *PostgresRepo) PushEmail(ctx context.Context, e *activity.EmailQueueItem) error {
	query := `
		INSERT INTO mail_email_queue (
			notification_id, recipient_email, subject, body, status,
			attempts, next_attempt_at, last_error, company_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		e.NotificationID, e.RecipientEmail, e.Subject, e.Body, e.Status,
		e.Attempts, e.NextAttemptAt, e.LastError, e.CompanyID,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to push email to queue", err)
	}
	return nil
}

func (r *PostgresRepo) PopEmails(ctx context.Context, limit int) ([]activity.EmailQueueItem, error) {
	// Simple pop: select items that are queued or failed but ready for retry
	query := `
		UPDATE mail_email_queue SET status = 'processing', updated_at = NOW()
		WHERE id IN (
			SELECT id FROM mail_email_queue
			WHERE status IN ('queued', 'failed') AND next_attempt_at <= NOW()
			ORDER BY next_attempt_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, notification_id, recipient_email, subject, body, status,
		          attempts, next_attempt_at, last_error, sent_at, created_at, updated_at, company_id
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, platformerrors.Internal("failed to pop emails from queue", err)
	}
	defer rows.Close()

	items := []activity.EmailQueueItem{}
	for rows.Next() {
		var e activity.EmailQueueItem
		if err := rows.Scan(
			&e.ID, &e.NotificationID, &e.RecipientEmail, &e.Subject, &e.Body, &e.Status,
			&e.Attempts, &e.NextAttemptAt, &e.LastError, &e.SentAt, &e.CreatedAt, &e.UpdatedAt, &e.CompanyID,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan email queue item", err)
		}
		items = append(items, e)
	}
	return items, nil
}

func (r *PostgresRepo) UpdateEmail(ctx context.Context, e *activity.EmailQueueItem) error {
	_, err := r.pool.Exec(ctx, `UPDATE mail_email_queue SET status = $1, last_error = $2, updated_at = NOW() WHERE id = $3`, e.Status, e.LastError, e.ID)
	if err != nil {
		return platformerrors.Internal("failed to update email queue item", err)
	}
	return nil
}

func (r *PostgresRepo) CreateTrackingValues(ctx context.Context, values []activity.TrackingValue) error {
	if len(values) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	query := `
		INSERT INTO mail_tracking_values (
			message_id, field_name, field_desc, old_value_text, new_value_text
		) VALUES ($1, $2, $3, $4, $5)
	`
	for _, v := range values {
		batch.Queue(query, v.MessageID, v.Field, v.FieldName, v.OldValueText, v.NewValueText)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range values {
		if _, err := br.Exec(); err != nil {
			return platformerrors.Internal("failed to create tracking value", err)
		}
	}
	return nil
}

func (r *PostgresRepo) ListTrackingValues(ctx context.Context, messageID int64) ([]activity.TrackingValue, error) {
	query := `
		SELECT id, message_id, field_name, field_desc, old_value_text, new_value_text
		FROM mail_tracking_values WHERE message_id = $1
	`
	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list tracking values", err)
	}
	defer rows.Close()

	var items []activity.TrackingValue
	for rows.Next() {
		var v activity.TrackingValue
		if err := rows.Scan(&v.ID, &v.MessageID, &v.Field, &v.FieldName, &v.OldValueText, &v.NewValueText); err != nil {
			return nil, platformerrors.Internal("failed to scan tracking value", err)
		}
		items = append(items, v)
	}
	return items, nil
}

// --- FollowerRepository ---

func (r *PostgresRepo) CreateFollower(ctx context.Context, f *activity.Follower) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return platformerrors.Internal("failed to start transaction", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO mail_followers (res_model, res_id, partner_id, user_id, company_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (res_model, res_id, partner_id) WHERE partner_id IS NOT NULL DO UPDATE SET company_id = EXCLUDED.company_id
		ON CONFLICT (res_model, res_id, user_id) WHERE user_id IS NOT NULL DO UPDATE SET company_id = EXCLUDED.company_id
		RETURNING id
	`
	// Note: PostgreSQL 9.5+ ON CONFLICT doesn't support multiple constraints easily.
	// Simplified version:
	if f.UserID != nil {
		query = `
			INSERT INTO mail_followers (res_model, res_id, user_id, company_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (res_model, res_id, user_id) DO UPDATE SET company_id = EXCLUDED.company_id
			RETURNING id
		`
		err = tx.QueryRow(ctx, query, f.ResModel, f.ResID, f.UserID, f.CompanyID).Scan(&f.ID)
	} else {
		query = `
			INSERT INTO mail_followers (res_model, res_id, partner_id, company_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (res_model, res_id, partner_id) DO UPDATE SET company_id = EXCLUDED.company_id
			RETURNING id
		`
		err = tx.QueryRow(ctx, query, f.ResModel, f.ResID, f.PartnerID, f.CompanyID).Scan(&f.ID)
	}

	if err != nil {
		return platformerrors.Internal("failed to create follower", err)
	}

	// Subtypes relation
	if len(f.SubtypeIDs) > 0 {
		_, err = tx.Exec(ctx, "DELETE FROM mail_followers_subtypes_rel WHERE follower_id = $1", f.ID)
		if err != nil {
			return platformerrors.Internal("failed to clear follower subtypes", err)
		}
		for _, sid := range f.SubtypeIDs {
			_, err = tx.Exec(ctx, "INSERT INTO mail_followers_subtypes_rel (follower_id, subtype_id) VALUES ($1, $2)", f.ID, sid)
			if err != nil {
				return platformerrors.Internal("failed to link follower subtype", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) DeleteFollower(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM mail_followers WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete follower", err)
	}
	return nil
}

func (r *PostgresRepo) ListFollowers(ctx context.Context, resModel string, resID int64) ([]activity.Follower, error) {
	query := `
		SELECT f.id, f.res_model, f.res_id, f.partner_id, f.user_id, f.company_id,
		       ARRAY_AGG(rel.subtype_id) FILTER (WHERE rel.subtype_id IS NOT NULL)
		FROM mail_followers f
		LEFT JOIN mail_followers_subtypes_rel rel ON f.id = rel.follower_id
		WHERE f.res_model = $1 AND f.res_id = $2
		GROUP BY f.id
	`
	rows, err := r.pool.Query(ctx, query, resModel, resID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list followers", err)
	}
	defer rows.Close()

	var items []activity.Follower
	for rows.Next() {
		var f activity.Follower
		var subtypes []int64
		if err := rows.Scan(&f.ID, &f.ResModel, &f.ResID, &f.PartnerID, &f.UserID, &f.CompanyID, &subtypes); err != nil {
			return nil, platformerrors.Internal("failed to scan follower", err)
		}
		f.SubtypeIDs = subtypes
		items = append(items, f)
	}
	return items, nil
}

func (r *PostgresRepo) GetFollowersForNotification(ctx context.Context, resModel string, resID int64, subtypeID int64) ([]activity.Follower, error) {
	query := `
		SELECT f.id, f.res_model, f.res_id, f.partner_id, f.user_id, f.company_id
		FROM mail_followers f
		JOIN mail_followers_subtypes_rel rel ON f.id = rel.follower_id
		WHERE f.res_model = $1 AND f.res_id = $2 AND rel.subtype_id = $3
	`
	if subtypeID == 0 {
		// If no subtype provided, maybe notify all? In Odoo it's specific.
		// For now, let's assume if subtype is 0, we take all who have at least one subtype (or specifically default ones)
		query = `
			SELECT DISTINCT f.id, f.res_model, f.res_id, f.partner_id, f.user_id, f.company_id
			FROM mail_followers f
			WHERE f.res_model = $1 AND f.res_id = $2
		`
	}
	rows, err := r.pool.Query(ctx, query, resModel, resID, subtypeID)
	if err != nil {
		if subtypeID == 0 {
			rows, err = r.pool.Query(ctx, query, resModel, resID)
		}
		if err != nil {
			return nil, platformerrors.Internal("failed to get followers for notification", err)
		}
	}
	defer rows.Close()

	var items []activity.Follower
	for rows.Next() {
		var f activity.Follower
		if err := rows.Scan(&f.ID, &f.ResModel, &f.ResID, &f.PartnerID, &f.UserID, &f.CompanyID); err != nil {
			return nil, platformerrors.Internal("failed to scan follower", err)
		}
		items = append(items, f)
	}
	return items, nil
}

// --- SubtypeRepository ---

func (r *PostgresRepo) GetSubtypeByID(ctx context.Context, id int64) (*activity.MessageSubtype, error) {
	query := `SELECT id, name, res_model, description, internal, default_subtype, sequence FROM mail_message_subtypes WHERE id = $1`
	var s activity.MessageSubtype
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.Name, &s.ResModel, &s.Description, &s.Internal, &s.Default, &s.Sequence)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("subtype not found")
		}
		return nil, platformerrors.Internal("failed to get subtype", err)
	}
	return &s, nil
}

func (r *PostgresRepo) GetSubtypeByName(ctx context.Context, resModel string, name string) (*activity.MessageSubtype, error) {
	query := `
		SELECT id, name, res_model, description, internal, default_subtype, sequence
		FROM mail_message_subtypes
		WHERE name = $1 AND (res_model IS NULL OR res_model = $2)
		ORDER BY res_model DESC LIMIT 1
	`
	var s activity.MessageSubtype
	err := r.pool.QueryRow(ctx, query, name, resModel).Scan(&s.ID, &s.Name, &s.ResModel, &s.Description, &s.Internal, &s.Default, &s.Sequence)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("subtype %s for %s not found", name, resModel))
		}
		return nil, platformerrors.Internal("failed to get subtype by name", err)
	}
	return &s, nil
}

func (r *PostgresRepo) ListSubtypes(ctx context.Context, resModel string) ([]activity.MessageSubtype, error) {
	query := `
		SELECT id, name, res_model, description, internal, default_subtype, sequence
		FROM mail_message_subtypes
		WHERE res_model IS NULL OR res_model = $1
		ORDER BY sequence, name
	`
	rows, err := r.pool.Query(ctx, query, resModel)
	if err != nil {
		return nil, platformerrors.Internal("failed to list subtypes", err)
	}
	defer rows.Close()

	var items []activity.MessageSubtype
	for rows.Next() {
		var s activity.MessageSubtype
		if err := rows.Scan(&s.ID, &s.Name, &s.ResModel, &s.Description, &s.Internal, &s.Default, &s.Sequence); err != nil {
			return nil, platformerrors.Internal("failed to scan subtype", err)
		}
		items = append(items, s)
	}
	return items, nil
}
