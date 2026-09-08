package loyaltystorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is an in-memory implementation of loyalty.Repository for tests and
// local development. Children (rules/rewards/mails) are stored inline on their
// program, mirroring the PostgresRepo read model.
type MemoryRepo struct {
	mu           sync.RWMutex
	programs     map[int64]*loyalty.LoyaltyProgram
	cards        map[int64]*loyalty.LoyaltyCard
	cardsByCode  map[string]int64
	couponPoints map[int64]*loyalty.OrderCouponPoints
	history      map[int64]*loyalty.LoyaltyHistory
	programSeq   int64
	ruleSeq      int64
	rewardSeq    int64
	mailSeq      int64
	cardSeq      int64
	pointsSeq    int64
	historySeq   int64
}

// NewMemoryRepo creates an empty in-memory loyalty repository.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		programs:     make(map[int64]*loyalty.LoyaltyProgram),
		cards:        make(map[int64]*loyalty.LoyaltyCard),
		cardsByCode:  make(map[string]int64),
		couponPoints: make(map[int64]*loyalty.OrderCouponPoints),
		history:      make(map[int64]*loyalty.LoyaltyHistory),
		programSeq:   1,
		ruleSeq:      1,
		rewardSeq:    1,
		mailSeq:      1,
		cardSeq:      1,
		pointsSeq:    1,
		historySeq:   1,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Programs (with nested rules, rewards, mails)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateProgram(ctx context.Context, p *loyalty.LoyaltyProgram) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.programSeq
	r.programSeq++
	now := time.Now().UTC()
	p.Audit.CreatedAt = now
	p.Audit.UpdatedAt = now
	materializeChildIDs(p)
	r.programs[p.ID] = cloneProgram(p)
	return nil
}

func (r *MemoryRepo) UpdateProgram(ctx context.Context, p *loyalty.LoyaltyProgram) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.programs[p.ID]
	if !ok {
		return platformerrors.NotFound("loyalty program not found", nil)
	}
	p.Audit.CreatedAt = existing.Audit.CreatedAt
	p.Audit.UpdatedAt = time.Now().UTC()
	materializeChildIDs(p)
	r.programs[p.ID] = cloneProgram(p)
	return nil
}

func (r *MemoryRepo) GetProgramByID(ctx context.Context, id int64) (*loyalty.LoyaltyProgram, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.programs[id]
	if !ok {
		return nil, platformerrors.NotFound("loyalty program not found", nil)
	}
	return cloneProgram(p), nil
}

func (r *MemoryRepo) ListPrograms(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[loyalty.LoyaltyProgram], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []loyalty.LoyaltyProgram
	for _, p := range r.programs {
		if !matchesProgram(ctx, p, f) {
			continue
		}
		items = append(items, *cloneProgram(p))
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Sequence != items[j].Sequence {
			return items[i].Sequence < items[j].Sequence
		}
		return items[i].ID < items[j].ID
	})

	return paginate(items, page), nil
}

func (r *MemoryRepo) DeleteProgram(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.programs[id]; !ok {
		return platformerrors.NotFound("loyalty program not found", nil)
	}
	delete(r.programs, id)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Rules
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateRule(ctx context.Context, rule *loyalty.LoyaltyRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.programs[rule.ProgramID]
	if !ok {
		return platformerrors.NotFound("loyalty program not found", nil)
	}
	rule.ID = r.ruleSeq
	r.ruleSeq++
	clone := *rule
	p.Rules = append(p.Rules, clone)
	return nil
}

func (r *MemoryRepo) UpdateRule(ctx context.Context, rule *loyalty.LoyaltyRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		for i := range p.Rules {
			if p.Rules[i].ID == rule.ID {
				clone := *rule
				p.Rules[i] = clone
				return nil
			}
		}
	}
	return platformerrors.NotFound("loyalty rule not found", nil)
}

