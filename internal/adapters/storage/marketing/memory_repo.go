package marketingstorage

import (
	"cashflow_backend/internal/domain/marketing"
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryRepo struct {
	mu           sync.RWMutex
	campaigns    map[int64]*marketing.Campaign
	mailingLists map[int64]*marketing.MailingList
	contacts     map[int64]*marketing.Contact
	massMailings map[int64]*marketing.MassMailing
	traces       map[string]*marketing.MailingTrace
	automations  map[int64]*marketing.Automation
	blacklist    map[int64]*marketing.BlacklistEntry
	listContacts map[int64]map[int64]struct{}
	nextID       int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		campaigns:    make(map[int64]*marketing.Campaign),
		mailingLists: make(map[int64]*marketing.MailingList),
		contacts:     make(map[int64]*marketing.Contact),
		massMailings: make(map[int64]*marketing.MassMailing),
		traces:       make(map[string]*marketing.MailingTrace),
		automations:  make(map[int64]*marketing.Automation),
		blacklist:    make(map[int64]*marketing.BlacklistEntry),
		listContacts: make(map[int64]map[int64]struct{}),
	}
}

func (r *MemoryRepo) allocateID() int64 {
	r.nextID++
	return r.nextID
}

// CampaignRepository
func (r *MemoryRepo) Create(ctx context.Context, campaign *marketing.Campaign) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if campaign.ID == 0 {
		campaign.ID = r.allocateID()
	}
	r.campaigns[campaign.ID] = campaign
	return nil
}
func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*marketing.Campaign, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.campaigns[id], nil
}
func (r *MemoryRepo) Update(ctx context.Context, campaign *marketing.Campaign) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.campaigns[campaign.ID] = campaign
	return nil
}
func (r *MemoryRepo) List(ctx context.Context, companyID int64) ([]*marketing.Campaign, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*marketing.Campaign, 0)
	for _, campaign := range r.campaigns {
		if companyID == 0 || campaign.CompanyID == companyID {
			copy := *campaign
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.campaigns, id)
	return nil
}

