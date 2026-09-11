package activityusecase

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/domain/project"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/platform/pagination"
)

type memoryMsgRepo struct {
	messages       []*activity.Message
	trackingValues []activity.TrackingValue
	nextID         int64
}

func newMemoryMsgRepo() *memoryMsgRepo {
	return &memoryMsgRepo{nextID: 1}
}

func (r *memoryMsgRepo) CreateMessage(ctx context.Context, value *activity.Message) error {
	value.ID = r.nextID
	r.nextID++
	r.messages = append(r.messages, value)
	return nil
}

func (r *memoryMsgRepo) GetMessageByID(ctx context.Context, id int64) (*activity.Message, error) {
	for _, m := range r.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, nil
}

func (r *memoryMsgRepo) ListMessagesByResource(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[activity.Message], error) {
	var items []activity.Message
	for _, m := range r.messages {
		if m.ResModel == resModel && m.ResID != nil && *m.ResID == resID {
			items = append(items, *m)
		}
	}
	return pagination.PageResult[activity.Message]{
		Items:      items,
		TotalItems: int64(len(items)),
	}, nil
}

func (r *memoryMsgRepo) CreateTrackingValues(ctx context.Context, values []activity.TrackingValue) error {
	r.trackingValues = append(r.trackingValues, values...)
	return nil
}

func (r *memoryMsgRepo) ListTrackingValues(ctx context.Context, messageID int64) ([]activity.TrackingValue, error) {
	var res []activity.TrackingValue
	for _, tv := range r.trackingValues {
		if tv.MessageID == messageID {
			res = append(res, tv)
		}
	}
	return res, nil
}

type memoryFollowerRepo struct {
	followers []*activity.Follower
	nextID    int64
}

func newMemoryFollowerRepo() *memoryFollowerRepo {
	return &memoryFollowerRepo{nextID: 1}
}

func (r *memoryFollowerRepo) CreateFollower(ctx context.Context, value *activity.Follower) error {
	value.ID = r.nextID
	r.nextID++
	r.followers = append(r.followers, value)
	return nil
}

func (r *memoryFollowerRepo) DeleteFollower(ctx context.Context, id int64) error {
	var remaining []*activity.Follower
	for _, f := range r.followers {
		if f.ID != id {
			remaining = append(remaining, f)
		}
	}
	r.followers = remaining
	return nil
}

func (r *memoryFollowerRepo) ListFollowers(ctx context.Context, resModel string, resID int64) ([]activity.Follower, error) {
	var items []activity.Follower
	for _, f := range r.followers {
		if f.ResModel == resModel && f.ResID == resID {
			items = append(items, *f)
		}
	}
	return items, nil
}

func (r *memoryFollowerRepo) GetFollowersForNotification(ctx context.Context, resModel string, resID int64, subtypeID int64) ([]activity.Follower, error) {
	var items []activity.Follower
	for _, f := range r.followers {
		if f.ResModel == resModel && f.ResID == resID {
			items = append(items, *f)
		}
	}
	return items, nil
}

type dummyThreadable struct {
	model     string
	id        int64
	companyID int64
}

func (d *dummyThreadable) ThreadModel() string    { return d.model }
func (d *dummyThreadable) ThreadID() int64        { return d.id }
func (d *dummyThreadable) ThreadCompanyID() int64 { return d.companyID }

