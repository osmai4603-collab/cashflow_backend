package activityusecase

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/user"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

type fakeActivityRepo struct{}

type fakeTypeRepo struct{}

type fakeMsgRepo struct{}

type fakeNotifRepo struct{}

type fakeEmailRepo struct{ items []activity.EmailQueueItem }

type fakeUserRepo struct{ user *user.User }

func (r *fakeActivityRepo) CreateActivity(ctx context.Context, value *activity.Activity) error {
	return nil
}
func (r *fakeActivityRepo) GetActivityByID(ctx context.Context, id int64) (*activity.Activity, error) {
	return nil, platformerrors.NotFound("missing")
}
func (r *fakeActivityRepo) UpdateActivity(ctx context.Context, value *activity.Activity) error {
	return nil
}
func (r *fakeActivityRepo) CompleteActivity(ctx context.Context, id int64, now time.Time, feedback string) (*activity.Activity, error) {
	return nil, platformerrors.NotFound("missing")
}
func (r *fakeActivityRepo) ListActivities(ctx context.Context, filter activity.ActivityFilter) (pagination.PageResult[activity.Activity], error) {
	return pagination.PageResult[activity.Activity]{}, nil
}

func (r *fakeTypeRepo) CreateType(ctx context.Context, value *activity.ActivityType) error {
	return nil
}
func (r *fakeTypeRepo) GetTypeByID(ctx context.Context, id int64) (*activity.ActivityType, error) {
	return &activity.ActivityType{ID: id, Name: "Follow up", Summary: "Follow up"}, nil
}
func (r *fakeTypeRepo) UpdateType(ctx context.Context, value *activity.ActivityType) error {
	return nil
}
func (r *fakeTypeRepo) DeleteType(ctx context.Context, id int64) error { return nil }
func (r *fakeTypeRepo) ListTypes(ctx context.Context, companyID *int64) ([]activity.ActivityType, error) {
	return nil, nil
}

func (r *fakeMsgRepo) CreateMessage(ctx context.Context, value *activity.Message) error {
	return nil
}
func (r *fakeMsgRepo) GetMessageByID(ctx context.Context, id int64) (*activity.Message, error) {
	return nil, platformerrors.NotFound("missing")
}
func (r *fakeMsgRepo) ListMessagesByResource(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[activity.Message], error) {
	return pagination.PageResult[activity.Message]{}, nil
}

func (r *fakeNotifRepo) CreateNotification(ctx context.Context, value *activity.Notification) error {
	return nil
}
func (r *fakeNotifRepo) GetNotificationByID(ctx context.Context, id int64) (*activity.Notification, error) {
	return nil, platformerrors.NotFound("missing")
}
func (r *fakeNotifRepo) UpdateNotification(ctx context.Context, n *activity.Notification) error {
	return nil
}
func (r *fakeNotifRepo) ListNotificationsForUser(ctx context.Context, userID int64, status *activity.NotificationStatus, page pagination.PageRequest) (pagination.PageResult[activity.Notification], error) {
	return pagination.PageResult[activity.Notification]{}, nil
}
func (r *fakeNotifRepo) MarkAllRead(ctx context.Context, userID int64, companyID int64) error {
	return nil
}

func (r *fakeEmailRepo) PushEmail(ctx context.Context, value *activity.EmailQueueItem) error {
	r.items = append(r.items, *value)
	return nil
}
func (r *fakeEmailRepo) PopEmails(ctx context.Context, limit int) ([]activity.EmailQueueItem, error) {
	return nil, nil
}
func (r *fakeEmailRepo) UpdateEmail(ctx context.Context, value *activity.EmailQueueItem) error {
	return nil
}

func (r *fakeUserRepo) Create(ctx context.Context, u *user.User) error { return nil }
func (r *fakeUserRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	if r.user == nil {
		return nil, platformerrors.NotFound("missing")
	}
	return r.user, nil
}
func (r *fakeUserRepo) GetByLogin(ctx context.Context, login string) (*user.User, error) {
	return nil, platformerrors.NotFound("missing")
}
func (r *fakeUserRepo) Update(ctx context.Context, u *user.User) error { return nil }
func (r *fakeUserRepo) Delete(ctx context.Context, id int64) error     { return nil }
func (r *fakeUserRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[user.User], error) {
	return pagination.PageResult[user.User]{}, nil
}
func (r *fakeUserRepo) UpdateLastLogin(ctx context.Context, userID int64) error { return nil }

type fakeBus struct{}

func (b *fakeBus) NotifyUser(ctx context.Context, userID int64, channel string, payload any) error {
	return nil
}

func TestUseCase_notifyAssignmentUsesResolvedUserEmail(t *testing.T) {
	msgRepo := &fakeMsgRepo{}
	notifRepo := &fakeNotifRepo{}
	emailRepo := &fakeEmailRepo{}
	userRepo := &fakeUserRepo{user: &user.User{ID: 7, Email: "assignee@example.com", EmailNotificationsEnabled: true, CompanyID: 9, Login: "assignee", Name: "Assignee"}}
	uc := &UseCase{
		repo:      &fakeActivityRepo{},
		typeRepo:  &fakeTypeRepo{},
		msgRepo:   msgRepo,
		notifRepo: notifRepo,
		emailRepo: emailRepo,
		userRepo:  userRepo,
		bus:       &fakeBus{},
	}

	a := &activity.Activity{
		ActivityTypeID: 1,
		Summary:        "Review invoice",
		DateDeadline:   time.Now().UTC().Add(24 * time.Hour),
		AssignedUserID: 7,
		CompanyID:      9,
		CreatedBy:      ptrInt64(3),
	}

	uc.notifyAssignment(context.Background(), a)

	if len(emailRepo.items) != 1 {
		t.Fatalf("expected 1 queued email, got %d", len(emailRepo.items))
	}
	if emailRepo.items[0].RecipientEmail != "assignee@example.com" {
		t.Fatalf("expected email recipient to come from user record, got %q", emailRepo.items[0].RecipientEmail)
	}
}

func TestUseCase_notifyAssignmentSkipsEmailWhenUserOptedOut(t *testing.T) {
	emailRepo := &fakeEmailRepo{}
	uc := &UseCase{
		msgRepo:   &fakeMsgRepo{},
		notifRepo: &fakeNotifRepo{},
		emailRepo: emailRepo,
		userRepo:  &fakeUserRepo{user: &user.User{ID: 7, Email: "assignee@example.com"}},
	}

	uc.notifyAssignment(context.Background(), &activity.Activity{
		Summary:        "Review invoice",
		AssignedUserID: 7,
		CompanyID:      9,
	})

	if len(emailRepo.items) != 0 {
		t.Fatalf("expected no queued email for opted-out user, got %d", len(emailRepo.items))
	}
}

func ptrInt64(v int64) *int64 { return &v }
