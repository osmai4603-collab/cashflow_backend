package livechatusecase

import (
	"context"
	"testing"

	livechatstorage "cashflow_backend/internal/adapters/storage/livechat"
	"cashflow_backend/internal/domain/livechat"
)

func TestLiveChatUseCase_SessionLifecycle(t *testing.T) {
	repo := livechatstorage.NewMemoryRepo()
	channel := &livechat.Channel{ID: 1, Name: "Support", CompanyID: 1, Active: true, OperatorIDs: []int64{42}}
	if err := repo.CreateChannel(context.Background(), channel); err != nil {
		t.Fatalf("create channel: %v", err)
	}
	uc := New(repo)
	ctx := context.Background()

	session, err := uc.InitSession(ctx, 1, "visitor-1", "Visitor", "visitor@example.com")
	if err != nil {
		t.Fatalf("init session: %v", err)
	}
	if session.ID == 0 || session.Status != livechat.SessionStatusActive {
		t.Fatalf("unexpected session: %#v", session)
	}
	if err := uc.JoinSession(ctx, session.ID, 42); err != nil {
		t.Fatalf("join session: %v", err)
	}
	message, err := uc.SendMessage(ctx, session.ID, livechat.SenderTypeVisitor, nil, "Hello", "")
	if err != nil || message.ID == 0 {
		t.Fatalf("send message: %v, %#v", err, message)
	}
	if err := uc.RateSession(ctx, session.ID, 5, "Helpful"); err != nil {
		t.Fatalf("rate session: %v", err)
	}
	if err := uc.CloseSession(ctx, session.ID); err != nil {
		t.Fatalf("close session: %v", err)
	}
	if _, err := uc.SendMessage(ctx, session.ID, livechat.SenderTypeVisitor, nil, "After close", ""); err == nil {
		t.Fatal("expected closed session to reject messages")
	}
}
