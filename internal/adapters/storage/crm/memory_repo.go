package crmstorage

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/crm"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo provides a thread-safe, comprehensive in-memory implementation of crm.Repository.
type MemoryRepo struct {
	mu           sync.RWMutex
	leads        map[int64]*crm.Lead
	stages       map[int64]*crm.Stage
	lostReasons  map[int64]*crm.LostReason
	tags         map[int64]*crm.Tag
	leadTags     map[int64][]int64 // leadID -> []tagID
	lastLeadID   int64
	lastStageID  int64
	lastReasonID int64
	lastTagID    int64
}

// NewMemoryRepo initializes a MemoryRepo pre-seeded with standard stages, lost reasons, and tags.
func NewMemoryRepo() *MemoryRepo {
	repo := &MemoryRepo{
		leads:       make(map[int64]*crm.Lead),
		stages:      make(map[int64]*crm.Stage),
		lostReasons: make(map[int64]*crm.LostReason),
		tags:        make(map[int64]*crm.Tag),
		leadTags:    make(map[int64][]int64),
	}

	now := time.Now().UTC()

	// Seed Standard Stages
	stages := []crm.Stage{
		{ID: 1, Name: "New", Sequence: 10, IsWon: false, IsClosed: false, Fold: false, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Qualified", Sequence: 20, IsWon: false, IsClosed: false, Fold: false, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Proposition", Sequence: 30, IsWon: false, IsClosed: false, Fold: false, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Won", Sequence: 40, IsWon: true, IsClosed: true, Fold: false, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, s := range stages {
		clone := s
		repo.stages[s.ID] = &clone
		if s.ID > repo.lastStageID {
			repo.lastStageID = s.ID
		}
	}

	// Seed Standard Lost Reasons
	reasons := []crm.LostReason{
		{ID: 1, Name: "Too expensive", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "We don't have people/skills", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Not enough features", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Lost to competitor", Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, r := range reasons {
		clone := r
		repo.lostReasons[r.ID] = &clone
		if r.ID > repo.lastReasonID {
			repo.lastReasonID = r.ID
		}
	}

	// Seed Standard Tags
	tags := []crm.Tag{
		{ID: 1, Name: "Software", Color: 1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Consulting", Color: 2, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Support", Color: 3, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Enterprise", Color: 4, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, t := range tags {
		clone := t
		repo.tags[t.ID] = &clone
		if t.ID > repo.lastTagID {
			repo.lastTagID = t.ID
		}
	}

	return repo
}

// ─────────────────────────────────────────────────────────────────────────────
// Leads & Opportunities
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLead(ctx context.Context, lead *crm.Lead) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastLeadID++
	lead.ID = r.lastLeadID
	now := time.Now().UTC()
	lead.CreatedAt = now
	lead.UpdatedAt = now
	if !lead.Active && lead.LostReasonID == nil {
		lead.Active = true
	}
	lead.ComputeProratedRevenue()

	clone := *lead
	r.leads[lead.ID] = &clone

	if len(lead.TagIDs) > 0 {
		r.leadTags[lead.ID] = append([]int64(nil), lead.TagIDs...)
	}

	return nil
}

func (r *MemoryRepo) GetLeadByID(ctx context.Context, id int64) (*crm.Lead, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lead, exists := r.leads[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("lead/opportunity with ID %d not found", id))
	}

	clone := *lead
	clone.Tags = r.getTagsForLeadLocked(lead.ID)
	clone.TagIDs = r.leadTags[lead.ID]
	return &clone, nil
}

func (r *MemoryRepo) UpdateLead(ctx context.Context, lead *crm.Lead) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.leads[lead.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("lead/opportunity with ID %d not found", lead.ID))
	}

	lead.CreatedAt = existing.CreatedAt
	lead.UpdatedAt = time.Now().UTC()
	lead.ComputeProratedRevenue()

	clone := *lead
	r.leads[lead.ID] = &clone

	if lead.TagIDs != nil {
		r.leadTags[lead.ID] = append([]int64(nil), lead.TagIDs...)
	}

	return nil
}

func (r *MemoryRepo) DeleteLead(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.leads[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("lead/opportunity with ID %d not found", id))
	}

	delete(r.leads, id)
	delete(r.leadTags, id)
	return nil
}