func (r *MemoryRepo) DeleteRule(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		for i := range p.Rules {
			if p.Rules[i].ID == id {
				p.Rules = append(p.Rules[:i], p.Rules[i+1:]...)
				return nil
			}
		}
	}
	return platformerrors.NotFound("loyalty rule not found", nil)
}

// ─────────────────────────────────────────────────────────────────────────────
// Rewards
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateReward(ctx context.Context, reward *loyalty.LoyaltyReward) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.programs[reward.ProgramID]
	if !ok {
		return platformerrors.NotFound("loyalty program not found", nil)
	}
	reward.ID = r.rewardSeq
	r.rewardSeq++
	clone := *reward
	p.Rewards = append(p.Rewards, clone)
	return nil
}

func (r *MemoryRepo) UpdateReward(ctx context.Context, reward *loyalty.LoyaltyReward) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		for i := range p.Rewards {
			if p.Rewards[i].ID == reward.ID {
				clone := *reward
				p.Rewards[i] = clone
				return nil
			}
		}
	}
	return platformerrors.NotFound("loyalty reward not found", nil)
}

func (r *MemoryRepo) DeleteReward(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		for i := range p.Rewards {
			if p.Rewards[i].ID == id {
				p.Rewards = append(p.Rewards[:i], p.Rewards[i+1:]...)
				return nil
			}
		}
	}
	return platformerrors.NotFound("loyalty reward not found", nil)
}

func (r *MemoryRepo) GetRewardByID(ctx context.Context, id int64) (*loyalty.LoyaltyReward, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.programs {
		for i := range p.Rewards {
			if p.Rewards[i].ID == id {
				clone := p.Rewards[i]
				return &clone, nil
			}
		}
	}
	return nil, platformerrors.NotFound("loyalty reward not found", nil)
}

// ─────────────────────────────────────────────────────────────────────────────
// Mails (config only)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateMail(ctx context.Context, mail *loyalty.LoyaltyMail) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.programs[mail.ProgramID]
	if !ok {
		return platformerrors.NotFound("loyalty program not found", nil)
	}
	mail.ID = r.mailSeq
	r.mailSeq++
	mail.Audit.CreatedAt = time.Now().UTC()
	clone := *mail
	p.Mails = append(p.Mails, clone)
	return nil
}

func (r *MemoryRepo) UpdateMail(ctx context.Context, mail *loyalty.LoyaltyMail) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		for i := range p.Mails {
			if p.Mails[i].ID == mail.ID {
				clone := *mail
				clone.Audit.CreatedAt = p.Mails[i].Audit.CreatedAt
				clone.Audit.UpdatedAt = time.Now().UTC()
				p.Mails[i] = clone
				return nil
			}
		}
	}
	return platformerrors.NotFound("loyalty mail not found", nil)
}

func (r *MemoryRepo) DeleteMail(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.programs {
		for i := range p.Mails {
			if p.Mails[i].ID == id {
				p.Mails = append(p.Mails[:i], p.Mails[i+1:]...)
				return nil
			}
		}
	}
	return platformerrors.NotFound("loyalty mail not found", nil)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cards
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateCard(ctx context.Context, card *loyalty.LoyaltyCard) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	card.ID = r.cardSeq
	r.cardSeq++
	now := time.Now().UTC()
	card.Audit.CreatedAt = now
	card.Audit.UpdatedAt = now
	if card.Code == "" {
		card.Code = loyalty.GenerateCode()
	}
	if _, exists := r.cardsByCode[card.Code]; exists {
		return platformerrors.Conflict(fmt.Sprintf("loyalty card code '%s' already exists", card.Code), nil)
	}
	r.cards[card.ID] = cloneCard(card)
	r.cardsByCode[card.Code] = card.ID
	return nil
}

