package auth

import (
	"context"
	"net/http"
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
)

type contextKey string

const claimsContextKey contextKey = "auth_user_claims"

// WithClaims injects UserClaims into the context.
func WithClaims(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext retrieves UserClaims from context if present.
func ClaimsFromContext(ctx context.Context) *UserClaims {
	if ctx == nil {
		return nil
	}
	if c, ok := ctx.Value(claimsContextKey).(*UserClaims); ok {
		return c
	}
	return nil
}

// Middleware creates an HTTP authentication middleware that verifies Bearer tokens.
// On success, it automatically injects user ID and company ID into audit context.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractBearerToken(r)
			if tokenString == "" {
				response.Error(w, r, platformerrors.Unauthorized("missing authorization token"))
				return
			}

			claims, err := ValidateToken(tokenString, secret)
			if err != nil {
				response.Error(w, r, platformerrors.Unauthorized(err.Error()))
				return
			}

			ctx := r.Context()
			ctx = WithClaims(ctx, claims)
			ctx = audit.WithUserID(ctx, claims.UserID)
			ctx = audit.WithCompanyID(ctx, claims.CompanyID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware extracts and injects auth claims if a valid token is present,
// but does not reject unauthenticated requests.
func OptionalMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractBearerToken(r)
			if tokenString != "" {
				if claims, err := ValidateToken(tokenString, secret); err == nil {
					ctx := r.Context()
					ctx = WithClaims(ctx, claims)
					ctx = audit.WithUserID(ctx, claims.UserID)
					ctx = audit.WithCompanyID(ctx, claims.CompanyID)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole ensures that the authenticated user possesses at least one of the required roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.Error(w, r, platformerrors.Unauthorized("unauthenticated request"))
				return
			}

			hasRequiredRole := false
			for _, required := range roles {
				if claims.HasRole(required) {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				response.Error(w, r, platformerrors.Forbidden("insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return strings.TrimSpace(parts[1])
	}

	return ""
}
