package loyaltyusecase

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/sale"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// RedeemInput requests the redemption of a reward from an applied coupon.
type RedeemInput struct {
	Code     string  `json:"code"`
	RewardID int64   `json:"reward_id"`
	Points   float64 `json:"points,omitempty"` // optional; for per_point rewards defaults to all available points
}

// RedeemCoupon adds the discount / free-product reward line of a coupon to the order.
func (uc *UseCase) RedeemCoupon(ctx context.Context, orderID int64, in RedeemInput) (*OrderPreview, error) {
	if uc.saleRepo == nil {
		return nil, platformerrors.Internal("sale repository is not configured", nil)
	}
	if in.Code == "" {
		return nil, platformerrors.Validation("coupon code is required", map[string]string{"code": "cannot be empty"})
	}
	if in.RewardID <= 0 {
		return nil, platformerrors.Validation("reward is required", map[string]string{"reward_id": "must reference a valid reward"})
	}

	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.State != sale.OrderStateDraft && order.State != sale.OrderStateSent {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot redeem on an order in state '%s'; must be draft or sent", order.State))
	}

	card, err := uc.repo.GetCardByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if !card.Active {
		return nil, platformerrors.Conflict("coupon is inactive")
	}
	if card.ExpirationDate != nil && nowUTC().After(*card.ExpirationDate) {
		return nil, platformerrors.Conflict("coupon has expired")
	}

	applied := false
	for _, couponID := range order.AppliedCouponIDs {
		if couponID == card.ID {
			applied = true
			break
		}
	}
	if !applied {
		return nil, platformerrors.Conflict("coupon is not applied to this order")
	}

	program, err := uc.repo.GetProgramByID(ctx, card.ProgramID)
	if err != nil {
		return nil, err
	}
	if program.AppliesOn == loyalty.AppliesOnFuture {
		return nil, platformerrors.Conflict("this coupon can only be redeemed on future orders")
	}

	reward, err := uc.repo.GetRewardByID(ctx, in.RewardID)
	if err != nil {
		return nil, err
	}
	if reward.ProgramID != program.ID {
		return nil, platformerrors.Conflict("reward does not belong to the coupon's program")
	}
	if !reward.Active {
		return nil, platformerrors.Conflict("reward is inactive")
	}

	projected, consumed, err := uc.couponBalance(ctx, order, card, program)
	if err != nil {
		return nil, err
	}
	available := loyalty.AvailablePoints(card.Points, projected, consumed)

	pointsToUse := reward.RequiredPoints
	if reward.DiscountMode == loyalty.DiscountModePerPoint {
		pointsToUse = in.Points
		if pointsToUse <= 0 {
			pointsToUse = available
		}
		if pointsToUse > available {
			pointsToUse = available
		}
	}

	eligibleAmount, err := uc.rewardEligibleAmount(ctx, order, program, reward)
	if err != nil {
		return nil, err
	}

	discountAmount, pointsCost := reward.DiscountAmount(available, pointsToUse, eligibleAmount)
	if pointsCost <= 0 {
		return nil, platformerrors.Conflict("reward cannot be redeemed with the current order contents")
	}
	if pointsCost > available+0.0001 {
		return nil, platformerrors.Conflict("insufficient points to redeem this reward")
	}
	if reward.RewardType == loyalty.RewardTypeDiscount && discountAmount <= 0 {
		return nil, platformerrors.Conflict("discount amount is zero; reward cannot be applied")
	}

	sequence := (len(order.Lines) + 1) * 10
	line, err := uc.buildRewardLine(ctx, *reward, *card, program, discountAmount, pointsCost, sequence)
	if err != nil {
		return nil, err
	}
	if err := validateRewardLine(line, *reward); err != nil {
		return nil, err
	}
	order.Lines = append(order.Lines, line)
	order.RecomputeTotals()

	// Re-evaluate the same-order earning so projected points reflect the new line.
	if program.AppliesOn == loyalty.AppliesOnCurrent || program.AppliesOn == loyalty.AppliesOnBoth {
		points, err := uc.projectedPointsForProgram(ctx, program, order)
		if err != nil {
			return nil, err
		}
		if err := uc.repo.UpsertCouponPoints(ctx, order.ID, card.ID, points); err != nil {
			return nil, err
		}
	}

	if err := uc.saleRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "loyalty reward redeemed",
		"order_id", order.ID, "code", card.Code, "reward_id", reward.ID,
		"points_cost", pointsCost, "discount_amount", discountAmount)
	return uc.PreviewOrder(ctx, order.ID)
}

