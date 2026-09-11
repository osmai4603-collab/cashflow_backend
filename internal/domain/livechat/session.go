package livechat

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type SessionStatus string

const (
	SessionStatusActive SessionStatus = "active"
	SessionStatusClosed SessionStatus = "closed"
)

// Session represents an ongoing or past chat conversation (mail.channel with channel_type='livechat').
type Session struct {
	ID                int64         `json:"id"`
	ChannelID         int64         `json:"channel_id"`
	OperatorID        *int64        `json:"operator_id,omitempty"`
	VisitorUUID       string        `json:"visitor_uuid"`
	VisitorName       string        `json:"visitor_name"`
	VisitorEmail      string        `json:"visitor_email,omitempty"`
	PartnerID         *int64        `json:"partner_id,omitempty"`
	Status            SessionStatus `json:"status"`
	RatingScore       *int          `json:"rating_score,omitempty"`
	RatingComment     string        `json:"rating_comment,omitempty"`
	ConvertedTicketID *int64        `json:"converted_ticket_id,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	ClosedAt          *time.Time    `json:"closed_at,omitempty"`
}

// Close marks the session as closed.
func (s *Session) Close() error {
	if s.Status == SessionStatusClosed {
		return ErrSessionClosed
	}
	now := time.Now().UTC()
	s.Status = SessionStatusClosed
	s.ClosedAt = &now
	return nil
}

// AssignOperator assigns an operator to the session.
func (s *Session) AssignOperator(operatorID int64) error {
	if s.Status == SessionStatusClosed {
		return ErrSessionClosed
	}
	s.OperatorID = &operatorID
	return nil
}

// AddRating sets the rating for the session.
func (s *Session) AddRating(score int, comment string) error {
	if score < 1 || score > 5 {
		return platformerrors.Validation("rating score must be between 1 and 5", nil)
	}
	s.RatingScore = &score
	s.RatingComment = comment
	return nil
}

// ThreadModel satisfies activity.Threadable.
func (s *Session) ThreadModel() string { return "livechat.session" }

// ThreadID satisfies activity.Threadable.
func (s *Session) ThreadID() int64 { return s.ID }

// ThreadCompanyID satisfies activity.Threadable.
func (s *Session) ThreadCompanyID() int64 { return 1 } // To be improved if channel has company context
