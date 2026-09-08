package analyticstorage

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/analytic"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo provides a thread-safe, in-memory implementation of analytic.Repository.
type MemoryRepo struct {
	mu            sync.RWMutex
	plans         map[int64]*analytic.AnalyticPlan
	rules         map[int64][]analytic.AnalyticApplicability // planID -> rules
	accounts      map[int64]*analytic.AnalyticAccount
	lines         map[int64]*analytic.AnalyticLine
	distModels    map[int64]*analytic.DistributionModel
	lastPlanID    int64
	lastRuleID    int64
	lastAccountID int64
	lastLineID    int64
	lastModelID   int64
	projectPlanID int64
}

// NewMemoryRepo initializes a MemoryRepo pre-seeded with the default analytic plans.
func NewMemoryRepo() *MemoryRepo {
	now := time.Now().UTC()
	repo := &MemoryRepo{
		plans:         make(map[int64]*analytic.AnalyticPlan),
		rules:         make(map[int64][]analytic.AnalyticApplicability),
		accounts:      make(map[int64]*analytic.AnalyticAccount),
		lines:         make(map[int64]*analytic.AnalyticLine),
		distModels:    make(map[int64]*analytic.DistributionModel),
		projectPlanID: 1,
	}

	project := &analytic.AnalyticPlan{
		ID: 1, Name: "Project", Description: "Default project plan (analytic.project_plan)",
		Sequence: 10, Color: 0, DefaultApplicability: analytic.AppOptional, Active: true,
		RootID: 1, CompleteName: "Project",
		Audit: audit.Fields{CreatedAt: now, UpdatedAt: now},
	}
	depts := &analytic.AnalyticPlan{
		ID: 2, Name: "Departments", Description: "Department cost center plan (demo)",
		Sequence: 20, Color: 1, DefaultApplicability: analytic.AppOptional, Active: true,
		RootID: 2, CompleteName: "Departments",
		Audit: audit.Fields{CreatedAt: now, UpdatedAt: now},
	}
	repo.plans[project.ID] = project
	repo.plans[depts.ID] = depts
	repo.lastPlanID = depts.ID

	accounts := []analytic.AnalyticAccount{
		{ID: 1, Name: "General", Code: "PRJ", PlanID: 1, RootPlanID: 1, Active: true, Currency: "USD"},
		{ID: 2, Name: "Administration", Code: "ADM", PlanID: 2, RootPlanID: 2, Active: true, Currency: "USD"},
		{ID: 3, Name: "Sales & Marketing", Code: "S&M", PlanID: 2, RootPlanID: 2, Active: true, Currency: "USD"},
	}
	for _, a := range accounts {
		a.Audit = audit.Fields{CreatedAt: now, UpdatedAt: now}
		clone := a
		repo.accounts[a.ID] = &clone
		if a.ID > repo.lastAccountID {
			repo.lastAccountID = a.ID
		}
	}

	return repo
}

// ─────────────────────────────────────────────────────────────────────────────
// Plans
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreatePlan(ctx context.Context, p *analytic.AnalyticPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPlanID++
	p.ID = r.lastPlanID
	now := time.Now().UTC()
	p.Active = true
	p.Audit = audit.NewFields(ctx)
	p.Audit.UpdatedAt = now
	if p.RootID == 0 {
		if p.ParentID != nil {
			if parent, ok := r.plans[*p.ParentID]; ok {
				p.RootID = parent.RootID
				p.ParentPath = string(parent.CompleteName)
			} else {
				return platformerrors.NotFound(fmt.Sprintf("parent analytic plan with ID %d not found", *p.ParentID))
			}
		} else {
			p.RootID = p.ID
		}
	}
	p.CompleteName = p.BuildCompleteName()

	clone := *p
	r.plans[p.ID] = &clone
	if len(p.Applicabilities) > 0 {
		r.rules[p.ID] = append([]analytic.AnalyticApplicability(nil), p.Applicabilities...)
	}
	return nil
}

func (r *MemoryRepo) GetPlanByID(ctx context.Context, id int64) (*analytic.AnalyticPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.plans[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", id))
	}
	clone := *p
	clone.Applicabilities = r.getRulesLocked(id)
	return &clone, nil
}

