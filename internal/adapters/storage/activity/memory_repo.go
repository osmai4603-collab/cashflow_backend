package activitystorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cashflow_backend/internal/domain/activity"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/pagination"
)

type MemoryRepo struct {
	mu             sync.RWMutex
	types          map[int64]activity.ActivityType
	activities     map[int64]activity.Activity
	messages       map[int64]activity.Message
	notifs         map[int64]activity.Notification
	emails         map[int64]activity.EmailQueueItem
	followers      map[int64]activity.Follower
	subtypes       map[int64]activity.MessageSubtype
	trackingValues map[int64][]activity.TrackingValue
	nextTypeID     int64
	nextActID      int64
	nextMsgID      int64
	nextNotifID    int64
	nextEmailID    int64
	nextFollowerID int64
	nextSubtypeID  int64
}

func NewMemoryRepo() *MemoryRepo {
	r := &MemoryRepo{
		types:          make(map[int64]activity.ActivityType),
		activities:     make(map[int64]activity.Activity),
		messages:       make(map[int64]activity.Message),
		notifs:         make(map[int64]activity.Notification),
		emails:         make(map[int64]activity.EmailQueueItem),
		followers:      make(map[int64]activity.Follower),
		subtypes:       make(map[int64]activity.MessageSubtype),
		trackingValues: make(map[int64][]activity.TrackingValue),
		nextTypeID:     1,
		nextActID:      1,
		nextMsgID:      1,
		nextNotifID:    1,
		nextEmailID:    1,
		nextFollowerID: 1,
		nextSubtypeID:  1,
	}
	r.seedSubtypes()
	return r
}

func (r *MemoryRepo) seedSubtypes() {
	r.subtypes[1] = activity.MessageSubtype{ID: 1, Name: "discussions", Default: true, Sequence: 1}
	r.subtypes[2] = activity.MessageSubtype{ID: 2, Name: "note", Internal: true, Sequence: 10}
	r.subtypes[3] = activity.MessageSubtype{ID: 3, Name: "activities", Internal: true, Sequence: 20}
	r.subtypes[4] = activity.MessageSubtype{ID: 4, Name: "mt_comment", Description: "Comment", Default: true, Sequence: 1}
	r.subtypes[5] = activity.MessageSubtype{ID: 5, Name: "mt_note", Description: "Note", Internal: true, Sequence: 10}
	r.nextSubtypeID = 6
}

// --- ActivityTypeRepository ---

func (r *MemoryRepo) CreateType(ctx context.Context, t *activity.ActivityType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.ID = r.nextTypeID
	r.nextTypeID++
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	r.types[t.ID] = *t
	return nil
}

func (r *MemoryRepo) GetTypeByID(ctx context.Context, id int64) (*activity.ActivityType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.types[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("activity type %d not found", id))
	}
	return &t, nil
}

func (r *MemoryRepo) UpdateType(ctx context.Context, t *activity.ActivityType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.types[t.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("activity type %d not found", t.ID))
	}
	t.UpdatedAt = time.Now().UTC()
	r.types[t.ID] = *t
	return nil
}

func (r *MemoryRepo) DeleteType(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.types[id]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("activity type %d not found", id))
	}
	if t.SystemType {
		return platformerrors.Forbidden("cannot delete system activity type")
	}
	t.Active = false
	t.UpdatedAt = time.Now().UTC()
	r.types[id] = t
	return nil
}

func (r *MemoryRepo) ListTypes(ctx context.Context, companyID *int64) ([]activity.ActivityType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []activity.ActivityType
	for _, t := range r.types {
		if t.Active && (companyID == nil || t.CompanyID == nil || *t.CompanyID == *companyID) {
			res = append(res, t)
		}
	}
	return res, nil
}

// --- ActivityRepository ---

func (r *MemoryRepo) CreateActivity(ctx context.Context, a *activity.Activity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a.ID = r.nextActID
	r.nextActID++
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	r.activities[a.ID] = *a
	return nil
}

func (r *MemoryRepo) GetActivityByID(ctx context.Context, id int64) (*activity.Activity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.activities[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("activity %d not found", id))
	}
	return &a, nil
}