func (r *MemoryRepo) ListLeads(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[crm.Lead], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []crm.Lead
	for _, l := range r.leads {
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, crit := range f.Criteria {
				valStr := fmt.Sprintf("%v", crit.Value)
				switch crit.Field {
				case "type":
					if string(l.Type) != valStr {
						match = false
					}
				case "stage_id":
					if fmt.Sprintf("%d", l.StageID) != valStr {
						match = false
					}
				case "salesperson_id":
					if l.SalespersonID == nil || fmt.Sprintf("%d", *l.SalespersonID) != valStr {
						match = false
					}
				case "partner_id":
					if l.PartnerID == nil || fmt.Sprintf("%d", *l.PartnerID) != valStr {
						match = false
					}
				case "active":
					isActive := valStr == "true"
					if l.Active != isActive {
						match = false
					}
				case "search", "name":
					s := strings.ToLower(valStr)
					if !strings.Contains(strings.ToLower(l.Name), s) &&
						!strings.Contains(strings.ToLower(l.ContactName), s) &&
						!strings.Contains(strings.ToLower(l.PartnerName), s) &&
						!strings.Contains(strings.ToLower(l.EmailFrom), s) {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}

		clone := *l
		clone.Tags = r.getTagsForLeadLocked(l.ID)
		clone.TagIDs = r.leadTags[l.ID]
		all = append(all, clone)
	}

	// Sort by ID desc by default
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID > all[j].ID
	})

	total := int64(len(all))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]crm.Lead{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(all[offset:end], total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Stages
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateStage(ctx context.Context, stage *crm.Stage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastStageID++
	stage.ID = r.lastStageID
	now := time.Now().UTC()
	stage.CreatedAt = now
	stage.UpdatedAt = now
	stage.Active = true

	clone := *stage
	r.stages[stage.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetStageByID(ctx context.Context, id int64) (*crm.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, exists := r.stages[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("stage with ID %d not found", id))
	}
	clone := *s
	return &clone, nil
}

func (r *MemoryRepo) UpdateStage(ctx context.Context, stage *crm.Stage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.stages[stage.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("stage with ID %d not found", stage.ID))
	}

	stage.CreatedAt = existing.CreatedAt
	stage.UpdatedAt = time.Now().UTC()

	clone := *stage
	r.stages[stage.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteStage(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.stages[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("stage with ID %d not found", id))
	}

	// Check if any leads reference this stage
	for _, l := range r.leads {
		if l.StageID == id {
			return platformerrors.Conflict("cannot delete stage with associated leads/opportunities")
		}
	}

	delete(r.stages, id)
	return nil
}

func (r *MemoryRepo) ListStages(ctx context.Context) ([]crm.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var stages []crm.Stage
	for _, s := range r.stages {
		if s.Active {
			stages = append(stages, *s)
		}
	}

	sort.Slice(stages, func(i, j int) bool {
		if stages[i].Sequence == stages[j].Sequence {
			return stages[i].ID < stages[j].ID
		}
		return stages[i].Sequence < stages[j].Sequence
	})

	return stages, nil
}

func (r *MemoryRepo) GetWonStage(ctx context.Context) (*crm.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.stages {
		if s.Active && s.IsWon {
			clone := *s
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("won stage not configured in system")
}

func (r *MemoryRepo) GetInitialStage(ctx context.Context) (*crm.Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activeStages []crm.Stage
	for _, s := range r.stages {
		if s.Active && !s.IsClosed && !s.IsWon {
			activeStages = append(activeStages, *s)
		}
	}

	if len(activeStages) == 0 {
		return nil, platformerrors.NotFound("no initial active stages available")
	}

	sort.Slice(activeStages, func(i, j int) bool {
		return activeStages[i].Sequence < activeStages[j].Sequence
	})

	res := activeStages[0]
	return &res, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Lost Reasons
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLostReason(ctx context.Context, reason *crm.LostReason) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastReasonID++
	reason.ID = r.lastReasonID
	now := time.Now().UTC()
	reason.CreatedAt = now
	reason.UpdatedAt = now
	reason.Active = true

	clone := *reason
	r.lostReasons[reason.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetLostReasonByID(ctx context.Context, id int64) (*crm.LostReason, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reason, exists := r.lostReasons[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d not found", id))
	}
	clone := *reason
	return &clone, nil
}

func (r *MemoryRepo) UpdateLostReason(ctx context.Context, reason *crm.LostReason) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.lostReasons[reason.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d not found", reason.ID))
	}

	reason.CreatedAt = existing.CreatedAt
	reason.UpdatedAt = time.Now().UTC()

	clone := *reason
	r.lostReasons[reason.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteLostReason(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.lostReasons[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("lost reason with ID %d not found", id))
	}

	delete(r.lostReasons, id)
	return nil
}

func (r *MemoryRepo) ListLostReasons(ctx context.Context) ([]crm.LostReason, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var reasons []crm.LostReason
	for _, res := range r.lostReasons {
		if res.Active {
			reasons = append(reasons, *res)
		}
	}

	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].ID < reasons[j].ID
	})

	return reasons, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tags
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateTag(ctx context.Context, tag *crm.Tag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, t := range r.tags {
		if strings.EqualFold(string(t.Name), string(tag.Name)) {
			return platformerrors.Conflict(fmt.Sprintf("tag '%s' already exists", tag.Name))
		}
	}

	r.lastTagID++
	tag.ID = r.lastTagID
	now := time.Now().UTC()
	tag.CreatedAt = now
	tag.UpdatedAt = now
	tag.Active = true

	clone := *tag
	r.tags[tag.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetTagByID(ctx context.Context, id int64) (*crm.Tag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.tags[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("tag with ID %d not found", id))
	}
	clone := *t
	return &clone, nil
}

func (r *MemoryRepo) ListTags(ctx context.Context) ([]crm.Tag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tags []crm.Tag
	for _, t := range r.tags {
		if t.Active {
			tags = append(tags, *t)
		}
	}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})
	return tags, nil
}

func (r *MemoryRepo) AssignTags(ctx context.Context, leadID int64, tagIDs []int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.leads[leadID]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("lead with ID %d not found", leadID))
	}

	r.leadTags[leadID] = append([]int64(nil), tagIDs...)
	return nil
}

func (r *MemoryRepo) GetTagsByLeadID(ctx context.Context, leadID int64) ([]crm.Tag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.getTagsForLeadLocked(leadID), nil
}

func (r *MemoryRepo) getTagsForLeadLocked(leadID int64) []crm.Tag {
	tagIDs := r.leadTags[leadID]
	var res []crm.Tag
	for _, tid := range tagIDs {
		if t, ok := r.tags[tid]; ok && t.Active {
			res = append(res, *t)
		}
	}
	return res
}

// ─────────────────────────────────────────────────────────────────────────────
// Pipeline & Analytics
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) GetPipeline(ctx context.Context, salespersonID *int64) ([]crm.PipelineStageData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stages, err := r.ListStages(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]crm.PipelineStageData, len(stages))
	for i, stage := range stages {
		var opps []crm.Lead
		var totalExpected, totalProrated float64

		for _, l := range r.leads {
			if l.Type != crm.LeadTypeOpportunity || !l.Active || l.StageID != stage.ID {
				continue
			}
			if salespersonID != nil && (l.SalespersonID == nil || *l.SalespersonID != *salespersonID) {
				continue
			}

			clone := *l
			clone.Tags = r.getTagsForLeadLocked(l.ID)
			clone.TagIDs = r.leadTags[l.ID]
			opps = append(opps, clone)

			totalExpected += l.ExpectedRevenue
			totalProrated += l.ProratedRevenue
		}

		// Sort opportunities within each stage by ID desc
		sort.Slice(opps, func(a, b int) bool {
			return opps[a].ID > opps[b].ID
		})

		result[i] = crm.PipelineStageData{
			Stage:                stage,
			TotalOpportunities:   len(opps),
			TotalExpectedRevenue: math.Round(totalExpected*10000) / 10000,
			TotalProratedRevenue: math.Round(totalProrated*10000) / 10000,
			Opportunities:        opps,
		}
	}

	return result, nil
}