func TestThreadService_PostMessage(t *testing.T) {
	msgRepo := newMemoryMsgRepo()
	followerRepo := newMemoryFollowerRepo()
	service := NewThreadService(nil, msgRepo, followerRepo, nil, nil, nil, nil)

	ctx := context.Background()
	thread := &dummyThreadable{
		model:     "sale.order",
		id:        42,
		companyID: 10,
	}

	authorID := int64(99)
	msg, err := service.PostMessage(ctx, thread, PostMessageInput{
		Subject:     "Order Confirmed",
		Body:        "The sale order was confirmed.",
		MessageType: activity.MessageTypeComment,
		AuthorID:    &authorID,
		TrackingValues: []activity.TrackingValue{
			{Field: "state", FieldName: "Status", OldValueText: "draft", NewValueText: "sale"},
		},
	})

	if err != nil {
		t.Fatalf("expected no error posting message, got %v", err)
	}
	if msg.ID <= 0 {
		t.Errorf("expected positive message ID, got %d", msg.ID)
	}
	if msg.ResModel != "sale.order" {
		t.Errorf("expected ResModel 'sale.order', got '%s'", msg.ResModel)
	}
	if msg.ResID == nil || *msg.ResID != 42 {
		t.Errorf("expected ResID 42, got %v", msg.ResID)
	}
	if msg.CompanyID != 10 {
		t.Errorf("expected CompanyID 10, got %d", msg.CompanyID)
	}
	if len(msgRepo.trackingValues) != 1 {
		t.Fatalf("expected 1 tracking value, got %d", len(msgRepo.trackingValues))
	}
	if msgRepo.trackingValues[0].MessageID != msg.ID {
		t.Errorf("expected tracking value message ID %d, got %d", msg.ID, msgRepo.trackingValues[0].MessageID)
	}

	// Test nil thread validation
	_, err = service.PostMessage(ctx, nil, PostMessageInput{Body: "invalid"})
	if err == nil {
		t.Error("expected error for nil thread, got nil")
	}
}

func TestThreadService_Followers(t *testing.T) {
	msgRepo := newMemoryMsgRepo()
	followerRepo := newMemoryFollowerRepo()
	service := NewThreadService(nil, msgRepo, followerRepo, nil, nil, nil, nil)

	ctx := context.Background()
	thread := &dummyThreadable{
		model:     "purchase.order",
		id:        15,
		companyID: 2,
	}

	partnerID := int64(100)
	userID := int64(200)
	err := service.Subscribe(ctx, thread, &partnerID, &userID, []int64{1, 2})
	if err != nil {
		t.Fatalf("unexpected error subscribing: %v", err)
	}

	followers, err := service.ListFollowers(ctx, "purchase.order", 15)
	if err != nil {
		t.Fatalf("unexpected error listing followers: %v", err)
	}
	if len(followers) != 1 {
		t.Fatalf("expected 1 follower, got %d", len(followers))
	}
	if *followers[0].PartnerID != 100 || *followers[0].UserID != 200 {
		t.Errorf("follower partner/user mismatch: %+v", followers[0])
	}

	// Unsubscribe
	err = service.Unsubscribe(ctx, followers[0].ID)
	if err != nil {
		t.Fatalf("unexpected error unsubscribing: %v", err)
	}

	followersAfter, err := service.ListFollowers(ctx, "purchase.order", 15)
	if err != nil {
		t.Fatalf("unexpected error listing followers: %v", err)
	}
	if len(followersAfter) != 0 {
		t.Errorf("expected 0 followers after unsubscribe, got %d", len(followersAfter))
	}
}

