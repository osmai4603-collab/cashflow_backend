package examples

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Authorizer defines the interface for evaluating permissions against assigned roles.
type Authorizer interface {
	HasPermission(roles []string, requiredPermission string) bool
}

// MemoryAuthorizer is an in-memory role-to-permission mapping engine.
type MemoryAuthorizer struct {
	mu          sync.RWMutex
	rolePerms   map[string][]string
}

// NewMemoryAuthorizer initializes a thread-safe authorizer.
func NewMemoryAuthorizer() *MemoryAuthorizer {
	return &MemoryAuthorizer{
		rolePerms: make(map[string][]string),
	}
}

// DefineRole associates a role with a list of permission patterns.
func (a *MemoryAuthorizer) DefineRole(role string, permissions ...string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rolePerms[role] = append(a.rolePerms[role], permissions...)
}

// HasPermission checks if any of the given roles possesses the required permission.
func (a *MemoryAuthorizer) HasPermission(roles []string, requiredPermission string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, role := range roles {
		perms, ok := a.rolePerms[role]
		if !ok {
			continue
		}
		for _, perm := range perms {
			if matchPermission(perm, requiredPermission) {
				return true
			}
		}
	}
	return false
}

// matchPermission evaluates exact matches and wildcards (e.g., "invoices:*" or "*").
func matchPermission(pattern, target string) bool {
	if pattern == "*" || pattern == target {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(target, prefix+":")
	}
	return false
}

// RequirePermission wraps an HTTP handler and ensures the caller has the required permission.
func RequirePermission(auth Authorizer, requiredPerm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, err := FromContext(r.Context())
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"missing tenant context"}`, http.StatusUnauthorized)
				return
			}

			if !auth.HasPermission(tenant.Roles, requiredPerm) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintf(w, `{"error":"forbidden","message":"insufficient permissions","required":%q}`, requiredPerm)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
