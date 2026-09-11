package portal

import (
	"errors"
	"fmt"
)

var (
	ErrUserNotFound      = errors.New("portal user not found")
	ErrInvalidPassword   = errors.New("invalid portal credentials")
	ErrUserInactive      = errors.New("portal account is inactive")
	ErrAccessDenied      = errors.New("access denied to requested document")
	ErrInviteExpired     = errors.New("portal invitation has expired")
)

func WrapErrorf(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