func TestTrackingService_TrackChanges(t *testing.T) {
	msgRepo := newMemoryMsgRepo()
	followerRepo := newMemoryFollowerRepo()
	threadService := NewThreadService(nil, msgRepo, followerRepo, nil, nil, nil, nil)
	trackingService := NewTrackingService(threadService, nil)

	ctx := context.Background()
	thread := &dummyThreadable{
		model:     "account.move",
		id:        77,
		companyID: 1,
	}

	// Case 1: No actual changes
	err := trackingService.TrackChanges(ctx, thread, 5, []TrackedField{
		{Name: "state", FieldDesc: "Status", OldValue: "draft", NewValue: "draft"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgRepo.messages) != 0 {
		t.Errorf("expected no message posted when values unchanged, got %d", len(msgRepo.messages))
	}

	// Case 2: Changed fields
	err = trackingService.TrackChanges(ctx, thread, 5, []TrackedField{
		{Name: "state", FieldDesc: "Status", OldValue: "draft", NewValue: "posted"},
		{Name: "amount_total", FieldDesc: "Total Amount", OldValue: "100.00", NewValue: "150.00"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgRepo.messages) != 1 {
		t.Fatalf("expected 1 message posted, got %d", len(msgRepo.messages))
	}
	msg := msgRepo.messages[0]
	if msg.MessageType != activity.MessageTypeNotification {
		t.Errorf("expected MessageTypeNotification, got '%s'", msg.MessageType)
	}
	if len(msgRepo.trackingValues) != 2 {
		t.Fatalf("expected 2 tracking values recorded, got %d", len(msgRepo.trackingValues))
	}
	if msgRepo.trackingValues[0].Field != "state" || msgRepo.trackingValues[0].OldValueText != "draft" || msgRepo.trackingValues[0].NewValueText != "posted" {
		t.Errorf("unexpected tracking value 0: %+v", msgRepo.trackingValues[0])
	}
	if msgRepo.trackingValues[1].Field != "amount_total" || msgRepo.trackingValues[1].OldValueText != "100.00" || msgRepo.trackingValues[1].NewValueText != "150.00" {
		t.Errorf("unexpected tracking value 1: %+v", msgRepo.trackingValues[1])
	}
}

func TestThreadableEntities_SatisfyInterface(t *testing.T) {
	companyID := int64(5)
	now := time.Now().UTC()

	// 1. SaleOrder
	so := &sale.SaleOrder{ID: 101, CompanyID: &companyID}
	var thread activity.Threadable = so
	if thread.ThreadModel() != "sale.order" || thread.ThreadID() != 101 || thread.ThreadCompanyID() != 5 {
		t.Errorf("SaleOrder Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 2. PurchaseOrder
	poCompanyID := int64(6)
	po := &purchase.PurchaseOrder{ID: 102, CompanyID: &poCompanyID}
	thread = po
	if thread.ThreadModel() != "purchase.order" || thread.ThreadID() != 102 || thread.ThreadCompanyID() != 6 {
		t.Errorf("PurchaseOrder Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 3. AccountMove
	am := &accounting.AccountMove{ID: 103}
	thread = am
	if thread.ThreadModel() != "account.move" || thread.ThreadID() != 103 || thread.ThreadCompanyID() != 1 {
		t.Errorf("AccountMove Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 4. Lead
	lead := &crm.Lead{ID: 104, CompanyID: &companyID}
	thread = lead
	if thread.ThreadModel() != "crm.lead" || thread.ThreadID() != 104 || thread.ThreadCompanyID() != 5 {
		t.Errorf("Lead Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 5. StockPicking
	picking := &stock.StockPicking{ID: 105, CompanyID: &companyID}
	thread = picking
	if thread.ThreadModel() != "stock.picking" || thread.ThreadID() != 105 || thread.ThreadCompanyID() != 5 {
		t.Errorf("StockPicking Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 6. Employee
	emp := &hr.Employee{ID: 106, CompanyID: &companyID}
	thread = emp
	if thread.ThreadModel() != "hr.employee" || thread.ThreadID() != 106 || thread.ThreadCompanyID() != 5 {
		t.Errorf("Employee Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 7. Task
	task := &project.Task{ID: 107, CompanyID: 8}
	thread = task
	if thread.ThreadModel() != "project.task" || thread.ThreadID() != 107 || thread.ThreadCompanyID() != 8 {
		t.Errorf("Task Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}

	// 8. MaintenanceRequest
	mr := &maintenance.MaintenanceRequest{ID: 108, CompanyID: 9, RequestDate: now}
	thread = mr
	if thread.ThreadModel() != "maintenance.request" || thread.ThreadID() != 108 || thread.ThreadCompanyID() != 9 {
		t.Errorf("MaintenanceRequest Threadable mismatch: %s, %d, %d", thread.ThreadModel(), thread.ThreadID(), thread.ThreadCompanyID())
	}
}
