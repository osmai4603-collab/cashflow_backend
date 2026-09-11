package website

import (
	"errors"
	"fmt"
)

var (
	ErrSiteNotFound = errors.New("website site not found")
	ErrPageNotFound = errors.New("website page not found")
	ErrMenuNotFound = errors.New("website menu not found")
)

func WrapErrorf(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
