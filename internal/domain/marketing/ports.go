package marketing

import (
	"context"
)

type CampaignRepository interface {
	Create(ctx context.Context, campaign *Campaign) error
	GetByID(ctx context.Context, id int64) (*Campaign, error)
	Update(ctx context.Context, campaign *Campaign) error
	List(ctx context.Context, companyID int64) ([]*Campaign, error)
	Delete(ctx context.Context, id int64) error
}

type MailingListRepository interface {
	Create(ctx context.Context, list *MailingList) error
	GetByID(ctx context.Context, id int64) (*MailingList, error)
	Update(ctx context.Context, list *MailingList) error
	List(ctx context.Context, companyID int64) ([]*MailingList, error)
	Delete(ctx context.Context, id int64) error
	AddContact(ctx context.Context, listID, contactID int64) error
	RemoveContact(ctx context.Context, listID, contactID int64) error
}

type ContactRepository interface {
	Create(ctx context.Context, contact *Contact) error
	GetByID(ctx context.Context, id int64) (*Contact, error)
	GetByEmail(ctx context.Context, companyID int64, email string) (*Contact, error)
	Update(ctx context.Context, contact *Contact) error
	List(ctx context.Context, companyID int64) ([]*Contact, error)
	Delete(ctx context.Context, id int64) error
	SetOptOut(ctx context.Context, id int64, optOut bool) error
	SetBlacklist(ctx context.Context, id int64, blacklist bool) error
}

type MassMailingRepository interface {
	Create(ctx context.Context, mailing *MassMailing) error
	GetByID(ctx context.Context, id int64) (*MassMailing, error)
	Update(ctx context.Context, mailing *MassMailing) error
	List(ctx context.Context, companyID int64) ([]*MassMailing, error)
	Delete(ctx context.Context, id int64) error
}

type TrackingRepository interface {
	CreateTrace(ctx context.Context, trace *MailingTrace) error
	GetTraceByCode(ctx context.Context, code string) (*MailingTrace, error)
	UpdateTrace(ctx context.Context, trace *MailingTrace) error
	LogClick(ctx context.Context, event *ClickEvent) error
}

type AutomationRepository interface {
	Create(ctx context.Context, automation *Automation) error
	GetByID(ctx context.Context, id int64) (*Automation, error)
	Update(ctx context.Context, automation *Automation) error
	List(ctx context.Context, companyID int64) ([]*Automation, error)
	ListActive(ctx context.Context, companyID int64) ([]*Automation, error)
	Delete(ctx context.Context, id int64) error

	CreateActivity(ctx context.Context, activity *AutomationActivity) error
	GetActivities(ctx context.Context, automationID int64) ([]*AutomationActivity, error)
}

type EmailProvider interface {
	Send(ctx context.Context, to, subject, body string) error
}

type SMSProvider interface {
	Send(ctx context.Context, to, message string) error
}
