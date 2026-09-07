package auth

import (
	"context"
	"errors"
	"sync"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type authorizationCacheKey struct {
	userID    int64
	companyID int64
	model     string
	action    Action
}

type authorizationCacheEntry struct {
	allowed bool
	version uint64
}

// CachedAuthorizer caches successful and denied model-level decisions.
type CachedAuthorizer struct {
	delegate Authorizer
	mu       sync.RWMutex
	version  uint64
	entries  map[authorizationCacheKey]authorizationCacheEntry
}

func NewCachedAuthorizer(delegate Authorizer) *CachedAuthorizer {
	return &CachedAuthorizer{delegate: delegate, entries: make(map[authorizationCacheKey]authorizationCacheEntry)}
}

func (a *CachedAuthorizer) Check(ctx context.Context, subject Subject, model string, action Action) error {
	key := authorizationCacheKey{userID: subject.UserID, companyID: subject.CompanyID, model: model, action: action}
	a.mu.RLock()
	entry, ok := a.entries[key]
	version := a.version
	a.mu.RUnlock()
	if ok && entry.version == version {
		if entry.allowed {
			return nil
		}
		return platformerrors.Forbidden("insufficient permissions")
	}

	err := a.delegate.Check(ctx, subject, model, action)
	if err == nil || isForbidden(err) {
		a.mu.Lock()
		a.entries[key] = authorizationCacheEntry{allowed: err == nil, version: a.version}
		a.mu.Unlock()
	}
	return err
}

// Invalidate discards every cached decision after an ACL, group, or rule change.
func (a *CachedAuthorizer) Invalidate() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.version++
	a.entries = make(map[authorizationCacheKey]authorizationCacheEntry)
}

func isForbidden(err error) bool {
	var appErr *platformerrors.AppError
	return errors.As(err, &appErr) && appErr.Code == platformerrors.CodeForbidden
}
