package pos

import (
	"fmt"

	platformerrors "cashflow_backend/internal/platform/errors"
)

func invalidState(entity string, state any) error {
	return platformerrors.Conflict(fmt.Sprintf("cannot change %s from state '%s'", entity, state))
}
