package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEngine_DefaultDeny(t *testing.T) {
	engine := NewEngine(DenyOverrides, discardLogger())

	evalCtx := EvaluationContext{
		Subject:     Subject{ID: "usr_1", TenantID: "tenant_a"},
		Resource:    Resource{ID: "res_1", Type: "document", TenantID: "tenant_a"},
		Action:      Action{Verb: "read"},
		Environment: Environment{RequestTime: time.Now()},
	}

	decision, reason := engine.Evaluate(evalCtx)
	if decision != DecisionDeny {
		t.Fatalf("expected Default Deny when no rules exist, got %s (%s)", decision, reason)
	}
}

func TestEngine_CombiningAlgorithms(t *testing.T) {
	ruleAllow := RuleFunc{
		RuleID: "rule_allow_all",
		Desc:   "Permits read verb",
		TargetFunc: func(ctx EvaluationContext) bool {
			return ctx.Action.Verb == "read"
		},
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			return DecisionPermit, "rule allows read"
		},
	}

	ruleDenyDraft := RuleFunc{
		RuleID: "rule_deny_draft",
		Desc:   "Denies access if resource is draft",
		TargetFunc: func(ctx EvaluationContext) bool {
			return ctx.Resource.Status == "draft"
		},
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			return DecisionDeny, "draft resources cannot be read"
		},
	}

	evalCtx := EvaluationContext{
		Subject:     Subject{ID: "usr_1", TenantID: "tenant_a"},
		Resource:    Resource{ID: "res_1", Type: "document", Status: "draft", TenantID: "tenant_a"},
		Action:      Action{Verb: "read"},
		Environment: Environment{RequestTime: time.Now()},
	}

	// 1. Deny-Overrides: Permit + Deny => Deny
	engineDenyOverrides := NewEngine(DenyOverrides, discardLogger(), ruleAllow, ruleDenyDraft)
	dec1, reason1 := engineDenyOverrides.Evaluate(evalCtx)
	if dec1 != DecisionDeny {
		t.Errorf("expected DenyOverrides to produce Deny, got %s: %s", dec1, reason1)
	}

	// 2. Permit-Overrides: Permit + Deny => Permit
	enginePermitOverrides := NewEngine(PermitOverrides, discardLogger(), ruleAllow, ruleDenyDraft)
	dec2, reason2 := enginePermitOverrides.Evaluate(evalCtx)
	if dec2 != DecisionPermit {
		t.Errorf("expected PermitOverrides to produce Permit, got %s: %s", dec2, reason2)
	}

	// 3. First-Applicable with [ruleAllow, ruleDenyDraft] => Permit
	engineFirstApp1 := NewEngine(FirstApplicable, discardLogger(), ruleAllow, ruleDenyDraft)
	dec3, _ := engineFirstApp1.Evaluate(evalCtx)
	if dec3 != DecisionPermit {
		t.Errorf("expected FirstApplicable with Allow-first to produce Permit, got %s", dec3)
	}

	// 4. First-Applicable with [ruleDenyDraft, ruleAllow] => Deny
	engineFirstApp2 := NewEngine(FirstApplicable, discardLogger(), ruleDenyDraft, ruleAllow)
	dec4, _ := engineFirstApp2.Evaluate(evalCtx)
	if dec4 != DecisionDeny {
		t.Errorf("expected FirstApplicable with Deny-first to produce Deny, got %s", dec4)
	}
}