func (r *MemoryRepo) UpdateCard(ctx context.Context, card *loyalty.LoyaltyCard) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.cards[card.ID]
	if !ok {
		return platformerrors.NotFound("loyalty card not found", nil)
	}
	card.Audit.CreatedAt = existing.Audit.CreatedAt
	card.Audit.UpdatedAt = time.Now().UTC()
	delete(r.cardsByCode, existing.Code)
	r.cards[card.ID] = cloneCard(card)
	r.cardsByCode[card.Code] = card.ID
	return nil
}

func (r *MemoryRepo) GetCardByID(ctx context.Context, id int64) (*loyalty.LoyaltyCard, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	card, ok := r.cards[id]
	if !ok {
		return nil, platformerrors.NotFound("loyalty card not found", nil)
	}
	return cloneCard(card), nil
}

func (r *MemoryRepo) GetCardByCode(ctx context.Context, code string) (*loyalty.LoyaltyCard, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.cardsByCode[code]
	if !ok {
		return nil, platformerrors.NotFound("loyalty card not found", nil)
	}
	return cloneCard(r.cards[id]), nil
}

func (r *MemoryRepo) ListCards(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[loyalty.LoyaltyCard], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []loyalty.LoyaltyCard
	for _, c := range r.cards {
		if !matchesCard(c, f) {
			continue
		}
		items = append(items, *cloneCard(c))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })

	return paginateCards(items, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// History
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) AddHistory(ctx context.Context, h *loyalty.LoyaltyHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	h.ID = r.historySeq
	r.historySeq++
	h.CreatedAt = time.Now().UTC()
	clone := *h
	r.history[h.ID] = &clone
	return nil
}

