package loyalty

import "math"

// EarnLine is a projection of a sale order line usable by the points engine.
type EarnLine struct {
	ProductID         int64
	Qty               float64
	PriceSubtotal     float64
	PriceTax          float64
	PriceTotal        float64
	IsRewardLine      bool
	RewardProgramType ProgramType
}

// ProgramEarn is the computed earning for a program on a given order.
type ProgramEarn struct {
	ProgramID   int64       `json:"program_id"`
	ProgramName string      `json:"program_name"`
	ProgramType ProgramType `json:"program_type"`
	Earned      float64     `json:"earned"`
	Eligible    bool        `json:"eligible"`
}

// ComputeProgramPoints computes the points a program earns on a set of order
// lines, mirroring Odoo `_program_check_compute_points`. enabledRuleID reports
// whether a with_code rule has been activated for the order; ruleMatch reports
// whether a rule applies to a product.
func ComputeProgramPoints(
	program *LoyaltyProgram,
	lines []EarnLine,
	enabledRuleID func(id int64) bool,
	ruleMatch func(rule *LoyaltyRule, productID int64) bool,
) (points float64, eligible bool) {
	if program == nil {
		return 0, false
	}

	// Misconfigured coupon-style program (no rules) that is code-triggered on the
	// current order is always "applicable" with zero earned points; the coupon
	// itself (loyalty.card) carries the spendable balance.
	if len(program.Rules) == 0 {
		if program.AppliesOn == AppliesOnCurrent && program.Trigger == TriggerWithCode {
			return 0, true
		}
		return 0, false
	}

	// eWallet bottomless-spending guard (Odoo): a payment program without any
	// trigger product never earns points from a plain order.
	if program.ProgramType == ProgramTypeEWallet && !hasTriggerProducts(program.Rules) {
		return 0, false
	}

	codeMatched := false
	minimumAmountMatched := false
	productQtyMatched := false
	var total float64

	for _, rule := range program.Rules {
		if !rule.Active {
			continue
		}
		if rule.Mode == RuleModeWithCode && !enabledRuleID(rule.ID) {
			continue
		}
		codeMatched = true

		var untaxed, tax, matchedQty float64
		for _, line := range lines {
			if line.IsRewardLine {
				continue
			}
			if !ruleMatch(&rule, line.ProductID) {
				continue
			}
			untaxed += line.PriceSubtotal
			tax += line.PriceTax
			matchedQty += line.Qty
		}

		amount := untaxed
		if rule.MinimumAmountTaxMode == TaxModeIncl {
			amount = untaxed + tax
		}
		if rule.MinimumAmount > amount {
			continue
		}
		minimumAmountMatched = true

		// Odoo requires at least one product of the order to match the rule.
		if matchedQty <= 0 || matchedQty < float64(rule.MinimumQty) {
			continue
		}
		productQtyMatched = true

		if rule.RewardPointAmount <= 0 {
			continue
		}

		switch rule.RewardPointMode {
		case RewardPointModeOrder:
			total += rule.RewardPointAmount
		case RewardPointModeMoney:
			var amountPaid float64
			for _, line := range lines {
				if line.IsRewardLine {
					continue
				}
				if line.RewardProgramType == ProgramTypeEWallet ||
					line.RewardProgramType == ProgramTypeGiftCard ||
					line.RewardProgramType == program.ProgramType {
					// Exclude gift card / eWallet redemption lines and lines of the
					// current program to prevent self-referential earning.
					continue
				}
				if ruleMatch(&rule, line.ProductID) {
					amountPaid += line.PriceTotal
				}
			}
			total += floor4(rule.RewardPointAmount * amountPaid)
		case RewardPointModeUnit:
			total += rule.RewardPointAmount * matchedQty
		}
	}

	if !program.IsNominative {
		switch {
		case !codeMatched:
			return 0, false
		case !minimumAmountMatched:
			return 0, false
		case !productQtyMatched:
			return 0, false
		}
	}

	return floor4(total), true
}

func hasTriggerProducts(rules []LoyaltyRule) bool {
	for _, rule := range rules {
		if len(rule.ProductIDs) > 0 {
			return true
		}
	}
	return false
}

func floor4(v float64) float64 {
	return math.Floor(v*10000) / 10000
}

// AvailablePoints computes the spendable points of a coupon for an order
// (mirrors Odoo `_get_real_points_for_coupon`): the card balance, plus the
// projected earning on the current order for non-future programs, minus the
// points already consumed by reward lines using the same coupon.
func AvailablePoints(cardPoints, projectedPoints, consumedPoints float64) float64 {
	available := cardPoints + projectedPoints - consumedPoints
	if available < 0 {
		return 0
	}
	return available
}