func (r *MemoryRepo) GetStats(ctx context.Context) (*crm.CRMStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &crm.CRMStats{
		Stages:         make([]crm.StageStat, 0),
		TopLostReasons: make([]crm.LostReasonStat, 0),
	}

	stageMap := make(map[int64]*crm.StageStat)
	for _, s := range r.stages {
		if s.Active {
			stageMap[s.ID] = &crm.StageStat{
				StageID:   s.ID,
				StageName: string(s.Name),
			}
		}
	}

	lostMap := make(map[int64]*crm.LostReasonStat)
	for _, lr := range r.lostReasons {
		if lr.Active {
			lostMap[lr.ID] = &crm.LostReasonStat{
				ReasonID:   lr.ID,
				ReasonName: string(lr.Name),
			}
		}
	}

	for _, l := range r.leads {
		if l.Type == crm.LeadTypeLead {
			stats.TotalLeads++
		} else if l.Type == crm.LeadTypeOpportunity {
			stats.TotalOpportunities++
			if l.Active {
				stats.TotalExpectedRevenue += l.ExpectedRevenue
				stats.TotalProratedRevenue += l.ProratedRevenue
			}
		}

		// Won evaluation
		stage := r.stages[l.StageID]
		isWon := (stage != nil && stage.IsWon) || l.Probability >= 100.0
		if isWon {
			stats.WonCount++
			stats.TotalWonRevenue += l.ExpectedRevenue
		}

		// Lost evaluation
		if l.LostReasonID != nil || (!l.Active && !isWon) {
			stats.LostCount++
			if l.LostReasonID != nil {
				if rstat, ok := lostMap[*l.LostReasonID]; ok {
					rstat.Count++
					rstat.LostRevenue += l.ExpectedRevenue
				}
			}
		}

		// Accumulate stage metrics
		if sstat, ok := stageMap[l.StageID]; ok && l.Active && l.Type == crm.LeadTypeOpportunity {
			sstat.Count++
			sstat.ExpectedRevenue += l.ExpectedRevenue
			sstat.ProratedRevenue += l.ProratedRevenue
		}
	}

	// Win Rate: Won / (Won + Lost) * 100
	totalDecided := stats.WonCount + stats.LostCount
	if totalDecided > 0 {
		stats.WinRate = math.Round((float64(stats.WonCount)/float64(totalDecided)*100.0)*100) / 100
	}

	// Conversion Rate: Opportunities / (Leads + Opportunities) * 100
	totalEntities := stats.TotalLeads + stats.TotalOpportunities
	if totalEntities > 0 {
		stats.ConversionRate = math.Round((float64(stats.TotalOpportunities)/float64(totalEntities)*100.0)*100) / 100
	}

	// Average Deal Size
	if stats.TotalOpportunities > 0 {
		stats.AvgDealSize = math.Round((stats.TotalExpectedRevenue/float64(stats.TotalOpportunities))*10000) / 10000
	}

	stats.TotalExpectedRevenue = math.Round(stats.TotalExpectedRevenue*10000) / 10000
	stats.TotalProratedRevenue = math.Round(stats.TotalProratedRevenue*10000) / 10000
	stats.TotalWonRevenue = math.Round(stats.TotalWonRevenue*10000) / 10000

	// Flatten stage stats in sequence order
	stagesList, _ := r.ListStages(ctx)
	for _, st := range stagesList {
		if sstat, ok := stageMap[st.ID]; ok {
			sstat.ExpectedRevenue = math.Round(sstat.ExpectedRevenue*10000) / 10000
			sstat.ProratedRevenue = math.Round(sstat.ProratedRevenue*10000) / 10000
			stats.Stages = append(stats.Stages, *sstat)
		}
	}

	// Flatten lost reasons stats
	for _, lr := range lostMap {
		if lr.Count > 0 {
			lr.LostRevenue = math.Round(lr.LostRevenue*10000) / 10000
			stats.TopLostReasons = append(stats.TopLostReasons, *lr)
		}
	}
	sort.Slice(stats.TopLostReasons, func(i, j int) bool {
		return stats.TopLostReasons[i].Count > stats.TopLostReasons[j].Count
	})

	return stats, nil
}
