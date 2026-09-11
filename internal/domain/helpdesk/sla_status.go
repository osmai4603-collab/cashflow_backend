package helpdesk

import "time"

// SLAStatus represents the current SLA standing of a ticket.
type SLAStatus struct {
	TicketID           int64      `json:"ticket_id"`
	PolicyID           int64      `json:"policy_id"`
	DeadlineFirstResp  *time.Time `json:"deadline_first_resp,omitempty"`
	DeadlineResolution *time.Time `json:"deadline_resolution,omitempty"`
	ReachedFirstResp   bool       `json:"reached_first_resp"`
	ReachedResolution  bool       `json:"reached_resolution"`
	IsBreached         bool       `json:"is_breached"`
}

// CalculateBreach checks if any deadline has passed.
func (s *SLAStatus) CalculateBreach() {
	now := time.Now().UTC()
	s.IsBreached = false
	if !s.ReachedFirstResp && s.DeadlineFirstResp != nil && now.After(*s.DeadlineFirstResp) {
		s.IsBreached = true
	}
	if !s.ReachedResolution && s.DeadlineResolution != nil && now.After(*s.DeadlineResolution) {
		s.IsBreached = true
	}
}
