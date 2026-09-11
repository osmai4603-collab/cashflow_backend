package marketingusecase

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"cashflow_backend/internal/domain/marketing"
	platformerrors "cashflow_backend/internal/platform/errors"
)

const defaultMarketingCompanyID int64 = 1

type UseCase struct {
	campaignRepo  marketing.CampaignRepository
	mailingRepo   marketing.MailingListRepository
	contactRepo   marketing.ContactRepository
	massRepo      marketing.MassMailingRepository
	trackRepo     marketing.TrackingRepository
	autoRepo      marketing.AutomationRepository
	blacklistRepo marketing.BlacklistRepository
}

func (u *UseCase) ListCampaigns(ctx context.Context) ([]marketing.Campaign, error) {
	items, err := u.campaignRepo.List(ctx, defaultMarketingCompanyID)
	if err != nil {
		return nil, err
	}
	result := make([]marketing.Campaign, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (u *UseCase) CreateCampaign(ctx context.Context, c *marketing.Campaign) (*marketing.Campaign, error) {
	if c == nil {
		return nil, platformerrors.Validation("campaign is required", nil)
	}
	if c.CompanyID == 0 {
		c.CompanyID = defaultMarketingCompanyID
	}
	if err := c.Validate(); err != nil {
		return nil, platformerrors.Validation(err.Error(), nil)
	}
	if c.UTMSource == "" {
		c.UTMSource = "marketing"
	}
	if c.UTMMedium == "" {
		c.UTMMedium = "email"
	}
	if c.UTMCampaign == "" {
		c.UTMCampaign = strings.ToLower(strings.ReplaceAll(c.Name, " ", "-"))
	}
	if c.State == "" {
		c.State = marketing.CampaignStateDraft
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if err := u.campaignRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *UseCase) ListMailingLists(ctx context.Context) ([]marketing.MailingList, error) {
	items, err := u.mailingRepo.ListMailingLists(ctx, defaultMarketingCompanyID)
	if err != nil {
		return nil, err
	}
	result := make([]marketing.MailingList, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (u *UseCase) CreateMailingList(ctx context.Context, l *marketing.MailingList) (*marketing.MailingList, error) {
	if l == nil || strings.TrimSpace(l.Name) == "" {
		return nil, platformerrors.Validation("mailing list name is required", nil)
	}
	if l.CompanyID == 0 {
		l.CompanyID = defaultMarketingCompanyID
	}
	l.Name = strings.TrimSpace(l.Name)
	l.CreatedAt = time.Now().UTC()
	if err := u.mailingRepo.CreateMailingList(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (u *UseCase) ImportContacts(ctx context.Context, listID int64, data []byte) error {
	if listID <= 0 || len(data) == 0 {
		return platformerrors.Validation("list_id and contact data are required", nil)
	}
	if _, err := u.mailingRepo.GetMailingListByID(ctx, listID); err != nil {
		return err
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	rows, err := reader.ReadAll()
	if err != nil {
		return platformerrors.Validation("invalid contacts CSV", err.Error())
	}
	for index, row := range rows {
		if index == 0 && len(row) > 0 && strings.EqualFold(strings.TrimSpace(row[0]), "email") {
			continue
		}
		if len(row) < 2 {
			return platformerrors.Validation("invalid contact row", fmt.Sprintf("row %d requires email and name", index+1))
		}
		address := strings.TrimSpace(row[0])
		if _, err := mail.ParseAddress(address); err != nil {
			return platformerrors.Validation("invalid contact email", fmt.Sprintf("row %d", index+1))
		}
		contact, findErr := u.contactRepo.GetByEmail(ctx, defaultMarketingCompanyID, address)
		if findErr != nil && findErr != marketing.ErrContactNotFound {
			return findErr
		}
		if contact == nil {
			contact = &marketing.Contact{Email: address, Name: strings.TrimSpace(row[1]), CompanyID: defaultMarketingCompanyID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
			if len(row) > 2 {
				contact.Mobile = strings.TrimSpace(row[2])
			}
			if err := u.contactRepo.CreateContact(ctx, contact); err != nil {
				return err
			}
		}
		if err := u.mailingRepo.AddContact(ctx, listID, contact.ID); err != nil {
			return err
		}
	}
	return nil
}

func (u *UseCase) ListBlacklist(ctx context.Context) ([]marketing.BlacklistEntry, error) {
	if u.blacklistRepo == nil {
		return []marketing.BlacklistEntry{}, nil
	}
	items, err := u.blacklistRepo.ListBlacklist(ctx, defaultMarketingCompanyID)
	if err != nil {
		return nil, err
	}
	result := make([]marketing.BlacklistEntry, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (u *UseCase) AddToBlacklist(ctx context.Context, email string) error {
	address := strings.TrimSpace(strings.ToLower(email))
	if _, err := mail.ParseAddress(address); err != nil {
		return platformerrors.Validation("invalid email address", nil)
	}
	if u.blacklistRepo == nil {
		return platformerrors.NotImplemented("blacklist repository is not configured")
	}
	blocked, err := u.blacklistRepo.IsBlacklisted(ctx, defaultMarketingCompanyID, address)
	if err != nil {
		return err
	}
	if blocked {
		return platformerrors.Conflict("email is already blacklisted")
	}
	return u.blacklistRepo.Add(ctx, &marketing.BlacklistEntry{Value: address, Type: marketing.BlacklistTypeEmail, Reason: "manual", CompanyID: defaultMarketingCompanyID, CreatedAt: time.Now().UTC()})
}

func (u *UseCase) CreateMassMailing(ctx context.Context, m *marketing.MassMailing) (*marketing.MassMailing, error) {
	if m == nil || strings.TrimSpace(m.Subject) == "" || strings.TrimSpace(m.BodyHTML) == "" {
		return nil, platformerrors.Validation("mailing subject and body are required", nil)
	}
	if m.CompanyID == 0 {
		m.CompanyID = defaultMarketingCompanyID
	}
	if m.State == "" {
		m.State = marketing.MassMailingStateDraft
	}
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	if err := u.massRepo.CreateMassMailing(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (u *UseCase) SendTestMailing(ctx context.Context, id int64, email string) error {
	if id <= 0 || strings.TrimSpace(email) == "" {
		return platformerrors.Validation("mailing id and recipient email are required", nil)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return platformerrors.Validation("invalid recipient email", nil)
	}
	mailing, err := u.massRepo.GetMassMailingByID(ctx, id)
	if err != nil {
		return err
	}
	if mailing == nil {
		return platformerrors.NotFound(fmt.Sprintf("mailing %d not found", id))
	}
	return nil
}

func (u *UseCase) ScheduleMailing(ctx context.Context, id int64) error {
	mailing, err := u.massRepo.GetMassMailingByID(ctx, id)
	if err != nil {
		return err
	}
	if mailing == nil {
		return platformerrors.NotFound(fmt.Sprintf("mailing %d not found", id))
	}
	if mailing.State != marketing.MassMailingStateDraft {
		return platformerrors.Conflict("only draft mailings can be scheduled")
	}
	mailing.State = marketing.MassMailingStateInQueue
	mailing.ScheduledDate = timePtr(time.Now().UTC())
	mailing.UpdatedAt = time.Now().UTC()
	return u.massRepo.UpdateMassMailing(ctx, mailing)
}

func (u *UseCase) CancelMailing(ctx context.Context, id int64) error {
	mailing, err := u.massRepo.GetMassMailingByID(ctx, id)
	if err != nil {
		return err
	}
	if mailing == nil {
		return platformerrors.NotFound(fmt.Sprintf("mailing %d not found", id))
	}
	if mailing.State == marketing.MassMailingStateDone {
		return platformerrors.Conflict("completed mailings cannot be cancelled")
	}
	mailing.State = marketing.MassMailingStateCancelled
	mailing.UpdatedAt = time.Now().UTC()
	return u.massRepo.UpdateMassMailing(ctx, mailing)
}

func (u *UseCase) GetMailingStats(ctx context.Context, id int64) (interface{}, error) {
	mailing, err := u.massRepo.GetMassMailingByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if mailing == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("mailing %d not found", id))
	}
	return map[string]int{"sent": 0, "delivered": 0, "opened": 0, "clicked": 0, "bounced": 0}, nil
}

func (u *UseCase) TrackOpen(ctx context.Context, code string) error {
	if strings.TrimSpace(code) == "" {
		return platformerrors.Validation("tracking code is required", nil)
	}
	trace, err := u.trackRepo.GetTraceByCode(ctx, code)
	if err != nil {
		return err
	}
	if trace == nil {
		return platformerrors.NotFound("mailing trace not found")
	}
	if trace.OpenedAt == nil {
		now := time.Now().UTC()
		trace.OpenedAt = &now
		if err := u.trackRepo.UpdateTrace(ctx, trace); err != nil {
			return err
		}
	}
	return nil
}

func (u *UseCase) TrackClick(ctx context.Context, code string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", platformerrors.Validation("tracking code is required", nil)
	}
	trace, err := u.trackRepo.GetTraceByCode(ctx, code)
	if err != nil {
		return "", err
	}
	if trace == nil {
		return "", platformerrors.NotFound("mailing trace not found")
	}
	now := time.Now().UTC()
	trace.ClickedAt = &now
	if err := u.trackRepo.UpdateTrace(ctx, trace); err != nil {
		return "", err
	}
	return "/", nil
}

func (u *UseCase) Unsubscribe(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return platformerrors.Validation("unsubscribe token is required", nil)
	}
	contact, err := u.contactRepo.GetByEmail(ctx, defaultMarketingCompanyID, token)
	if err != nil {
		return err
	}
	if contact == nil {
		return platformerrors.NotFound("marketing contact not found")
	}
	contact.IsOptOut = true
	contact.UpdatedAt = time.Now().UTC()
	return u.contactRepo.UpdateContact(ctx, contact)
}

func (u *UseCase) ListAutomations(ctx context.Context) ([]marketing.Automation, error) {
	if u.autoRepo == nil {
		return []marketing.Automation{}, nil
	}
	items, err := u.autoRepo.ListAutomations(ctx, defaultMarketingCompanyID)
	if err != nil {
		return nil, err
	}
	result := make([]marketing.Automation, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (u *UseCase) CreateAutomation(ctx context.Context, a *marketing.Automation) (*marketing.Automation, error) {
	if a == nil || strings.TrimSpace(a.Name) == "" {
		return nil, platformerrors.Validation("automation name is required", nil)
	}
	if a.CompanyID == 0 {
		a.CompanyID = defaultMarketingCompanyID
	}
	a.Name = strings.TrimSpace(a.Name)
	if a.TriggerType == "" {
		a.TriggerType = marketing.TriggerCustom
	}
	a.Active = true
	a.CreatedAt = time.Now().UTC()
	if err := u.autoRepo.CreateAutomation(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *UseCase) UpdateAutomation(ctx context.Context, id int64, a *marketing.Automation) (*marketing.Automation, error) {
	if a == nil || id <= 0 {
		return nil, platformerrors.Validation("automation and id are required", nil)
	}
	existing, err := u.autoRepo.GetAutomationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("automation %d not found", id))
	}
	a.ID = id
	a.CompanyID = existing.CompanyID
	a.CreatedAt = existing.CreatedAt
	if err := u.autoRepo.UpdateAutomation(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *UseCase) TriggerTestAutomation(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.Validation("automation id is required", nil)
	}
	a, err := u.autoRepo.GetAutomationByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return platformerrors.NotFound(fmt.Sprintf("automation %d not found", id))
	}
	if !a.Active {
		return platformerrors.Conflict("automation is inactive")
	}
	return nil
}

func New(
	campaignRepo marketing.CampaignRepository,
	mailingRepo marketing.MailingListRepository,
	contactRepo marketing.ContactRepository,
	massRepo marketing.MassMailingRepository,
	trackRepo marketing.TrackingRepository,
	autoRepo marketing.AutomationRepository,
) *UseCase {
	var blacklistRepo marketing.BlacklistRepository
	if repo, ok := campaignRepo.(marketing.BlacklistRepository); ok {
		blacklistRepo = repo
	}
	return &UseCase{
		campaignRepo:  campaignRepo,
		mailingRepo:   mailingRepo,
		contactRepo:   contactRepo,
		massRepo:      massRepo,
		trackRepo:     trackRepo,
		autoRepo:      autoRepo,
		blacklistRepo: blacklistRepo,
	}
}

func timePtr(value time.Time) *time.Time { return &value }

func newTrackingCode() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", value), nil
}