// Implement other methods as stubs...
func (r *MemoryRepo) CreateMailingList(ctx context.Context, list *marketing.MailingList) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if list.ID == 0 {
		list.ID = r.allocateID()
	}
	r.mailingLists[list.ID] = list
	return nil
}
func (r *MemoryRepo) GetMailingListByID(ctx context.Context, id int64) (*marketing.MailingList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := r.mailingLists[id]
	if list == nil {
		return nil, marketing.ErrMailingListNotFound
	}
	copy := *list
	return &copy, nil
}
func (r *MemoryRepo) UpdateMailingList(ctx context.Context, list *marketing.MailingList) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.mailingLists[list.ID]; !ok {
		return marketing.ErrMailingListNotFound
	}
	r.mailingLists[list.ID] = list
	return nil
}
func (r *MemoryRepo) ListMailingLists(ctx context.Context, companyID int64) ([]*marketing.MailingList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*marketing.MailingList, 0)
	for _, list := range r.mailingLists {
		if companyID == 0 || list.CompanyID == companyID {
			copy := *list
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) DeleteMailingList(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.mailingLists[id]; !ok {
		return marketing.ErrMailingListNotFound
	}
	delete(r.mailingLists, id)
	delete(r.listContacts, id)
	return nil
}
func (r *MemoryRepo) AddContact(ctx context.Context, listID, contactID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.mailingLists[listID]; !ok {
		return marketing.ErrMailingListNotFound
	}
	if _, ok := r.contacts[contactID]; !ok {
		return marketing.ErrContactNotFound
	}
	if r.listContacts[listID] == nil {
		r.listContacts[listID] = make(map[int64]struct{})
	}
	r.listContacts[listID][contactID] = struct{}{}
	return nil
}
func (r *MemoryRepo) RemoveContact(ctx context.Context, listID, contactID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if contacts := r.listContacts[listID]; contacts != nil {
		delete(contacts, contactID)
	}
	return nil
}
func (r *MemoryRepo) CreateContact(ctx context.Context, contact *marketing.Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if contact.ID == 0 {
		contact.ID = r.allocateID()
	}
	r.contacts[contact.ID] = contact
	return nil
}
func (r *MemoryRepo) GetContactByID(ctx context.Context, id int64) (*marketing.Contact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	contact := r.contacts[id]
	if contact == nil {
		return nil, marketing.ErrContactNotFound
	}
	copy := *contact
	return &copy, nil
}
func (r *MemoryRepo) GetByEmail(ctx context.Context, companyID int64, email string) (*marketing.Contact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, contact := range r.contacts {
		if contact.CompanyID == companyID && strings.EqualFold(contact.Email, email) {
			copy := *contact
			return &copy, nil
		}
	}
	return nil, marketing.ErrContactNotFound
}
func (r *MemoryRepo) UpdateContact(ctx context.Context, contact *marketing.Contact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.contacts[contact.ID]; !ok {
		return marketing.ErrContactNotFound
	}
	r.contacts[contact.ID] = contact
	return nil
}
func (r *MemoryRepo) ListContacts(ctx context.Context, companyID int64) ([]*marketing.Contact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*marketing.Contact, 0)
	for _, contact := range r.contacts {
		if companyID == 0 || contact.CompanyID == companyID {
			copy := *contact
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) DeleteContact(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.contacts[id]; !ok {
		return marketing.ErrContactNotFound
	}
	delete(r.contacts, id)
	for _, contacts := range r.listContacts {
		delete(contacts, id)
	}
	return nil
}
func (r *MemoryRepo) SetOptOut(ctx context.Context, id int64, optOut bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	contact := r.contacts[id]
	if contact == nil {
		return marketing.ErrContactNotFound
	}
	contact.IsOptOut = optOut
	return nil
}
func (r *MemoryRepo) SetBlacklist(ctx context.Context, id int64, blacklist bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	contact := r.contacts[id]
	if contact == nil {
		return marketing.ErrContactNotFound
	}
	contact.IsBlacklist = blacklist
	return nil
}

func (r *MemoryRepo) Add(ctx context.Context, entry *marketing.BlacklistEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry.ID == 0 {
		entry.ID = r.allocateID()
	}
	r.blacklist[entry.ID] = entry
	return nil
}
func (r *MemoryRepo) ListBlacklist(ctx context.Context, companyID int64) ([]*marketing.BlacklistEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*marketing.BlacklistEntry, 0)
	for _, entry := range r.blacklist {
		if companyID == 0 || entry.CompanyID == companyID {
			copy := *entry
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) Remove(ctx context.Context, companyID int64, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, entry := range r.blacklist {
		if entry.CompanyID == companyID && strings.EqualFold(entry.Value, value) {
			delete(r.blacklist, id)
		}
	}
	return nil
}
func (r *MemoryRepo) IsBlacklisted(ctx context.Context, companyID int64, value string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entry := range r.blacklist {
		if entry.CompanyID == companyID && strings.EqualFold(entry.Value, value) {
			return true, nil
		}
	}
	return false, nil
}
func (r *MemoryRepo) CreateMassMailing(ctx context.Context, mailing *marketing.MassMailing) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mailing.ID == 0 {
		mailing.ID = r.allocateID()
	}
	r.massMailings[mailing.ID] = mailing
	return nil
}
func (r *MemoryRepo) GetMassMailingByID(ctx context.Context, id int64) (*marketing.MassMailing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mailing := r.massMailings[id]
	if mailing == nil {
		return nil, nil
	}
	copy := *mailing
	return &copy, nil
}
func (r *MemoryRepo) UpdateMassMailing(ctx context.Context, mailing *marketing.MassMailing) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.massMailings[mailing.ID]; !ok {
		return marketing.ErrMailingListNotFound
	}
	r.massMailings[mailing.ID] = mailing
	return nil
}
func (r *MemoryRepo) ListMassMailings(ctx context.Context, companyID int64) ([]*marketing.MassMailing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*marketing.MassMailing, 0)
	for _, mailing := range r.massMailings {
		if companyID == 0 || mailing.CompanyID == companyID {
			copy := *mailing
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) DeleteMassMailing(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.massMailings, id)
	return nil
}
func (r *MemoryRepo) CreateTrace(ctx context.Context, trace *marketing.MailingTrace) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if trace.ID == 0 {
		trace.ID = r.allocateID()
	}
	r.traces[trace.TrackingCode] = trace
	return nil
}
func (r *MemoryRepo) GetTraceByCode(ctx context.Context, code string) (*marketing.MailingTrace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	trace := r.traces[code]
	if trace == nil {
		return nil, nil
	}
	copy := *trace
	return &copy, nil
}
func (r *MemoryRepo) UpdateTrace(ctx context.Context, trace *marketing.MailingTrace) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.traces[trace.TrackingCode]; !ok {
		return marketing.ErrContactNotFound
	}
	r.traces[trace.TrackingCode] = trace
	return nil
}
func (r *MemoryRepo) LogClick(ctx context.Context, event *marketing.ClickEvent) error { return nil }
func (r *MemoryRepo) CreateAutomation(ctx context.Context, automation *marketing.Automation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if automation.ID == 0 {
		automation.ID = r.allocateID()
	}
	r.automations[automation.ID] = automation
	return nil
}
func (r *MemoryRepo) GetAutomationByID(ctx context.Context, id int64) (*marketing.Automation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	automation := r.automations[id]
	if automation == nil {
		return nil, nil
	}
	copy := *automation
	return &copy, nil
}
func (r *MemoryRepo) UpdateAutomation(ctx context.Context, automation *marketing.Automation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.automations[automation.ID]; !ok {
		return marketing.ErrAutomationNotFound
	}
	r.automations[automation.ID] = automation
	return nil
}
func (r *MemoryRepo) ListAutomations(ctx context.Context, companyID int64) ([]*marketing.Automation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*marketing.Automation, 0)
	for _, automation := range r.automations {
		if companyID == 0 || automation.CompanyID == companyID {
			copy := *automation
			result = append(result, &copy)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r *MemoryRepo) ListActive(ctx context.Context, companyID int64) ([]*marketing.Automation, error) {
	items, err := r.ListAutomations(ctx, companyID)
	if err != nil {
		return nil, err
	}
	result := make([]*marketing.Automation, 0, len(items))
	for _, item := range items {
		if item.Active {
			result = append(result, item)
		}
	}
	return result, nil
}
func (r *MemoryRepo) DeleteAutomation(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.automations, id)
	return nil
}
func (r *MemoryRepo) CreateActivity(ctx context.Context, activity *marketing.AutomationActivity) error {
	return nil
}
func (r *MemoryRepo) GetActivities(ctx context.Context, automationID int64) ([]*marketing.AutomationActivity, error) {
	return nil, nil
}
