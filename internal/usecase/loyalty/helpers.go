package loyaltyusecase

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/sale"
	platformerrors "cashflow_backend/internal/platform/errors"
)

func nowUTC() time.Time { return time.Now().UTC() }

// categoryResolver answers "does product P belong to category C or a descendant
// of C" using the product catalog's category hierarchy.
type categoryResolver struct {
	children map[int64]map[int64]bool // children[C] = {C, all descendants}
	prodCat  map[int64]int64          // productID -> categoryID
}

func (uc *UseCase) buildCategoryResolver(ctx context.Context) (*categoryResolver, error) {
	res := &categoryResolver{
		children: map[int64]map[int64]bool{},
		prodCat:  map[int64]int64{},
	}
	if uc.productRepo == nil {
		return res, nil
	}
	categories, err := uc.productRepo.ListCategories(ctx)
	if err != nil {
		return nil, platformerrors.Internal("failed to load product categories", err)
	}
	parent := map[int64]*int64{}
	for _, c := range categories {
		parent[c.ID] = c.ParentID
	}
	for _, c := range categories {
		cur := c.ID
		if _, ok := res.children[cur]; !ok {
			res.children[cur] = map[int64]bool{cur: true}
		}
		res.children[cur][cur] = true
		pid := c.ParentID
		seen := map[int64]bool{cur: true}
		for pid != nil && *pid > 0 && !seen[*pid] {
			if _, ok := res.children[*pid]; !ok {
				res.children[*pid] = map[int64]bool{}
			}
			res.children[*pid][*pid] = true
			res.children[*pid][cur] = true
			seen[*pid] = true
			pid = parent[*pid]
		}
	}
	return res, nil
}

// categorize assigns the category of each distinct product appearing in the order.
func (r *categoryResolver) categorize(ctx context.Context, repo product.Repository, order *sale.SaleOrder) error {
	if repo == nil {
		return nil
	}
	for _, l := range order.Lines {
		if _, ok := r.prodCat[l.ProductID]; ok {
			continue
		}
		pt, err := repo.GetTemplateByID(ctx, l.ProductID)
		if err != nil {
			continue
		}
		if pt.CategoryID != nil {
			r.prodCat[l.ProductID] = *pt.CategoryID
		}
	}
	return nil
}

// matchesProduct mirrors loyalty rule product resolution (ids, then category).
// A rule with no product restriction matches every order line.
func (r *categoryResolver) matchesProduct(rule *loyalty.LoyaltyRule, productID int64) bool {
	if len(rule.ProductIDs) == 0 && rule.ProductCategoryID == nil && rule.ProductTagID == nil {
		return true
	}
	for _, id := range rule.ProductIDs {
		if id == productID {
			return true
		}
	}
	if rule.ProductCategoryID != nil {
		children, ok := r.children[*rule.ProductCategoryID]
		if ok {
			cat, known := r.prodCat[productID]
			if known && children[cat] {
				return true
			}
		} else if r.prodCat[productID] == *rule.ProductCategoryID {
			// Fallback when the catalog has not been indexed (e.g. in-memory repo).
			return true
		}
	}
	return false
}

// enabledRuleID builds the rule-enablement check for a given order.
func (uc *UseCase) enabledRuleIDFor(order *sale.SaleOrder) func(id int64) bool {
	enabled := make(map[int64]bool, len(order.CodeEnabledRuleIDs))
	for _, id := range order.CodeEnabledRuleIDs {
		enabled[id] = true
	}
	return func(id int64) bool { return enabled[id] }
}

// earnLinesFromOrder projects sale order lines into engine input, resolving the
// program type of reward lines so the money earning mode can exclude them.
func (uc *UseCase) earnLinesFromOrder(ctx context.Context, order *sale.SaleOrder) ([]loyalty.EarnLine, error) {
	lines := make([]loyalty.EarnLine, 0, len(order.Lines))
	for _, l := range order.Lines {
		el := loyalty.EarnLine{
			ProductID:     l.ProductID,
			Qty:           l.ProductUomQty,
			PriceSubtotal: l.PriceSubtotal,
			PriceTax:      l.PriceTax,
			PriceTotal:    l.PriceTotal,
			IsRewardLine:  l.IsRewardLine,
		}
		if l.IsRewardLine && l.CouponID != nil {
			if card, err := uc.repo.GetCardByID(ctx, *l.CouponID); err == nil && card != nil {
				if program, err := uc.repo.GetProgramByID(ctx, card.ProgramID); err == nil && program != nil {
					el.RewardProgramType = program.ProgramType
				}
			}
		}
		lines = append(lines, el)
	}
	return lines, nil
}

// projectedPointsForProgram computes the points a program would earn on an order.
func (uc *UseCase) projectedPointsForProgram(ctx context.Context, program *loyalty.LoyaltyProgram, order *sale.SaleOrder) (float64, error) {
	lines, err := uc.earnLinesFromOrder(ctx, order)
	if err != nil {
		return 0, err
	}
	resolver, err := uc.buildCategoryResolver(ctx)
	if err != nil {
		return 0, err
	}
	_ = resolver.categorize(ctx, uc.productRepo, order)
	points, _ := loyalty.ComputeProgramPoints(
		program,
		lines,
		uc.enabledRuleIDFor(order),
		resolver.matchesProduct,
	)
	return points, nil
}

// projectedPointsForCoupon returns the projected same-order points of a coupon and
// the points already consumed by reward lines for the same coupon.
func (uc *UseCase) couponBalance(ctx context.Context, order *sale.SaleOrder, card *loyalty.LoyaltyCard, program *loyalty.LoyaltyProgram) (projected, consumed float64, err error) {
	for _, l := range order.Lines {
		if l.CouponID != nil && *l.CouponID == card.ID {
			consumed += l.PointsCost
		}
	}
	if program.AppliesOn == loyalty.AppliesOnCurrent || program.AppliesOn == loyalty.AppliesOnBoth {
		points, err := uc.repo.ListCouponPointsByOrder(ctx, order.ID)
		if err != nil {
			return 0, 0, err
		}
		for _, p := range points {
			if p.CouponID == card.ID {
				projected = p.Points
				break
			}
		}
	}
	return projected, consumed, nil
}