func (r *MemoryRepo) UpdatePlan(ctx context.Context, p *analytic.AnalyticPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.plans[p.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", p.ID))
	}
	if p.ParentID != nil && *p.ParentID == p.ID {
		return platformerrors.Validation("analytic plan cannot be its own parent", nil)
	}

	p.Audit = existing.Audit
	p.Audit.UpdatedAt = time.Now().UTC()
	if p.RootID == 0 {
		p.RootID = existing.RootID
	}
	if p.ParentPath == "" {
		p.ParentPath = existing.ParentPath
	}
	p.CompleteName = p.BuildCompleteName()

	clone := *p
	r.plans[p.ID] = &clone
	if p.Applicabilities != nil {
		r.rules[p.ID] = append([]analytic.AnalyticApplicability(nil), p.Applicabilities...)
	}
	return nil
}

func (r *MemoryRepo) DeletePlan(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plans[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", id))
	}

	// Block deletion while accounts reference the plan (mirrors DB RESTRICT).
	for _, a := range r.accounts {
		if a.PlanID == id {
			return platformerrors.Conflict("cannot delete analytic plan with associated accounts")
		}
	}

	// Cascade delete child plans and applicability rules.
	var toDelete []int64
	r.collectPlanChildrenLocked(id, &toDelete)
	for _, pid := range toDelete {
		delete(r.plans, pid)
		delete(r.rules, pid)
	}
	if r.projectPlanID == id {
		r.projectPlanID = 0
	}
	return nil
}

func (r *MemoryRepo) collectPlanChildrenLocked(id int64, acc *[]int64) {
	*acc = append(*acc, id)
	for _, p := range r.plans {
		if p.ParentID != nil && *p.ParentID == id {
			r.collectPlanChildrenLocked(p.ID, acc)
		}
	}
}

func (r *MemoryRepo) ListPlans(ctx context.Context, includeInactive bool) ([]analytic.AnalyticPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var plans []analytic.AnalyticPlan
	for _, p := range r.plans {
		if !includeInactive && !p.Active {
			continue
		}
		clone := *p
		plans = append(plans, clone)
	}
	sort.Slice(plans, func(i, j int) bool {
		if plans[i].Sequence == plans[j].Sequence {
			return plans[i].ID < plans[j].ID
		}
		return plans[i].Sequence < plans[j].Sequence
	})
	return plans, nil
}

func (r *MemoryRepo) GetChildrenPlans(ctx context.Context, parentID int64) ([]analytic.AnalyticPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var children []analytic.AnalyticPlan
	for _, p := range r.plans {
		if p.ParentID != nil && *p.ParentID == parentID {
			clone := *p
			children = append(children, clone)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].ID < children[j].ID
	})
	return children, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Applicabilities (G2)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) SetApplicability(ctx context.Context, a *analytic.AnalyticApplicability) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plans[a.PlanID]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", a.PlanID))
	}

	rules := r.rules[a.PlanID]
	for i := range rules {
		rule := &rules[i]
		if rule.BusinessDomain == a.BusinessDomain && sameOptionalCompanyID(rule.CompanyID, a.CompanyID) {
			rule.Applicability = a.Applicability
			rule.Sequence = a.Sequence
			a.ID = rule.ID
			return nil
		}
	}

	r.lastRuleID++
	a.ID = r.lastRuleID
	r.rules[a.PlanID] = append(rules, *a)
	return nil
}

func (r *MemoryRepo) GetApplicabilities(ctx context.Context, planID int64) ([]analytic.AnalyticApplicability, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.getRulesLocked(planID), nil
}