func TestEngine_MultiTenant_BOLA_IDOR_Isolation(t *testing.T) {
	// Rule enforcing tenant isolation
	tenantRule := RuleFunc{
		RuleID: "enforce_tenant_isolation",
		Desc:   "Enforces strict multi-tenant boundary",
		TargetFunc: func(ctx EvaluationContext) bool {
			return true
		},
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			if ctx.Subject.TenantID == "" || ctx.Resource.TenantID == "" {
				return DecisionDeny, "missing tenant ID"
			}
			if ctx.Subject.TenantID != ctx.Resource.TenantID {
				return DecisionDeny, "cross-tenant access violation"
			}
			return DecisionPermit, "tenants match"
		},
	}

	engine := NewEngine(DenyOverrides, discardLogger(), tenantRule)

	// Cross-tenant access attempt (BOLA attack)
	maliciousCtx := EvaluationContext{
		Subject:     Subject{ID: "usr_tenant_1", TenantID: "tenant_1", Roles: []string{"super_admin"}},
		Resource:    Resource{ID: "doc_999", Type: "document", TenantID: "tenant_2"},
		Action:      Action{Verb: "read"},
		Environment: Environment{RequestTime: time.Now()},
	}

	decision, reason := engine.Evaluate(maliciousCtx)
	if decision != DecisionDeny {
		t.Fatalf("expected BOLA attempt to be strictly denied, got %s (%s)", decision, reason)
	}

	// Same tenant access
	legitCtx := EvaluationContext{
		Subject:     Subject{ID: "usr_tenant_1", TenantID: "tenant_1"},
		Resource:    Resource{ID: "doc_111", Type: "document", TenantID: "tenant_1"},
		Action:      Action{Verb: "read"},
		Environment: Environment{RequestTime: time.Now()},
	}

	decision2, _ := engine.Evaluate(legitCtx)
	if decision2 != DecisionPermit {
		t.Fatalf("expected same tenant access to be permitted, got %s", decision2)
	}
}

func TestEngine_EnvironmentalConstraints(t *testing.T) {
	timeRule := RuleFunc{
		RuleID: "block_weekends",
		Desc:   "Blocks access on Saturdays and Sundays",
		TargetFunc: func(ctx EvaluationContext) bool {
			return true
		},
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			weekday := ctx.Environment.RequestTime.Weekday()
			if weekday == time.Saturday || weekday == time.Sunday {
				return DecisionDeny, "access blocked on weekends"
			}
			return DecisionPermit, "weekday access allowed"
		},
	}

	engine := NewEngine(DenyOverrides, discardLogger(), timeRule)

	// Saturday test
	saturdayTime := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC) // Saturday
	ctxWeekend := EvaluationContext{
		Subject:     Subject{ID: "u1", TenantID: "t1"},
		Resource:    Resource{ID: "r1", TenantID: "t1"},
		Action:      Action{Verb: "update"},
		Environment: Environment{RequestTime: saturdayTime},
	}

	dec1, _ := engine.Evaluate(ctxWeekend)
	if dec1 != DecisionDeny {
		t.Fatalf("expected weekend request to be denied, got %s", dec1)
	}

	// Wednesday test
	wednesdayTime := time.Date(2026, time.September, 16, 12, 0, 0, 0, time.UTC) // Wednesday
	ctxWeekday := EvaluationContext{
		Subject:     Subject{ID: "u1", TenantID: "t1"},
		Resource:    Resource{ID: "r1", TenantID: "t1"},
		Action:      Action{Verb: "update"},
		Environment: Environment{RequestTime: wednesdayTime},
	}

	dec2, _ := engine.Evaluate(ctxWeekday)
	if dec2 != DecisionPermit {
		t.Fatalf("expected weekday request to be permitted, got %s", dec2)
	}
}

