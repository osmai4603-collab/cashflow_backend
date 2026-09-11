package marketingstorage

import (
	"cashflow_backend/internal/domain/marketing"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// CampaignRepository
func (r *PostgresRepo) Create(ctx context.Context, campaign *marketing.Campaign) error { return nil }
func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*marketing.Campaign, error) {
	return nil, nil
}
func (r *PostgresRepo) Update(ctx context.Context, campaign *marketing.Campaign) error { return nil }
func (r *PostgresRepo) List(ctx context.Context, companyID int64) ([]*marketing.Campaign, error) {
	return nil, nil
}
func (r *PostgresRepo) Delete(ctx context.Context, id int64) error { return nil }

// MailingListRepository
func (r *PostgresRepo) CreateMailingList(ctx context.Context, list *marketing.MailingList) error {
	return nil
}
func (r *PostgresRepo) GetMailingListByID(ctx context.Context, id int64) (*marketing.MailingList, error) {
	return nil, nil
}
func (r *PostgresRepo) UpdateMailingList(ctx context.Context, list *marketing.MailingList) error {
	return nil
}
func (r *PostgresRepo) ListMailingLists(ctx context.Context, companyID int64) ([]*marketing.MailingList, error) {
	return nil, nil
}
func (r *PostgresRepo) DeleteMailingList(ctx context.Context, id int64) error            { return nil }
func (r *PostgresRepo) AddContact(ctx context.Context, listID, contactID int64) error    { return nil }
func (r *PostgresRepo) RemoveContact(ctx context.Context, listID, contactID int64) error { return nil }

// ContactRepository
func (r *PostgresRepo) CreateContact(ctx context.Context, contact *marketing.Contact) error {
	return nil
}
func (r *PostgresRepo) GetContactByID(ctx context.Context, id int64) (*marketing.Contact, error) {
	return nil, nil
}
func (r *PostgresRepo) GetByEmail(ctx context.Context, companyID int64, email string) (*marketing.Contact, error) {
	return nil, nil
}
func (r *PostgresRepo) UpdateContact(ctx context.Context, contact *marketing.Contact) error {
	return nil
}
func (r *PostgresRepo) ListContacts(ctx context.Context, companyID int64) ([]*marketing.Contact, error) {
	return nil, nil
}
func (r *PostgresRepo) DeleteContact(ctx context.Context, id int64) error                { return nil }
func (r *PostgresRepo) SetOptOut(ctx context.Context, id int64, optOut bool) error       { return nil }
func (r *PostgresRepo) SetBlacklist(ctx context.Context, id int64, blacklist bool) error { return nil }

// MassMailingRepository
func (r *PostgresRepo) CreateMassMailing(ctx context.Context, mailing *marketing.MassMailing) error {
	return nil
}
func (r *PostgresRepo) GetMassMailingByID(ctx context.Context, id int64) (*marketing.MassMailing, error) {
	return nil, nil
}
func (r *PostgresRepo) UpdateMassMailing(ctx context.Context, mailing *marketing.MassMailing) error {
	return nil
}
func (r *PostgresRepo) ListMassMailings(ctx context.Context, companyID int64) ([]*marketing.MassMailing, error) {
	return nil, nil
}
func (r *PostgresRepo) DeleteMassMailing(ctx context.Context, id int64) error { return nil }

// TrackingRepository
func (r *PostgresRepo) CreateTrace(ctx context.Context, trace *marketing.MailingTrace) error {
	return nil
}
func (r *PostgresRepo) GetTraceByCode(ctx context.Context, code string) (*marketing.MailingTrace, error) {
	return nil, nil
}
func (r *PostgresRepo) UpdateTrace(ctx context.Context, trace *marketing.MailingTrace) error {
	return nil
}
func (r *PostgresRepo) LogClick(ctx context.Context, event *marketing.ClickEvent) error { return nil }

func (r *PostgresRepo) Add(ctx context.Context, entry *marketing.BlacklistEntry) error { return nil }
func (r *PostgresRepo) ListBlacklist(ctx context.Context, companyID int64) ([]*marketing.BlacklistEntry, error) {
	return nil, nil
}
func (r *PostgresRepo) Remove(ctx context.Context, companyID int64, value string) error { return nil }
func (r *PostgresRepo) IsBlacklisted(ctx context.Context, companyID int64, value string) (bool, error) {
	return false, nil
}

// AutomationRepository
func (r *PostgresRepo) CreateAutomation(ctx context.Context, automation *marketing.Automation) error {
	return nil
}
func (r *PostgresRepo) GetAutomationByID(ctx context.Context, id int64) (*marketing.Automation, error) {
	return nil, nil
}
func (r *PostgresRepo) UpdateAutomation(ctx context.Context, automation *marketing.Automation) error {
	return nil
}
func (r *PostgresRepo) ListAutomations(ctx context.Context, companyID int64) ([]*marketing.Automation, error) {
	return nil, nil
}
func (r *PostgresRepo) ListActive(ctx context.Context, companyID int64) ([]*marketing.Automation, error) {
	return nil, nil
}
func (r *PostgresRepo) DeleteAutomation(ctx context.Context, id int64) error { return nil }

func (r *PostgresRepo) CreateActivity(ctx context.Context, activity *marketing.AutomationActivity) error {
	return nil
}
func (r *PostgresRepo) GetActivities(ctx context.Context, automationID int64) ([]*marketing.AutomationActivity, error) {
	return nil, nil
}
