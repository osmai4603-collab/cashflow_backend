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
	CreateMailingList(ctx context.Context, list *MailingList) error
	GetMailingListByID(ctx context.Context, id int64) (*MailingList, error)
	UpdateMailingList(ctx context.Context, list *MailingList) error
	ListMailingLists(ctx context.Context, companyID int64) ([]*MailingList, error)
	DeleteMailingList(ctx context.Context, id int64) error
	AddContact(ctx context.Context, listID, contactID int64) error
	RemoveContact(ctx context.Context, listID, contactID int64) error
}

type ContactRepository interface {
	CreateContact(ctx context.Context, contact *Contact) error
	GetContactByID(ctx context.Context, id int64) (*Contact, error)
	GetByEmail(ctx context.Context, companyID int64, email string) (*Contact, error)
	UpdateContact(ctx context.Context, contact *Contact) error
	ListContacts(ctx context.Context, companyID int64) ([]*Contact, error)
	DeleteContact(ctx context.Context, id int64) error
	SetOptOut(ctx context.Context, id int64, optOut bool) error
	SetBlacklist(ctx context.Context, id int64, blacklist bool) error
}

type MassMailingRepository interface {
	CreateMassMailing(ctx context.Context, mailing *MassMailing) error
	GetMassMailingByID(ctx context.Context, id int64) (*MassMailing, error)
	UpdateMassMailing(ctx context.Context, mailing *MassMailing) error
	ListMassMailings(ctx context.Context, companyID int64) ([]*MassMailing, error)
	DeleteMassMailing(ctx context.Context, id int64) error
}

type TrackingRepository interface {
	CreateTrace(ctx context.Context, trace *MailingTrace) error
	GetTraceByCode(ctx context.Context, code string) (*MailingTrace, error)
	UpdateTrace(ctx context.Context, trace *MailingTrace) error
	LogClick(ctx context.Context, event *ClickEvent) error
}

type AutomationRepository interface {
	CreateAutomation(ctx context.Context, automation *Automation) error
	GetAutomationByID(ctx context.Context, id int64) (*Automation, error)
	UpdateAutomation(ctx context.Context, automation *Automation) error
	ListAutomations(ctx context.Context, companyID int64) ([]*Automation, error)
	ListActive(ctx context.Context, companyID int64) ([]*Automation, error)
	DeleteAutomation(ctx context.Context, id int64) error

	CreateActivity(ctx context.Context, activity *AutomationActivity) error
	GetActivities(ctx context.Context, automationID int64) ([]*AutomationActivity, error)
}

type EmailProvider interface {
	Send(ctx context.Context, to, subject, body string) error
}

type SMSProvider interface {
	Send(ctx context.Context, to, message string) error
}