func (r *MemoryRepo) ListHistoryByCard(ctx context.Context, cardID int64) ([]loyalty.LoyaltyHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []loyalty.LoyaltyHistory
	for _, h := range r.history {
		if h.CardID == cardID {
			out = append(out, *h)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Sale Order Coupon Points
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) UpsertCouponPoints(ctx context.Context, orderID, couponID int64, points float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.couponPoints {
		if p.OrderID == orderID && p.CouponID == couponID {
			p.Points = points
			return nil
		}
	}
	p := &loyalty.OrderCouponPoints{
		ID:        r.pointsSeq,
		OrderID:   orderID,
		CouponID:  couponID,
		Points:    points,
		CreatedAt: time.Now().UTC(),
	}
	r.pointsSeq++
	r.couponPoints[p.ID] = p
	return nil
}

func (r *MemoryRepo) RemoveCouponPointsByOrder(ctx context.Context, orderID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, p := range r.couponPoints {
		if p.OrderID == orderID {
			delete(r.couponPoints, id)
		}
	}
	return nil
}

func (r *MemoryRepo) ListCouponPointsByOrder(ctx context.Context, orderID int64) ([]loyalty.OrderCouponPoints, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []loyalty.OrderCouponPoints
	for _, p := range r.couponPoints {
		if p.OrderID == orderID {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func matchesProgram(ctx context.Context, p *loyalty.LoyaltyProgram, f *filter.Filter) bool {
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		if p.CompanyID == nil || *p.CompanyID != *companyID {
			return false
		}
	}
	if f == nil {
		return true
	}
	for _, c := range f.Criteria {
		switch c.Field {
		case "id":
			if p.ID != toInt64(c.Value) {
				return false
			}
		case "name":
			if !strings.Contains(strings.ToLower(p.Name), strings.ToLower(fmt.Sprintf("%v", c.Value))) {
				return false
			}
		case "active":
			if boolValue(p.Active) != boolValue(c.Value) {
				return false
			}
		case "program_type":
			if !fmtValueEqual(string(p.ProgramType), c.Value) {
				return false
			}
		case "applies_on":
			if !fmtValueEqual(string(p.AppliesOn), c.Value) {
				return false
			}
		case "trigger":
			if !fmtValueEqual(string(p.Trigger), c.Value) {
				return false
			}
		case "company_id":
			if p.CompanyID == nil || *p.CompanyID != toInt64(c.Value) {
				return false
			}
		}
	}
	return true
}

func matchesCard(c *loyalty.LoyaltyCard, f *filter.Filter) bool {
	if f == nil {
		return true
	}
	for _, cr := range f.Criteria {
		switch cr.Field {
		case "id":
			if c.ID != toInt64(cr.Value) {
				return false
			}
		case "program_id":
			if c.ProgramID != toInt64(cr.Value) {
				return false
			}
		case "partner_id":
			if c.PartnerID == nil || *c.PartnerID != toInt64(cr.Value) {
				return false
			}
		case "code":
			if !fmtValueEqual(c.Code, cr.Value) {
				return false
			}
		case "active":
			if boolValue(c.Active) != boolValue(cr.Value) {
				return false
			}
		case "order_id":
			if c.OrderID == nil || *c.OrderID != toInt64(cr.Value) {
				return false
			}
		}
	}
	return true
}

func boolValue(v any) bool {
	b, ok := v.(bool)
	if ok {
		return b
	}
	// Some callers pass filter values as strings ("true"/"false").
	return fmt.Sprintf("%v", v) == "true"
}

func fmtValueEqual(a string, v any) bool {
	return a == fmt.Sprintf("%v", v)
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		var out int64
		_, _ = fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &out)
		return out
	}
}

func paginate[T any](items []T, page pagination.PageRequest) pagination.PageResult[T] {
	total := int64(len(items))
	offset := page.Offset()
	limit := page.LimitClamped()
	var slice []T
	if offset < len(items) {
		end := offset + limit
		if end > len(items) {
			end = len(items)
		}
		slice = items[offset:end]
	} else {
		slice = []T{}
	}
	return pagination.NewPageResult(slice, total, page)
}

func paginateCards(items []loyalty.LoyaltyCard, page pagination.PageRequest) pagination.PageResult[loyalty.LoyaltyCard] {
	// narrow to the concrete type to satisfy pagination.NewPageResult generics
	total := int64(len(items))
	offset := page.Offset()
	limit := page.LimitClamped()
	var slice []loyalty.LoyaltyCard
	if offset < len(items) {
		end := offset + limit
		if end > len(items) {
			end = len(items)
		}
		slice = items[offset:end]
	} else {
		slice = []loyalty.LoyaltyCard{}
	}
	return pagination.NewPageResult(slice, total, page)
}

// materializeChildIDs assigns stable IDs and the real ProgramID to nested
// rules/rewards/mails so transactional child operations behave like the PG store.
func materializeChildIDs(program *loyalty.LoyaltyProgram) {
	for i := range program.Rules {
		program.Rules[i].ID = int64(i + 1)
		program.Rules[i].ProgramID = program.ID
	}
	for i := range program.Rewards {
		program.Rewards[i].ID = int64(i + 1)
		program.Rewards[i].ProgramID = program.ID
	}
	for i := range program.Mails {
		program.Mails[i].ID = int64(i + 1)
		program.Mails[i].ProgramID = program.ID
	}
}

func cloneProgram(p *loyalty.LoyaltyProgram) *loyalty.LoyaltyProgram {
	if p == nil {
		return nil
	}
	clone := *p
	clone.PricelistIDs = append([]int64(nil), p.PricelistIDs...)
	clone.Rules = append([]loyalty.LoyaltyRule(nil), p.Rules...)
	clone.Rewards = append([]loyalty.LoyaltyReward(nil), p.Rewards...)
	clone.Mails = append([]loyalty.LoyaltyMail(nil), p.Mails...)
	return &clone
}

func cloneCard(c *loyalty.LoyaltyCard) *loyalty.LoyaltyCard {
	if c == nil {
		return nil
	}
	clone := *c
	if c.CompanyID != nil {
		v := *c.CompanyID
		clone.CompanyID = &v
	}
	if c.PartnerID != nil {
		v := *c.PartnerID
		clone.PartnerID = &v
	}
	if c.OrderID != nil {
		v := *c.OrderID
		clone.OrderID = &v
	}
	return &clone
}