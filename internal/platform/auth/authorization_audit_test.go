package auth

import (
	"context"
	"testing"

	groupstorage "cashflow_backend/internal/adapters/storage/group"
	domaingroup "cashflow_backend/internal/domain/group"
	"cashflow_backend/internal/platform/audit"
	domainerrors "cashflow_backend/internal/platform/errors"
)

type decisionSink struct {
	decisions []audit.AuthorizationDecision
}

func (s *decisionSink) RecordAuthorizationDecision(decision audit.AuthorizationDecision) {
	s.decisions = append(s.decisions, decision)
}

func TestACLAuthorizerRecordsAllowAndDeny(t *testing.T) {
	repo := groupstorage.NewMemoryRepo()
	group := &domaingroup.Group{Name: "Users"}
	if err := repo.Create(context.Background(), group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := repo.AssignUserToGroup(context.Background(), 10, group.ID); err != nil {
		t.Fatalf("assign user: %v", err)
	}
	if err := repo.CreatePermission(context.Background(), &domaingroup.Permission{GroupID: group.ID, Model: "res.partner", CanRead: true}); err != nil {
		t.Fatalf("create permission: %v", err)
	}
	sink := &decisionSink{}
	authorizer := NewACLAuthorizerWithSink(repo, sink)
	subject := Subject{UserID: 10, CompanyID: 2}

	if err := authorizer.Check(context.Background(), subject, "res.partner", ActionRead); err != nil {
		t.Fatalf("allowed check: %v", err)
	}
	if err := authorizer.Check(context.Background(), subject, "res.company", ActionWrite); err == nil || domainerrors.HTTPStatus(err) != 403 {
		t.Fatalf("expected denied check, got %v", err)
	}
	if len(sink.decisions) != 2 {
		t.Fatalf("decision count = %d, want 2", len(sink.decisions))
	}
	if sink.decisions[0].Event != "authorization.allowed" || !sink.decisions[0].Allowed {
		t.Fatalf("unexpected allow event: %+v", sink.decisions[0])
	}
	if sink.decisions[1].Event != "authorization.denied" || sink.decisions[1].Allowed {
		t.Fatalf("unexpected deny event: %+v", sink.decisions[1])
	}
}
