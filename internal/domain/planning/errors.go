package planning

import (
	"fmt"

	platformerrors "cashflow_backend/internal/platform/errors"
)

func errNotFound(domain string, id int64) error {
	return platformerrors.NotFound(fmt.Sprintf("%s with ID %d not found", domain, id))
}

func errInvalid(message string) error {
	return platformerrors.Validation(message, nil)
}

func errConflict(message string) error {
	return platformerrors.Conflict(message)
}