func (r *MemoryRepo) GetRelevantPlans(ctx context.Context, companyID int64, businessDomain analytic.BusinessDomain) ([]analytic.RelevantPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projectPlanID := r.projectPlanID

	type candidate struct {
		plan         analytic.AnalyticPlan
		rule         *analytic.AnalyticApplicability
		bestScore    float64
		bestSequence int
	}
	var candidates []candidate

	// Root plans only, restricted to those with at least one active account.
	for _, p := range r.plans {
		if p.ParentID != nil || !p.Active {
			continue
		}
		hasAccounts := false
		for _, a := range r.accounts {
			if a.PlanID == p.ID && a.Active {
				hasAccounts = true
				break
			}
		}
		if !hasAccounts {
			continue
		}

		applicability := p.DefaultApplicability
		var bestRule *analytic.AnalyticApplicability
		bestScore, bestSeq := 0.0, 0

		for _, rule := range r.rules[p.ID] {
			if rule.BusinessDomain != businessDomain {
				continue
			}
			if rule.CompanyID != nil && *rule.CompanyID != companyID {
				continue
			}
			score := 1.0 // matches business domain (+1)
			if rule.CompanyID != nil && *rule.CompanyID == companyID {
				score += 0.5 // company match weighs 0.5 (G2)
			}
			if bestRule == nil || score > bestScore ||
				(score == bestScore && rule.Sequence < bestSeq) {
				bestRule = &rule
				bestScore = score
				bestSeq = rule.Sequence
			}
		}
		if bestRule != nil {
			applicability = bestRule.Applicability
		}
		if applicability == analytic.AppUnavailable {
			continue
		}

		candidates = append(candidates, candidate{
			plan:         *p,
			rule:         bestRule,
			bestScore:    bestScore,
			bestSequence: bestSeq,
		})
	}

	relevant := make([]analytic.RelevantPlan, 0, len(candidates))
	for _, c := range candidates {
		columnName := "x_plan" + fmt.Sprintf("%d", c.plan.ID) + "_id"
		if c.plan.ID == projectPlanID {
			columnName = "account_id"
		}
		relevant = append(relevant, analytic.RelevantPlan{
			ID:            c.plan.ID,
			Name:          string(c.plan.Name),
			Applicability: planApplicability(c),
			ColumnName:    columnName,
		})
	}

	sort.Slice(relevant, func(i, j int) bool {
		return relevant[i].ID < relevant[j].ID
	})
	return relevant, nil
}

func planApplicability(c struct {
	plan         analytic.AnalyticPlan
	rule         *analytic.AnalyticApplicability
	bestScore    float64
	bestSequence int
}) analytic.Applicability {
	if c.rule != nil {
		return c.rule.Applicability
	}
	return c.plan.DefaultApplicability
}

func (r *MemoryRepo) getRulesLocked(planID int64) []analytic.AnalyticApplicability {
	rules := r.rules[planID]
	cloned := make([]analytic.AnalyticApplicability, len(rules))
	for i := range rules {
		cloned[i] = rules[i]
	}
	sort.Slice(cloned, func(i, j int) bool {
		if cloned[i].Sequence == cloned[j].Sequence {
			return cloned[i].ID < cloned[j].ID
		}
		return cloned[i].Sequence < cloned[j].Sequence
	})
	return cloned
}

