package livechat

import platformerrors "cashflow_backend/internal/platform/errors"

var (
	ErrChannelNotFound = platformerrors.NotFound("livechat channel not found")
	ErrSessionNotFound = platformerrors.NotFound("livechat session not found")
	ErrSessionClosed   = platformerrors.Conflict("livechat session is already closed")
	ErrNotOperator     = platformerrors.PermissionDenied("user is not an operator for this channel")
)
