package analytic

import (
	"math"
	"strconv"
	"strings"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// DistributionTolerance is the allowed rounding deviation when asserting that a
// distribution totals 100% (mirrors Odoo's rounding to 2 decimal places).
const DistributionTolerance = 0.01

// AnalyticDistribution is a JSON map {account_ids: percentage} stored as JSONB.
// The key may be a comma-separated combination of account IDs from several plans
// (e.g. "1,5"), which represents a partnership of accounts together (G3).
type AnalyticDistribution map[string]float64

// DistributionKey builds a canonical JSON map key from a sorted set of account IDs.
// e.g. []int64{1, 5} -> "1,5".
func DistributionKey(accountIDs []int64) string {
	ids := append([]int64(nil), accountIDs...)
	sortInt64(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}

// MapFromAccountIDs converts a list of account IDs into a distribution where every
// account combination is allocated 100% ({"id": 100} or {"1,5": 100}).
func MapFromAccountIDs(accountIDs []int64) AnalyticDistribution {
	if len(accountIDs) == 0 {
		return nil
	}
	return AnalyticDistribution{DistributionKey(accountIDs): 100}
}

// AccountIDs extracts the flattened, unique account IDs referenced by the distribution.
func (d AnalyticDistribution) AccountIDs() []int64 {
	seen := make(map[int64]bool)
	var ids []int64
	for key := range d {
		for _, part := range strings.Split(key, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// Total returns the sum of all percentages in the distribution.
func (d AnalyticDistribution) Total() float64 {
	var total float64
	for _, pct := range d {
		total += pct
	}
	return total
}

// Validate100 asserts that the distribution percentages total 100% within the
// allowed tolerance (equivalent to Odoo's _validate_distribution).
func (d AnalyticDistribution) Validate100() error {
	if len(d) == 0 {
		return nil
	}
	total := d.Total()
	if math.Abs(total-100) > DistributionTolerance {
		return platformerrors.Validation("analytic distribution must total 100%", map[string]any{
			"total":      math.Round(total*100) / 100,
			"percentage": "the distribution must sum to exactly 100.00%",
		})
	}
	return nil
}

// Merge merges a new distribution into an old one preserving the values of keys not
// present in the new distribution (equivalent to Odoo's _merge_distribution).
func Merge(old, newDist AnalyticDistribution) MergeResult {
	if len(newDist) == 0 {
		return MergeResult{Merged: cloneDistribution(old), Changed: false}
	}
	if len(old) == 0 {
		return MergeResult{Merged: cloneDistribution(newDist), Changed: true}
	}

	merged := cloneDistribution(newDist)
	changed := false
	for key, value := range old {
		if _, exists := merged[key]; !exists {
			merged[key] = value
			changed = true
		}
	}

	// Report Changed=true even when only values were overridden.
	for key, value := range newDist {
		if oldValue, ok := old[key]; ok && math.Abs(oldValue-value) > DistributionTolerance {
			changed = true
		}
	}

	return MergeResult{Merged: merged, Changed: changed}
}

func cloneDistribution(d AnalyticDistribution) AnalyticDistribution {
	if d == nil {
		return nil
	}
	cloned := make(AnalyticDistribution, len(d))
	for k, v := range d {
		cloned[k] = v
	}
	return cloned
}

func sortInt64(ids []int64) {
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
}

// MergeResult is the outcome of merging two distributions.
type MergeResult struct {
	Merged  AnalyticDistribution `json:"merged"`
	Changed bool                 `json:"changed"`
}

// DistributionModel represents an automatic analytic distribution rule
// (account.analytic.distribution.model in Odoo - G7).
//
// Matching order follows Odoo: partner_id, then partner_category_id, then global,
// refined by sequence. A more specific rule always wins over a generic one.
type DistributionModel struct {
	ID                int64                `json:"id"`
	Sequence          int                  `json:"sequence"`
	PartnerID         *int64               `json:"partner_id,omitempty"`
	PartnerCategoryID *int64               `json:"partner_category_id,omitempty"`
	CompanyID         *int64               `json:"company_id,omitempty"`
	Distribution      AnalyticDistribution `json:"distribution"`
	Active            bool                 `json:"active"`
	Audit             audit.Fields         `json:"audit"`
}

// Validate checks DistributionModel constraints.
func (m *DistributionModel) Validate() error {
	if len(m.Distribution) == 0 {
		return platformerrors.Validation("distribution model requires a distribution", nil)
	}
	if err := m.Distribution.Validate100(); err != nil {
		return err
	}
	if m.PartnerID != nil && m.PartnerCategoryID != nil {
		return platformerrors.Validation("distribution model cannot target both partner and partner category", nil)
	}
	return nil
}