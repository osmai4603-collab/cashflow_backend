package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// ============================================================================
// 1. DOMAIN-AGNOSTIC TYPES (Core RBAC)
// ============================================================================

// Permission represents an atomic right to perform an operation on a resource (e.g. "documents:read").
type Permission string

// Role represents a functional collection of permissions (e.g. "editor", "admin").
type Role string

// Subject represents any authenticated principal (user, service account, system actor).
type Subject struct {
	ID       string            `json:"id"`
	TenantID string            `json:"tenant_id,omitempty"` // Supports multi-tenant isolation
	Roles    []Role            `json:"roles"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ============================================================================
// 2. IN-MEMORY HIERARCHICAL RBAC ENGINE (PDP & PAP)
// ============================================================================

// Engine encapsulates the role-permission mapping, inheritance hierarchy, and SSD constraints.
type Engine struct {
	mu           sync.RWMutex
	rolePerms    map[Role]map[Permission]bool
	inheritance  map[Role][]Role // Senior -> []Junior (inherited roles)
	ssdConflicts map[Role][]Role // Mutually exclusive roles (Static Separation of Duties)
}

// NewEngine initializes an empty, thread-safe RBAC engine.
func NewEngine() *Engine {
	return &Engine{
		rolePerms:    make(map[Role]map[Permission]bool),
		inheritance:  make(map[Role][]Role),
		ssdConflicts: make(map[Role][]Role),
	}
}

// DefineRole registers a set of direct permissions for a given role.
func (e *Engine) DefineRole(role Role, perms ...Permission) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.rolePerms[role] == nil {
		e.rolePerms[role] = make(map[Permission]bool)
	}
	for _, p := range perms {
		e.rolePerms[role][p] = true
	}
}

// AddInheritance configures seniorRole to inherit all permissions from juniorRole (Hierarchical RBAC).
func (e *Engine) AddInheritance(senior, junior Role) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.inheritance[senior] = append(e.inheritance[senior], junior)
}

// RegisterSSDConflict establishes a mutually exclusive constraint between two roles (NIST Level 3).
func (e *Engine) RegisterSSDConflict(roleA, roleB Role) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ssdConflicts[roleA] = append(e.ssdConflicts[roleA], roleB)
	e.ssdConflicts[roleB] = append(e.ssdConflicts[roleB], roleA)
}

// ValidateSSD verifies that a proposed set of roles does not violate mutual exclusion constraints.
func (e *Engine) ValidateSSD(roles []Role) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	assigned := make(map[Role]bool, len(roles))
	for _, r := range roles {
		assigned[r] = true
	}

	for _, r := range roles {
		for _, conflicting := range e.ssdConflicts[r] {
			if assigned[conflicting] {
				return fmt.Errorf("SSD constraint violation: cannot assign both %q and %q to the same subject", r, conflicting)
			}
		}
	}
	return nil
}

// resolveEffectivePermissionsLocked walks the inheritance DAG and compiles all granted permissions.
// Caller MUST hold at least e.mu.RLock().
func (e *Engine) resolveEffectivePermissionsLocked(roles []Role) map[Permission]bool {
	effective := make(map[Permission]bool)
	visited := make(map[Role]bool)

	var walk func(r Role)
	walk = func(r Role) {
		if visited[r] {
			return
		}
		visited[r] = true

		for p := range e.rolePerms[r] {
			effective[p] = true
		}
		for _, inherited := range e.inheritance[r] {
			walk(inherited)
		}
	}

	for _, r := range roles {
		walk(r)
	}
	return effective
}

// ResolveEffectivePermissions walks the inheritance DAG and compiles all granted permissions.
func (e *Engine) ResolveEffectivePermissions(roles []Role) map[Permission]bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.resolveEffectivePermissionsLocked(roles)
}

// hasPermissionLocked evaluates whether roles satisfy required permission while holding e.mu.RLock().
func (e *Engine) hasPermissionLocked(roles []Role, required Permission) bool {
	if len(roles) == 0 {
		return false
	}
	perms := e.resolveEffectivePermissionsLocked(roles)
	return perms[required]
}

// HasPermission executes a fail-closed evaluation for a required permission.
func (e *Engine) HasPermission(roles []Role, required Permission) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hasPermissionLocked(roles, required)
}

// WhoCanPerform performs a reverse query (Symmetric RBAC) returning all roles granting targetPerm.
func (e *Engine) WhoCanPerform(targetPerm Permission) []Role {
	e.mu.RLock()
	defer e.mu.RUnlock()

	allRoles := make(map[Role]bool)
	for r := range e.rolePerms {
		allRoles[r] = true
	}
	for senior, juniors := range e.inheritance {
		allRoles[senior] = true
		for _, j := range juniors {
			allRoles[j] = true
		}
	}

	var capable []Role
	for r := range allRoles {
		if e.hasPermissionLocked([]Role{r}, targetPerm) {
			capable = append(capable, r)
		}
	}
	return capable
}

// ============================================================================
// 3. TYPE-SAFE CONTEXT PROPAGATION
// ============================================================================

type contextKey struct{}

var subjectContextKey = contextKey{}

// WithSubject injects the authenticated subject into the Go context.
func WithSubject(ctx context.Context, sub Subject) context.Context {
	return context.WithValue(ctx, subjectContextKey, sub)
}

// SubjectFromContext extracts the subject from the Go context.
func SubjectFromContext(ctx context.Context) (Subject, bool) {
	sub, ok := ctx.Value(subjectContextKey).(Subject)
	return sub, ok
}

// ============================================================================
// 4. RFC 7807 PROBLEM DETAILS & PEP ENFORCEMENT
// ============================================================================

// ProblemDetails formats machine-readable HTTP error envelopes.
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

func WriteProblemDetails(w http.ResponseWriter, status int, title, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Type:     "https://golang.org/errors/access-denied",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	})
}

// RequirePermission constructs an idiomatic HTTP PEP guard middleware.
func RequirePermission(engine *Engine, logger *slog.Logger, required Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			subject, ok := SubjectFromContext(r.Context())
			if !ok {
				logger.Warn("Unauthenticated request reached PEP guard", "path", r.URL.Path)
				WriteProblemDetails(w, http.StatusUnauthorized, "Unauthorized", "Authentication required", r.URL.Path)
				return
			}

			allowed := engine.HasPermission(subject.Roles, required)
			latency := time.Since(start)

			if !allowed {
				// Audit log the rejection
				logger.Warn("RBAC access denied",
					"subject_id", subject.ID,
					"tenant_id", subject.TenantID,
					"roles", subject.Roles,
					"required_permission", required,
					"path", r.URL.Path,
					"latency_us", latency.Microseconds(),
				)
				// Anti-leakage: generic message
				WriteProblemDetails(w, http.StatusForbidden, "Forbidden", "You do not have permission to access this resource", r.URL.Path)
				return
			}

			// Audit log the permit
			logger.Info("RBAC access granted",
				"subject_id", subject.ID,
				"permission", required,
				"latency_us", latency.Microseconds(),
			)

			next.ServeHTTP(w, r)
		})
	}
}

// ============================================================================
// 5. RUNNABLE REFERENCE DEMONSTRATION
// ============================================================================

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// 1. Initialize engine and model roles
	engine := NewEngine()

	// Base permissions
	const (
		PermRead   Permission = "data:read"
		PermWrite  Permission = "data:write"
		PermDelete Permission = "data:delete"
	)

	// Roles
	const (
		RoleReader Role = "reader"
		RoleWriter Role = "writer"
		RoleAdmin  Role = "admin"
	)

	engine.DefineRole(RoleReader, PermRead)
	engine.DefineRole(RoleWriter, PermWrite)
	engine.DefineRole(RoleAdmin, PermDelete)

	// Hierarchy: Admin -> Writer -> Reader
	engine.AddInheritance(RoleWriter, RoleReader)
	engine.AddInheritance(RoleAdmin, RoleWriter)

	// SSD Conflict: Auditor cannot be an Operator
	const (
		RoleAuditor  Role = "auditor"
		RoleOperator Role = "operator"
	)
	engine.RegisterSSDConflict(RoleAuditor, RoleOperator)

	// Verify SSD validation
	err := engine.ValidateSSD([]Role{RoleAuditor, RoleOperator})
	if err != nil {
		logger.Info("SSD validation successfully caught conflicting roles", "error", err)
	}

	// 2. Setup Router
	mux := http.NewServeMux()

	// Mock AuthN Middleware
	authn := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In production, extract subject from validated JWT claims
			caller := Subject{
				ID:       "sub_dev_100",
				TenantID: "tenant_alpha",
				Roles:    []Role{RoleWriter}, // Has Writer + inherits Reader
			}
			next.ServeHTTP(w, r.WithContext(WithSubject(r.Context(), caller)))
		})
	}

	// Endpoint 1: Requires data:read (Writer inherits Reader -> ALLOWED)
	readHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","message":"Read permitted"}`))
	})
	mux.Handle("GET /data", authn(RequirePermission(engine, logger, PermRead)(readHandler)))

	// Endpoint 2: Requires data:delete (Writer does NOT inherit Admin -> DENIED 403)
	deleteHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","message":"Delete permitted"}`))
	})
	mux.Handle("DELETE /data", authn(RequirePermission(engine, logger, PermDelete)(deleteHandler)))

	logger.Info("Generic RBAC Lifecycle Service initialized successfully")
	_ = errors.New("ready")
}