// rewardEligibleAmount computes the base amount a discount reward applies to.
func (uc *UseCase) rewardEligibleAmount(ctx context.Context, order *sale.SaleOrder, program *loyalty.LoyaltyProgram, reward *loyalty.LoyaltyReward) (float64, error) {
	var eligible float64
	if reward.RewardType == loyalty.RewardTypeProduct {
		return 0, nil
	}
	if reward.DiscountApplicability == loyalty.DiscountApplicabilityCheapest {
		cheapest := 0.0
		found := false
		for _, l := range order.Lines {
			if l.IsRewardLine {
				continue
			}
			if !found || l.PriceTotal < cheapest {
				cheapest = l.PriceTotal
				found = true
			}
		}
		if !found {
			return 0, nil
		}
		return cheapest, nil
	}

	resolver, err := uc.buildCategoryResolver(ctx)
	if err != nil {
		return 0, err
	}
	_ = resolver.categorize(ctx, uc.productRepo, order)

	for _, l := range order.Lines {
		if l.IsRewardLine {
			continue
		}
		if reward.DiscountApplicability == loyalty.DiscountApplicabilitySpecific {
			if !rewardMatchesProduct(resolver, reward, l.ProductID) {
				continue
			}
		}
		eligible += l.PriceSubtotal
	}
	return eligible, nil
}

func rewardMatchesProduct(resolver *categoryResolver, reward *loyalty.LoyaltyReward, productID int64) bool {
	if len(reward.DiscountProductIDs) == 0 && reward.DiscountProductCategoryID == nil && reward.DiscountProductTagID == nil {
		return true
	}
	for _, id := range reward.DiscountProductIDs {
		if id == productID {
			return true
		}
	}
	if reward.DiscountProductCategoryID != nil {
		children, ok := resolver.children[*reward.DiscountProductCategoryID]
		if ok {
			cat, known := resolver.prodCat[productID]
			if known && children[cat] {
				return true
			}
		} else if resolver.prodCat[productID] == *reward.DiscountProductCategoryID {
			return true
		}
	}
	return false
}

// buildRewardLine assembles the sale order line representing a reward redemption.
func (uc *UseCase) buildRewardLine(ctx context.Context, reward loyalty.LoyaltyReward, card loyalty.LoyaltyCard, program *loyalty.LoyaltyProgram, discountAmount, pointsCost float64, sequence int) (sale.SaleOrderLine, error) {
	rewardID := reward.ID
	couponID := card.ID

	programName := string(program.Name)
	name := fmt.Sprintf("%s - %s", programName, reward.Description)
	if name == "" || name == programName+" - " {
		name = fmt.Sprintf("%s discount", programName)
	}

	if reward.RewardType == loyalty.RewardTypeProduct {
		line := sale.SaleOrderLine{
			Sequence:             sequence,
			ProductID:            *reward.RewardProductID,
			Name:                 i18n.NewTranslation(name),
			ProductUomQty:        float64(reward.RewardProductQty),
			ProductUom:           reward.RewardProductUomID,
			UnitPrice:            0,
			IsRewardLine:         true,
			RewardID:             &rewardID,
			CouponID:             &couponID,
			RewardIdentifierCode: card.Code,
			PointsCost:           pointsCost,
		}
		line.ComputeAmounts(nil)
		return line, nil
	}

	discountProductID := reward.DiscountLineProductID
	if discountProductID == nil {
		// Lazily create/reuse a per-program discount line product.
		if p, err := uc.ensureDiscountProduct(ctx, program); err == nil && p != nil {
			discountProductID = p
		}
	}
	if discountProductID != nil {
		line := sale.SaleOrderLine{
			Sequence:             sequence,
			ProductID:            *discountProductID,
			Name:                 i18n.NewTranslation(name),
			ProductUomQty:        1,
			UnitPrice:            -discountAmount,
			IsRewardLine:         true,
			RewardID:             &rewardID,
			CouponID:             &couponID,
			RewardIdentifierCode: card.Code,
			PointsCost:           pointsCost,
		}
		line.ComputeAmounts(nil)
		return line, nil
	}

	// Fallback: represent the discount on a synthetic line without a product.
	line := sale.SaleOrderLine{
		Sequence:             sequence,
		Name:                 i18n.NewTranslation(name),
		UnitPrice:            -discountAmount,
		ProductUomQty:        1,
		IsRewardLine:         true,
		RewardID:             &rewardID,
		CouponID:             &couponID,
		RewardIdentifierCode: card.Code,
		PointsCost:           pointsCost,
	}
	line.ComputeAmounts(nil)
	return line, nil
}

// ensureDiscountProduct creates a service template used to carry loyalty discount lines.
func (uc *UseCase) ensureDiscountProduct(ctx context.Context, program *loyalty.LoyaltyProgram) (*int64, error) {
	if uc.productRepo == nil {
		return nil, nil
	}
	pt := &product.ProductTemplate{
		Name:        i18n.NewTranslation(fmt.Sprintf("Loyalty Discount (%s)", string(program.Name))),
		Type:        product.ProductTypeService,
		SalePrice:   0,
		CostPrice:   0,
		SaleOK:      true,
		PurchaseOK:  false,
		Active:      true,
		CompanyID:   program.CompanyID,
		Description: "Discount line generated by the loyalty & rewards engine",
	}
	if err := uc.productRepo.CreateTemplate(ctx, pt); err != nil {
		return nil, platformerrors.Internal("failed to create discount line product", err)
	}
	return &pt.ID, nil
}

func validateRewardLine(line sale.SaleOrderLine, reward loyalty.LoyaltyReward) error {
	if line.ProductID <= 0 && reward.RewardType == loyalty.RewardTypeProduct {
		return platformerrors.Conflict("reward product is not configured")
	}
	return nil
}