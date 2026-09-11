package marketing

import "errors"

var (
	ErrCampaignNotFound    = errors.New("marketing campaign not found")
	ErrMailingListNotFound = errors.New("mailing list not found")
	ErrContactNotFound     = errors.New("mailing contact not found")
	ErrAutomationNotFound  = errors.New("marketing automation not found")
	ErrActivityNotFound    = errors.New("automation activity not found")
	ErrInvalidState        = errors.New("invalid campaign/mailing state")
	ErrContactOptedOut     = errors.New("contact has opted out")
	ErrContactBlacklisted  = errors.New("contact is blacklisted")
)
