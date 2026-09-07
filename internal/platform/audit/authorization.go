package audit

import "time"

// AuthorizationDecision records a model-level access decision without sensitive payloads.
type AuthorizationDecision struct {
	Event     string
	UserID    int64
	CompanyID int64
	Model     string
	Action    string
	Allowed   bool
	Reason    string
	CreatedAt time.Time
}

// AuthorizationSink receives authorization decisions and administrative events.
type AuthorizationSink interface {
	RecordAuthorizationDecision(AuthorizationDecision)
}
