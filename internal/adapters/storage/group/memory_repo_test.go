package groupstorage

import (
	"context"
	"testing"

	domaingroup "cashflow_backend/internal/domain/group"
)

func TestMemoryRepoEffectiveGroupIDsIncludesImpliedGroups(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	base := &domaingroup.Group{Name: "Base"}
	manager := &domaingroup.Group{Name: "Manager"}
	if err := repo.Create(ctx, base); err != nil {
		t.Fatalf("create base group: %v", err)
	}
	if err := repo.Create(ctx, manager); err != nil {
		t.Fatalf("create manager group: %v", err)
	}
	if err := repo.AddImpliedGroup(ctx, manager.ID, base.ID); err != nil {
		t.Fatalf("add implied group: %v", err)
	}
	if err := repo.AssignUserToGroup(ctx, 10, manager.ID); err != nil {
		t.Fatalf("assign user: %v", err)
	}

	groupIDs, err := repo.GetEffectiveGroupIDs(ctx, 10)
	if err != nil {
		t.Fatalf("get effective groups: %v", err)
	}
	want := []int64{base.ID, manager.ID}
	if len(groupIDs) != len(want) || groupIDs[0] != want[0] || groupIDs[1] != want[1] {
		t.Fatalf("effective groups = %v, want %v", groupIDs, want)
	}
}

func TestMemoryRepoRejectsImpliedGroupCycle(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	first := &domaingroup.Group{Name: "First"}
	second := &domaingroup.Group{Name: "Second"}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first group: %v", err)
	}
	if err := repo.Create(ctx, second); err != nil {
		t.Fatalf("create second group: %v", err)
	}
	if err := repo.AddImpliedGroup(ctx, first.ID, second.ID); err != nil {
		t.Fatalf("add first implied group: %v", err)
	}
	if err := repo.AddImpliedGroup(ctx, second.ID, first.ID); err == nil {
		t.Fatal("expected cycle to be rejected")
	}
}

func TestMemoryRepoCheckAccessUsesImpliedGroups(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	base := &domaingroup.Group{Name: "Base"}
	manager := &domaingroup.Group{Name: "Manager"}
	if err := repo.Create(ctx, base); err != nil {
		t.Fatalf("create base group: %v", err)
	}
	if err := repo.Create(ctx, manager); err != nil {
		t.Fatalf("create manager group: %v", err)
	}
	if err := repo.AddImpliedGroup(ctx, manager.ID, base.ID); err != nil {
		t.Fatalf("add implied group: %v", err)
	}
	if err := repo.AssignUserToGroup(ctx, 20, manager.ID); err != nil {
		t.Fatalf("assign user: %v", err)
	}
	if err := repo.CreatePermission(ctx, &domaingroup.Permission{GroupID: base.ID, Model: "sale.order", CanRead: true}); err != nil {
		t.Fatalf("create permission: %v", err)
	}

	allowed, err := repo.CheckAccess(ctx, 20, "sale.order", "read")
	if err != nil {
		t.Fatalf("check access: %v", err)
	}
	if !allowed {
		t.Fatal("expected inherited permission to be allowed")
	}
}

func TestMemoryRepoCheckAccessActionAliases(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()

	g := &domaingroup.Group{Name: "Finance"}
	if err := repo.Create(ctx, g); err != nil {
		t.Fatalf("create group: %v", err)
	}
	perm := &domaingroup.Permission{
		GroupID:   g.ID,
		Model:     "account.bank.statement",
		CanRead:   true,
		CanCreate: true,
		CanUpdate: true,
		CanDelete: false,
	}
	if err := repo.CreatePermission(ctx, perm); err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := repo.AssignUserToGroup(ctx, 30, g.ID); err != nil {
		t.Fatalf("assign user: %v", err)
	}

	for _, action := range []string{"read", "create", "update", "write"} {
		allowed, err := repo.CheckAccess(ctx, 30, "account.bank.statement", action)
		if err != nil {
			t.Fatalf("check access(%s): %v", action, err)
		}
		if !allowed {
			t.Errorf("expected action %q to be allowed via aliasing", action)
		}
	}
	for _, action := range []string{"delete", "unlink"} {
		allowed, err := repo.CheckAccess(ctx, 30, "account.bank.statement", action)
		if err != nil {
			t.Fatalf("check access(%s): %v", action, err)
		}
		if allowed {
			t.Errorf("expected action %q to be denied (can_delete=false)", action)
		}
	}
}
