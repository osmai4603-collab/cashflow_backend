package helpdesk

import platformerrors "cashflow_backend/internal/platform/errors"

var (
	ErrTicketNotFound    = platformerrors.NotFound("ticket not found")
	ErrTeamNotFound      = platformerrors.NotFound("helpdesk team not found")
	ErrStageNotFound     = platformerrors.NotFound("helpdesk stage not found")
	ErrSLAPolicyNotFound = platformerrors.NotFound("sla policy not found")
	ErrArticleNotFound   = platformerrors.NotFound("knowledge article not found")
)