func (r *MemoryRepo) UpdateActivity(ctx context.Context, a *activity.Activity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.activities[a.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("activity %d not found", a.ID))
	}
	a.UpdatedAt = time.Now().UTC()
	r.activities[a.ID] = *a
	return nil
}

func (r *MemoryRepo) CompleteActivity(ctx context.Context, id int64, now time.Time, feedback string) (*activity.Activity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.activities[id]
	if !ok || !a.Active {
		return nil, platformerrors.Conflict("activity already completed or not found")
	}
	a.Active = false
	a.DateDone = &now
	a.Feedback = feedback
	a.UpdatedAt = time.Now().UTC()
	r.activities[id] = a
	return &a, nil
}

func (r *MemoryRepo) ListActivities(ctx context.Context, f activity.ActivityFilter) (pagination.PageResult[activity.Activity], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []activity.Activity
	for _, a := range r.activities {
		match := true
		if f.AssignedUserID != nil && a.AssignedUserID != *f.AssignedUserID {
			match = false
		}
		if f.ResModel != "" && a.ResModel != f.ResModel {
			match = false
		}
		if f.ResID != nil && (a.ResID == nil || *a.ResID != *f.ResID) {
			match = false
		}
		if f.Active != nil && a.Active != *f.Active {
			match = false
		}
		if match {
			filtered = append(filtered, a)
		}
	}

	// Simple pagination
	total := int64(len(filtered))
	start := int64(f.Page.Offset())
	if start > total {
		start = total
	}
	end := start + int64(f.Page.LimitClamped())
	if end > total {
		end = total
	}

	return pagination.NewPageResult(filtered[start:end], total, f.Page), nil
}

// --- MessageRepository ---

func (r *MemoryRepo) CreateMessage(ctx context.Context, m *activity.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m.ID = r.nextMsgID
	r.nextMsgID++
	m.CreatedAt = time.Now().UTC()
	r.messages[m.ID] = *m
	return nil
}

func (r *MemoryRepo) GetMessageByID(ctx context.Context, id int64) (*activity.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.messages[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("message %d not found", id))
	}
	return &m, nil
}

func (r *MemoryRepo) ListMessagesByResource(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[activity.Message], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []activity.Message
	for _, m := range r.messages {
		if m.ResModel == resModel && m.ResID != nil && *m.ResID == resID {
			// Populate tracking values if requested (simplified for memory)
			if tvs, ok := r.trackingValues[m.ID]; ok {
				m.TrackingValues = tvs
			}
			filtered = append(filtered, m)
		}
	}
	total := int64(len(filtered))
	start := int64(page.Offset())
	if start > total {
		start = total
	}
	end := start + int64(page.LimitClamped())
	if end > total {
		end = total
	}
	return pagination.NewPageResult(filtered[start:end], total, page), nil
}

func (r *MemoryRepo) CreateTrackingValues(ctx context.Context, values []activity.TrackingValue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(values) == 0 {
		return nil
	}
	msgID := values[0].MessageID
	r.trackingValues[msgID] = append(r.trackingValues[msgID], values...)
	return nil
}

func (r *MemoryRepo) ListTrackingValues(ctx context.Context, messageID int64) ([]activity.TrackingValue, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.trackingValues[messageID], nil
}

// --- FollowerRepository ---

func (r *MemoryRepo) CreateFollower(ctx context.Context, f *activity.Follower) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f.ID = r.nextFollowerID
	r.nextFollowerID++
	r.followers[f.ID] = *f
	return nil
}

func (r *MemoryRepo) DeleteFollower(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.followers, id)
	return nil
}

func (r *MemoryRepo) ListFollowers(ctx context.Context, resModel string, resID int64) ([]activity.Follower, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []activity.Follower
	for _, f := range r.followers {
		if f.ResModel == resModel && f.ResID == resID {
			res = append(res, f)
		}
	}
	return res, nil
}

func (r *MemoryRepo) GetFollowersForNotification(ctx context.Context, resModel string, resID int64, subtypeID int64) ([]activity.Follower, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []activity.Follower
	for _, f := range r.followers {
		if f.ResModel == resModel && f.ResID == resID {
			if subtypeID == 0 {
				res = append(res, f)
				continue
			}
			for _, sid := range f.SubtypeIDs {
				if sid == subtypeID {
					res = append(res, f)
					break
				}
			}
		}
	}
	return res, nil
}

