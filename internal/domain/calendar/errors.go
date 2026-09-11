package calendar

import (
	"fmt"

	platformerrors "cashflow_backend/internal/platform/errors"
)

func invalidCalendar(message string) error {
	return platformerrors.Validation(message, nil)
}

func conflictCalendar(message string) error {
	return platformerrors.Conflict(fmt.Sprintf("calendar conflict: %s", message))
}
