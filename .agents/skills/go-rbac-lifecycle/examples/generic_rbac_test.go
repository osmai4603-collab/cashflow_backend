package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEngine_DirectAndHierarchicalPermissions(t *testing.T) {
	engine := NewEngine()

	const (
		PermRead   Permission = "data:read"
		PermWrite  Permission = "data:write"
		PermDelete Permission = "data:delete"
	)

	const (
		RoleReader Role = "reader"
		RoleWriter Role = "writer"
		RoleAdmin  Role = "admin"
	)

	engine.DefineRole(RoleReader, PermRead)
	engine.DefineRole(RoleWriter, PermWrite)
	engine.DefineRole(RoleAdmin, PermDelete)

	engine.AddInheritance(RoleWriter, RoleReader)
	engine.AddInheritance(RoleAdmin, RoleWriter)

	// 1. Reader has only PermRead
	if !engine.HasPermission([]Role{RoleReader}, PermRead) {
		t.Errorf("expected Reader to have PermRead")
	}
	if engine.HasPermission([]Role{RoleReader}, PermWrite) {
		t.Errorf("expected Reader NOT to have PermWrite")
	}

	// 2. Writer has PermWrite AND inherits PermRead
	if !engine.HasPermission([]Role{RoleWriter}, PermWrite) {
		t.Errorf("expected Writer to have PermWrite")
	}
	if !engine.HasPermission([]Role{RoleWriter}, PermRead) {
		t.Errorf("expected Writer to inherit PermRead")
	}
	if engine.HasPermission([]Role{RoleWriter}, PermDelete) {
		t.Errorf("expected Writer NOT to have PermDelete")
	}

	// 3. Admin has PermDelete AND inherits PermWrite AND PermRead
	if !engine.HasPermission([]Role{RoleAdmin}, PermDelete) {
		t.Errorf("expected Admin to have PermDelete")
	}
	if !engine.HasPermission([]Role{RoleAdmin}, PermWrite) {
		t.Errorf("expected Admin to inherit PermWrite")
	}
	if !engine.HasPermission([]Role{RoleAdmin}, PermRead) {
		t.Errorf("expected Admin to inherit PermRead")
	}

	// 4. Unknown Role -> Default Deny
	if engine.HasPermission([]Role{"stranger"}, PermRead) {
		t.Errorf("expected unknown role to be denied")
	}
	if engine.HasPermission(nil, PermRead) {
		t.Errorf("expected empty roles to be denied")
	}
}

func TestEngine_StaticSeparationOfDuties(t *testing.T) {
	engine := NewEngine()

	const (
		RoleCreator  Role = "creator"
		RoleApprover Role = "approver"
		RoleNormal   Role = "normal"
	)

	engine.RegisterSSDConflict(RoleCreator, RoleApprover)

	// Valid assignment
	if err := engine.ValidateSSD([]Role{RoleCreator, RoleNormal}); err != nil {
		t.Fatalf("expected valid assignment, got error: %v", err)
	}

	// Conflicting assignment
	if err := engine.ValidateSSD([]Role{RoleCreator, RoleApprover}); err == nil {
		t.Fatalf("expected SSD conflict error, got nil")
	}
}

func TestEngine_SymmetricRBAC_WhoCanPerform(t *testing.T) {
	engine := NewEngine()

	const (
		PermAudit Permission = "audit:export"
	)
	const (
		RoleAuditor Role = "auditor"
		RoleLead    Role = "lead_auditor"
		RoleGuest   Role = "guest"
	)

	engine.DefineRole(RoleAuditor, PermAudit)
	engine.DefineRole(RoleGuest)
	engine.AddInheritance(RoleLead, RoleAuditor)

	capable := engine.WhoCanPerform(PermAudit)
	rolesMap := make(map[Role]bool)
	for _, r := range capable {
		rolesMap[r] = true
	}

	if !rolesMap[RoleAuditor] {
		t.Errorf("expected Auditor in WhoCanPerform list")
	}
	if !rolesMap[RoleLead] {
		t.Errorf("expected Lead Auditor in WhoCanPerform list via inheritance")
	}
	if rolesMap[RoleGuest] {
		t.Errorf("expected Guest NOT to be in WhoCanPerform list")
	}
}

func TestRequirePermission_PEPGuard(t *testing.T) {
	engine := NewEngine()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	const PermSecret Permission = "secrets:read"
	const RoleAgent Role = "secret_agent"

	engine.DefineRole(RoleAgent, PermSecret)

	targetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("classified"))
	})

	guard := RequirePermission(engine, logger, PermSecret)(targetHandler)

	// Case 1: Unauthenticated request (no Subject in context) -> 401 Unauthorized
	req1 := httptest.NewRequest("GET", "/secret", nil)
	rec1 := httptest.NewRecorder()
	guard.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", rec1.Code)
	}

	// Case 2: Authenticated but insufficient role -> 403 Forbidden with RFC 7807
	subUnauthorized := Subject{ID: "usr_civilian", Roles: []Role{"civilian"}}
	req2 := httptest.NewRequest("GET", "/secret", nil)
	req2 = req2.WithContext(WithSubject(req2.Context(), subUnauthorized))
	rec2 := httptest.NewRecorder()
	guard.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unauthorized request, got %d", rec2.Code)
	}
	if rec2.Header().Get("Content-Type") != "application/problem+json" {
		t.Errorf("expected application/problem+json Content-Type, got %s", rec2.Header().Get("Content-Type"))
	}

	var prob ProblemDetails
	if err := json.NewDecoder(rec2.Body).Decode(&prob); err != nil {
		t.Fatalf("failed to decode RFC 7807 problem details: %v", err)
	}
	if prob.Status != http.StatusForbidden {
		t.Errorf("expected problem status 403, got %d", prob.Status)
	}

	// Case 3: Authorized request -> 200 OK
	subAuthorized := Subject{ID: "usr_bond", Roles: []Role{RoleAgent}}
	req3 := httptest.NewRequest("GET", "/secret", nil)
	req3 = req3.WithContext(WithSubject(req3.Context(), subAuthorized))
	rec3 := httptest.NewRecorder()
	guard.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200 for authorized request, got %d", rec3.Code)
	}
}

func TestEngine_ConcurrencySafety(t *testing.T) {
	engine := NewEngine()

	const PermExec Permission = "service:execute"
	const RoleWorker Role = "worker"
	engine.DefineRole(RoleWorker, PermExec)

	var wg sync.WaitGroup
	workers := 50
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Alternating read checks
				_ = engine.HasPermission([]Role{RoleWorker}, PermExec)
				_ = engine.ResolveEffectivePermissions([]Role{RoleWorker})

				// Periodic dynamic role definition to test write contention
				if workerID == 0 && j%20 == 0 {
					engine.DefineRole(Role("temp_role"), PermExec)
				}
			}
		}(i)
	}

	wg.Wait()
}