func TestPEP_Middleware_RFC7807(t *testing.T) {
	pip := NewInMemoryPIPResolver()
	pip.Put(Resource{
		ID:          "doc_42",
		Type:        "document",
		OwnerID:     "usr_alice",
		TenantID:    "tenant_corp",
		Sensitivity: 2,
	})

	allowOwnerOrClearance := RuleFunc{
		RuleID: "allow_owner_or_clearance",
		Desc:   "Allow if subject is owner or clearance >= sensitivity",
		TargetFunc: func(ctx EvaluationContext) bool {
			return ctx.Resource.Type == "document"
		},
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			if ctx.Subject.TenantID != ctx.Resource.TenantID {
				return DecisionDeny, "tenant mismatch"
			}
			if ctx.Subject.ID == ctx.Resource.OwnerID {
				return DecisionPermit, "subject is resource owner"
			}
			if ctx.Subject.Clearance >= ctx.Resource.Sensitivity {
				return DecisionPermit, "clearance level sufficient"
			}
			return DecisionDeny, "insufficient clearance"
		},
	}

	engine := NewEngine(DenyOverrides, discardLogger(), allowOwnerOrClearance)

	extractor := func(r *http.Request) (string, string, error) {
		return "document", "doc_42", nil
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	protectedHandler := RequireABAC(engine, "read", pip, extractor)(dummyHandler)

	// 1. Case: Unauthenticated context -> 403 RFC 7807
	req1 := httptest.NewRequest("GET", "/documents/doc_42", nil)
	w1 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for unauthenticated request, got %d", w1.Code)
	}
	if ct := w1.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %q", ct)
	}
	var prob1 ProblemDetails
	if err := json.Unmarshal(w1.Body.Bytes(), &prob1); err != nil {
		t.Fatalf("failed to decode ProblemDetails JSON: %v", err)
	}
	if prob1.Status != http.StatusForbidden {
		t.Errorf("expected ProblemDetails status 403, got %d", prob1.Status)
	}

	// 2. Case: Authenticated as stranger without clearance -> 403 RFC 7807
	stranger := Subject{
		ID:         "usr_bob",
		TenantID:   "tenant_corp",
		Clearance:  1, // Less than resource sensitivity 2
	}
	req2 := httptest.NewRequest("GET", "/documents/doc_42", nil)
	req2 = req2.WithContext(WithSubject(req2.Context(), stranger))
	w2 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for insufficient clearance, got %d", w2.Code)
	}

	// 3. Case: Authenticated as Owner -> 200 OK
	owner := Subject{
		ID:         "usr_alice",
		TenantID:   "tenant_corp",
		Clearance:  1,
	}
	req3 := httptest.NewRequest("GET", "/documents/doc_42", nil)
	req3 = req3.WithContext(WithSubject(req3.Context(), owner))
	w3 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 OK for resource owner, got %d", w3.Code)
	}
}

func TestEngine_ConcurrentRaceFree(t *testing.T) {
	rule1 := RuleFunc{
		RuleID: "rule_1",
		Desc:   "rule 1",
		TargetFunc: func(ctx EvaluationContext) bool { return true },
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			return DecisionPermit, "permit"
		},
	}

	rule2 := RuleFunc{
		RuleID: "rule_2",
		Desc:   "rule 2",
		TargetFunc: func(ctx EvaluationContext) bool { return true },
		EvaluateFunc: func(ctx EvaluationContext) (Decision, string) {
			return DecisionDeny, "deny"
		},
	}

	engine := NewEngine(DenyOverrides, discardLogger(), rule1)

	var wg sync.WaitGroup
	const numGoroutines = 50
	const iterations = 100

	// Concurrent readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			evalCtx := EvaluationContext{
				Subject:     Subject{ID: "usr_concurrent", TenantID: "t1"},
				Resource:    Resource{ID: "res_concurrent", TenantID: "t1"},
				Action:      Action{Verb: "read"},
				Environment: Environment{RequestTime: time.Now()},
			}
			for j := 0; j < iterations; j++ {
				_, _ = engine.Evaluate(evalCtx)
			}
		}(i)
	}

	// Concurrent reloader / modifier
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < iterations; j++ {
			if j%2 == 0 {
				engine.ReloadRules([]PolicyRule{rule1, rule2})
			} else {
				engine.ReloadRules([]PolicyRule{rule1})
			}
			time.Sleep(100 * time.Microsecond)
		}
	}()

	wg.Wait()
}