// --- SubtypeRepository ---

func (r *MemoryRepo) GetSubtypeByID(ctx context.Context, id int64) (*activity.MessageSubtype, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.subtypes[id]
	if !ok {
		return nil, platformerrors.NotFound("subtype not found")
	}
	return &s, nil
}

func (r *MemoryRepo) GetSubtypeByName(ctx context.Context, resModel string, name string) (*activity.MessageSubtype, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.subtypes {
		if s.Name == name && (s.ResModel == "" || s.ResModel == resModel) {
			return &s, nil
		}
	}
	return nil, platformerrors.NotFound("subtype not found")
}

func (r *MemoryRepo) ListSubtypes(ctx context.Context, resModel string) ([]activity.MessageSubtype, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []activity.MessageSubtype
	for _, s := range r.subtypes {
		if s.ResModel == "" || s.ResModel == resModel {
			res = append(res, s)
		}
	}
	return res, nil
}

// --- NotificationRepository ---

func (r *MemoryRepo) CreateNotification(ctx context.Context, n *activity.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n.ID = r.nextNotifID
	r.nextNotifID++
	n.CreatedAt = time.Now().UTC()
	n.UpdatedAt = n.CreatedAt
	r.notifs[n.ID] = *n
	return nil
}

func (r *MemoryRepo) GetNotificationByID(ctx context.Context, id int64) (*activity.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.notifs[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("notification %d not found", id))
	}
	return &n, nil
}

func (r *MemoryRepo) UpdateNotification(ctx context.Context, n *activity.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.notifs[n.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("notification %d not found", n.ID))
	}
	n.UpdatedAt = time.Now().UTC()
	r.notifs[n.ID] = *n
	return nil
}

func (r *MemoryRepo) ListNotificationsForUser(ctx context.Context, userID int64, status *activity.NotificationStatus, page pagination.PageRequest) (pagination.PageResult[activity.Notification], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []activity.Notification
	for _, n := range r.notifs {
		if n.RecipientUserID == userID && (status == nil || n.Status == *status) {
			filtered = append(filtered, n)
		}
	}
	total := int64(len(filtered))
	start := int64(page.Offset())
	if start > total {
		start = total
	}
	end := start + int64(page.LimitClamped())
	if end > total {
		end = total
	}
	return pagination.NewPageResult(filtered[start:end], total, page), nil
}

func (r *MemoryRepo) MarkAllRead(ctx context.Context, userID int64, companyID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for id, n := range r.notifs {
		if n.RecipientUserID == userID && n.CompanyID == companyID && n.Status == activity.NotificationStatusUnread {
			n.Status = activity.NotificationStatusRead
			n.ReadAt = &now
			n.UpdatedAt = now
			r.notifs[id] = n
		}
	}
	return nil
}

// --- EmailQueueRepository ---

func (r *MemoryRepo) PushEmail(ctx context.Context, e *activity.EmailQueueItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e.ID = r.nextEmailID
	r.nextEmailID++
	e.CreatedAt = time.Now().UTC()
	e.UpdatedAt = e.CreatedAt
	r.emails[e.ID] = *e
	return nil
}

func (r *MemoryRepo) PopEmails(ctx context.Context, limit int) ([]activity.EmailQueueItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []activity.EmailQueueItem
	now := time.Now().UTC()
	count := 0
	for id, e := range r.emails {
		if (e.Status == activity.EmailStatusQueued || e.Status == activity.EmailStatusFailed) && e.NextAttemptAt.Before(now) {
			e.Status = activity.EmailStatusProcessing
			e.UpdatedAt = now
			r.emails[id] = e
			res = append(res, e)
			count++
			if count >= limit {
				break
			}
		}
	}
	return res, nil
}

func (r *MemoryRepo) UpdateEmail(ctx context.Context, e *activity.EmailQueueItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.emails[e.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("email item %d not found", e.ID))
	}
	e.UpdatedAt = time.Now().UTC()
	r.emails[e.ID] = *e
	return nil
}