func sameOptionalCompanyID(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateAccount(ctx context.Context, a *analytic.AnalyticAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plan, exists := r.plans[a.PlanID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", a.PlanID))
	}

	r.lastAccountID++
	a.ID = r.lastAccountID
	a.Active = true
	a.Audit = audit.NewFields(ctx)
	if a.RootPlanID == 0 {
		a.RootPlanID = plan.RootID
	}
	if a.Currency == "" {
		a.Currency = "USD"
	}

	clone := *a
	r.accounts[a.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetAccountByID(ctx context.Context, id int64) (*analytic.AnalyticAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, exists := r.accounts[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", id))
	}
	clone := *a
	return &clone, nil
}

func (r *MemoryRepo) UpdateAccount(ctx context.Context, a *analytic.AnalyticAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.accounts[a.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", a.ID))
	}
	a.Audit = existing.Audit
	a.Audit.UpdatedAt = time.Now().UTC()

	clone := *a
	r.accounts[a.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteAccount(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accounts[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", id))
	}
	for _, l := range r.lines {
		if l.AccountID == id {
			return platformerrors.Conflict("cannot delete analytic account with associated lines")
		}
	}
	delete(r.accounts, id)
	return nil
}

func (r *MemoryRepo) ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[analytic.AnalyticAccount], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []analytic.AnalyticAccount
	for _, a := range r.accounts {
		if !accountMatches(a, f) {
			continue
		}
		all = append(all, *a)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	total := int64(len(all))
	offset := page.Offset()
	limit := page.LimitClamped()
	if offset >= int(total) {
		return pagination.NewPageResult([]analytic.AnalyticAccount{}, total, page), nil
	}
	end := offset + limit
	if end > int(total) {
		end = int(total)
	}
	return pagination.NewPageResult(all[offset:end], total, page), nil
}

func (r *MemoryRepo) GetAccountTotals(ctx context.Context, accountID int64, fromDate, toDate *time.Time) (analytic.DebitCreditBalance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, exists := r.accounts[accountID]
	if !exists {
		return analytic.DebitCreditBalance{}, platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", accountID))
	}

	var total, debit, credit float64
	for _, l := range r.lines {
		if l.AccountID != accountID {
			continue
		}
		if fromDate != nil && l.Date.Before(*fromDate) {
			continue
		}
		if toDate != nil && l.Date.After(*toDate) {
			continue
		}
		total += l.Amount
		if l.Amount < 0 {
			debit += -l.Amount
		} else {
			credit += l.Amount
		}
	}

	return analytic.DebitCreditBalance{
		Debit:    math.Round(debit*10000) / 10000,
		Credit:   math.Round(credit*10000) / 10000,
		Balance:  math.Round(total*10000) / 10000,
		Currency: a.Currency,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Lines
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLine(ctx context.Context, l *analytic.AnalyticLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accounts[l.AccountID]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic account with ID %d not found", l.AccountID))
	}

	now := time.Now().UTC()
	if l.Date.IsZero() {
		l.Date = now
	}
	if l.Source == "" {
		l.Source = analytic.SourceManual
	}
	if l.Category == "" {
		l.Category = "other"
	}
	if l.Currency == "" {
		l.Currency = "USD"
	}

	r.lastLineID++
	l.ID = r.lastLineID
	l.Audit = audit.NewFields(ctx)
	l.Audit.UpdatedAt = now

	clone := *l
	r.lines[l.ID] = &clone
	return nil
}

func (r *MemoryRepo) CreateLines(ctx context.Context, lines []analytic.AnalyticLine) error {
	for i := range lines {
		if err := r.CreateLine(ctx, &lines[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *MemoryRepo) GetLineByID(ctx context.Context, id int64) (*analytic.AnalyticLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, exists := r.lines[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("analytic line with ID %d not found", id))
	}
	clone := *l
	return &clone, nil
}

func (r *MemoryRepo) UpdateLine(ctx context.Context, l *analytic.AnalyticLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.lines[l.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic line with ID %d not found", l.ID))
	}
	l.Audit = existing.Audit
	l.Audit.UpdatedAt = time.Now().UTC()

	clone := *l
	r.lines[l.ID] = &clone
	return nil
}

func (r *MemoryRepo) ListLines(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[analytic.AnalyticLine], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []analytic.AnalyticLine
	for _, l := range r.lines {
		if !lineMatches(l, f) {
			continue
		}
		all = append(all, *l)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].ID > all[j].ID
	})

	total := int64(len(all))
	offset := page.Offset()
	limit := page.LimitClamped()
	if offset >= int(total) {
		return pagination.NewPageResult([]analytic.AnalyticLine{}, total, page), nil
	}
	end := offset + limit
	if end > int(total) {
		end = int(total)
	}
	return pagination.NewPageResult(all[offset:end], total, page), nil
}

func (r *MemoryRepo) ListLinesByMoveLine(ctx context.Context, moveLineID int64) ([]analytic.AnalyticLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res []analytic.AnalyticLine
	for _, l := range r.lines {
		if l.MoveLineID != nil && *l.MoveLineID == moveLineID {
			res = append(res, *l)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID < res[j].ID
	})
	return res, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Distribution Models (G7)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateDistributionModel(ctx context.Context, m *analytic.DistributionModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastModelID++
	m.ID = r.lastModelID
	m.Active = true
	m.Audit = audit.NewFields(ctx)

	clone := *m
	r.distModels[m.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetDistributionModelByID(ctx context.Context, id int64) (*analytic.DistributionModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.distModels[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("analytic distribution model with ID %d not found", id))
	}
	clone := *m
	return &clone, nil
}

func (r *MemoryRepo) UpdateDistributionModel(ctx context.Context, m *analytic.DistributionModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.distModels[m.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic distribution model with ID %d not found", m.ID))
	}
	m.Audit = existing.Audit
	m.Audit.UpdatedAt = time.Now().UTC()

	clone := *m
	r.distModels[m.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteDistributionModel(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.distModels[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic distribution model with ID %d not found", id))
	}
	delete(r.distModels, id)
	return nil
}

func (r *MemoryRepo) ListDistributionModels(ctx context.Context) ([]analytic.DistributionModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []analytic.DistributionModel
	for _, m := range r.distModels {
		if !m.Active {
			continue
		}
		models = append(models, *m)
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].Sequence < models[j].Sequence
	})
	return models, nil
}

func (r *MemoryRepo) MatchDistribution(ctx context.Context, partnerID, partnerCategoryID *int64, companyID int64) (analytic.AnalyticDistribution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *analytic.DistributionModel
	for _, m := range r.distModels {
		if !m.Active {
			continue
		}
		if !distributionModelEligible(m, partnerID, partnerCategoryID, companyID) {
			continue
		}
		clone := *m
		if best == nil || distributionModelBetter(&clone, best) {
			best = &clone
		}
	}

	if best == nil || len(best.Distribution) == 0 {
		return nil, nil
	}
	return cloneDistribution(best.Distribution), nil
}

func distributionModelEligible(m *analytic.DistributionModel, partnerID, partnerCategoryID *int64, companyID int64) bool {
	if m.CompanyID != nil && *m.CompanyID != companyID {
		return false
	}
	// Generic (nil) rules stay eligible; specific rules only when their IDs match.
	if m.PartnerID != nil && (partnerID == nil || *m.PartnerID != *partnerID) {
		return false
	}
	if m.PartnerCategoryID != nil && (partnerCategoryID == nil || *m.PartnerCategoryID != *partnerCategoryID) {
		return false
	}
	return true
}

// distributionModelBetter ranks rules by specificity: partner > category > global,
// refined by sequence (lower sequence wins).
func distributionModelBetter(a, b *analytic.DistributionModel) bool {
	aSpec := specificity(a)
	bSpec := specificity(b)
	if aSpec != bSpec {
		return aSpec > bSpec
	}
	return a.Sequence < b.Sequence
}

func specificity(m *analytic.DistributionModel) int {
	spec := 0
	if m.PartnerID != nil {
		spec += 2
	}
	if m.PartnerCategoryID != nil {
		spec += 1
	}
	return spec
}

func cloneDistribution(d analytic.AnalyticDistribution) analytic.AnalyticDistribution {
	cloned := make(analytic.AnalyticDistribution, len(d))
	for k, v := range d {
		cloned[k] = v
	}
	return cloned
}

// ─────────────────────────────────────────────────────────────────────────────
// Project Plan (G8)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) GetProjectPlanID(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.projectPlanID, nil
}

func (r *MemoryRepo) SetProjectPlanID(ctx context.Context, planID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.plans[planID]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("analytic plan with ID %d not found", planID))
	}
	r.projectPlanID = planID
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Filtering helpers
// ─────────────────────────────────────────────────────────────────────────────

func accountMatches(a *analytic.AnalyticAccount, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 {
		return true
	}
	for _, c := range f.Criteria {
		val := fmt.Sprintf("%v", c.Value)
		switch c.Field {
		case "plan_id":
			if fmt.Sprintf("%d", a.PlanID) != val {
				return false
			}
		case "name":
			if !strings.Contains(strings.ToLower(string(a.Name)), strings.ToLower(val)) {
				return false
			}
		case "code":
			if !strings.Contains(strings.ToLower(a.Code), strings.ToLower(val)) {
				return false
			}
		case "partner_id":
			if a.PartnerID == nil || fmt.Sprintf("%d", *a.PartnerID) != val {
				return false
			}
		case "company_id":
			if a.CompanyID == nil || fmt.Sprintf("%d", *a.CompanyID) != val {
				return false
			}
		case "active":
			isActive := val == "true"
			if a.Active != isActive {
				return false
			}
		}
	}
	return true
}

func lineMatches(l *analytic.AnalyticLine, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 {
		return true
	}
	for _, c := range f.Criteria {
		val := fmt.Sprintf("%v", c.Value)
		switch c.Field {
		case "account_id":
			if fmt.Sprintf("%d", l.AccountID) != val {
				return false
			}
		case "partner_id":
			if l.PartnerID == nil || fmt.Sprintf("%d", *l.PartnerID) != val {
				return false
			}
		case "user_id":
			if fmt.Sprintf("%d", l.UserID) != val {
				return false
			}
		case "company_id":
			if fmt.Sprintf("%d", l.CompanyID) != val {
				return false
			}
		case "source":
			if string(l.Source) != val {
				return false
			}
		case "move_line_id":
			if l.MoveLineID == nil || fmt.Sprintf("%d", *l.MoveLineID) != val {
				return false
			}
		case "name":
			if !strings.Contains(strings.ToLower(string(l.Name)), strings.ToLower(val)) {
				return false
			}
		case "date_from":
			if from, err := time.Parse("2006-01-02", val); err == nil && l.Date.Before(from) {
				return false
			}
		case "date_to":
			if to, err := time.Parse("2006-01-02", val); err == nil && l.Date.After(to) {
				return false
			}
		}
	}
	return true
}
