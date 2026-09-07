package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	domaingroup "cashflow_backend/internal/domain/group"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
)

// Action is a model-level operation checked by the authorization service.
type Action string

const (
	ActionRead   Action = "read"
	ActionCreate Action = "create"
	ActionWrite  Action = "write"
	ActionUnlink Action = "unlink"
)

// Subject identifies the authenticated principal for an authorization decision.
type Subject struct {
	UserID      int64
	CompanyID   int64
	IsSuperuser bool
}

// Authorizer evaluates model-level and record-level access decisions.
type Authorizer interface {
	Check(ctx context.Context, subject Subject, model string, action Action) error
}

// ACLAuthorizer evaluates the current model-level ACL repository.
type ACLAuthorizer struct {
	permissions domaingroup.PermissionRepository
	sink        audit.AuthorizationSink
}

func NewACLAuthorizer(permissions domaingroup.PermissionRepository) *ACLAuthorizer {
	return NewACLAuthorizerWithSink(permissions, nil)
}

func NewACLAuthorizerWithSink(permissions domaingroup.PermissionRepository, sink audit.AuthorizationSink) *ACLAuthorizer {
	return &ACLAuthorizer{permissions: permissions, sink: sink}
}

func (a *ACLAuthorizer) Check(ctx context.Context, subject Subject, model string, action Action) error {
	if subject.UserID <= 0 {
		return a.finish(subject, model, action, platformerrors.Unauthorized("authenticated user is required"))
	}
	model = strings.TrimSpace(model)
	if model == "" || !isSupportedAction(action) {
		return a.finish(subject, model, action, platformerrors.BadRequest("invalid authorization target"))
	}
	if subject.IsSuperuser {
		return a.finish(subject, model, action, nil)
	}
	if a == nil || a.permissions == nil {
		return a.finish(subject, model, action, platformerrors.Internal("authorization service is not configured"))
	}
	allowed, err := a.permissions.CheckAccess(ctx, subject.UserID, model, string(action))
	if err != nil {
		return a.finish(subject, model, action, err)
	}
	if !allowed {
		return a.finish(subject, model, action, platformerrors.Forbidden("insufficient permissions"))
	}
	return a.finish(subject, model, action, nil)
}

func (a *ACLAuthorizer) finish(subject Subject, model string, action Action, err error) error {
	if a != nil && a.sink != nil {
		event := audit.AuthorizationDecision{
			Event: "authorization.allowed", UserID: subject.UserID, CompanyID: subject.CompanyID,
			Model: model, Action: string(action), Allowed: err == nil, CreatedAt: time.Now().UTC(),
		}
		if err != nil {
			event.Event = "authorization.denied"
			event.Reason = err.Error()
		}
		a.sink.RecordAuthorizationDecision(event)
	}
	return err
}

func isSupportedAction(action Action) bool {
	switch action {
	case ActionRead, ActionCreate, ActionWrite, ActionUnlink:
		return true
	default:
		return false
	}
}

// RequireAccess enforces a model/action decision after authentication middleware.
func RequireAccess(authorizer Authorizer, model string, action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.Error(w, platformerrors.Unauthorized("unauthenticated request"))
				return
			}
			subject := Subject{UserID: claims.UserID, CompanyID: claims.CompanyID}
			if err := authorizer.Check(r.Context(), subject, model, action); err != nil {
				response.Error(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
