package audit

import "log/slog"

// SlogAuthorizationSink writes authorization decisions to the structured logger.
type SlogAuthorizationSink struct {
	logger *slog.Logger
}

func NewSlogAuthorizationSink(logger *slog.Logger) *SlogAuthorizationSink {
	return &SlogAuthorizationSink{logger: logger}
}

func (s *SlogAuthorizationSink) RecordAuthorizationDecision(decision AuthorizationDecision) {
	if s == nil || s.logger == nil {
		return
	}
	attrs := []any{
		"user_id", decision.UserID,
		"company_id", decision.CompanyID,
		"model", decision.Model,
		"action", decision.Action,
		"allowed", decision.Allowed,
		"reason", decision.Reason,
	}
	if decision.Allowed {
		s.logger.Info(decision.Event, attrs...)
		return
	}
	s.logger.Warn(decision.Event, attrs...)
}
