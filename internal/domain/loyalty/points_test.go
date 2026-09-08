package loyalty

import "testing"

// support helpers ------------------------------------------------------------

func testProgram(pType ProgramType, rules []LoyaltyRule, appliesOn AppliesOn, trigger Trigger, nominative bool) *LoyaltyProgram {
	return &LoyaltyProgram{
		ProgramType:  pType,
		Rules:        rules,
		AppliesOn:    appliesOn,
		Trigger:      trigger,
		IsNominative: nominative,
	}
}

func testRule(mode RewardPointMode, amount float64) LoyaltyRule {
	return LoyaltyRule{
		Active:            true,
		RewardPointMode:   mode,
		RewardPointAmount: amount,
	}
}

// noopMatcher matches every product; noopEnabled never enables with_code rules.
func noopMatcher(*LoyaltyRule, int64) bool { return true }

// ComputeProgramPoints --------------------------------------------------------

func TestComputeProgramPointsOrderMode(t *testing.T) {
	program := testProgram(ProgramTypePromotion, []LoyaltyRule{{
		Active:            true,
		RewardPointMode:   RewardPointModeOrder,
		RewardPointAmount: 5,
		MinimumQty:        0,
	}}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{
		{ProductID: 1, Qty: 1, PriceSubtotal: 30, PriceTax: 0, PriceTotal: 30},
	}

	points, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher)
	if !eligible {
		t.Fatalf("expected eligible order-mode program")
	}
	if points != 5 {
		t.Fatalf("expected 5 points for order mode, got %v", points)
	}
}

func TestComputeProgramPointsMoneyMode(t *testing.T) {
	rule := testRule(RewardPointModeMoney, 0.5)
	rule.MinimumQty = 0
	program := testProgram(ProgramTypeLoyalty, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{
		{ProductID: 1, Qty: 1, PriceSubtotal: 60, PriceTax: 10, PriceTotal: 70},
		{ProductID: 2, Qty: 1, PriceSubtotal: 30, PriceTax: 0, PriceTotal: 30},
	}

	points, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher)
	if !eligible {
		t.Fatalf("expected eligible money-mode program")
	}
	if points != 50 { // floor(0.5 * 100)
		t.Fatalf("expected 50 points for money mode, got %v", points)
	}
}

func TestComputeProgramPointsMoneyModeExcludesRewardLines(t *testing.T) {
	rule := testRule(RewardPointModeMoney, 1)
	rule.MinimumQty = 0
	program := testProgram(ProgramTypeLoyalty, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{
		{ProductID: 1, Qty: 1, PriceSubtotal: 40, PriceTax: 0, PriceTotal: 40},
		// Reward line of the same program — must be excluded from amountPaid.
		{ProductID: 2, Qty: 1, PriceSubtotal: 50, PriceTax: 0, PriceTotal: 50, IsRewardLine: true, RewardProgramType: ProgramTypeLoyalty},
		// Gift card redemption line — must be excluded.
		{ProductID: 3, Qty: 1, PriceSubtotal: 10, PriceTax: 0, PriceTotal: 10, IsRewardLine: true, RewardProgramType: ProgramTypeGiftCard},
	}

	points, _ := ComputeProgramPoints(program, lines, nil, noopMatcher)
	if points != 40 { // only the plain line contributes
		t.Fatalf("expected 40 points excluding reward lines, got %v", points)
	}
}

func TestComputeProgramPointsUnitMode(t *testing.T) {
	rule := testRule(RewardPointModeUnit, 2)
	rule.MinimumQty = 0
	program := testProgram(ProgramTypeBuyXGetY, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{
		{ProductID: 1, Qty: 3, PriceSubtotal: 90, PriceTax: 0, PriceTotal: 90},
	}

	points, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher)
	if !eligible {
		t.Fatalf("expected eligible unit-mode program")
	}
	if points != 6 { // 2 × 3
		t.Fatalf("expected 6 points for unit mode, got %v", points)
	}
}

func TestComputeProgramPointsMinimumAmountTaxModes(t *testing.T) {
	base := testRule(RewardPointModeOrder, 1)
	base.MinimumAmount = 90
	base.MinimumQty = 0

	lines := []EarnLine{
		{ProductID: 1, Qty: 1, PriceSubtotal: 80, PriceTax: 20, PriceTotal: 100},
	}

	inclRule := base
	inclRule.MinimumAmountTaxMode = TaxModeIncl
	program := testProgram(ProgramTypePromotion, []LoyaltyRule{inclRule}, AppliesOnCurrent, TriggerAuto, false)
	if _, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher); !eligible {
		t.Fatalf("tax-inclusive minimum (100 >= 90) should be met")
	}

	exclRule := base
	exclRule.MinimumAmountTaxMode = TaxModeExcl
	program = testProgram(ProgramTypePromotion, []LoyaltyRule{exclRule}, AppliesOnCurrent, TriggerAuto, false)
	if _, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher); eligible {
		t.Fatalf("tax-exclusive minimum (80 < 90) should not be met")
	}
}

func TestComputeProgramPointsMinimumQtyGate(t *testing.T) {
	rule := testRule(RewardPointModeUnit, 1)
	rule.MinimumQty = 2
	program := testProgram(ProgramTypePromotion, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{{ProductID: 1, Qty: 1, PriceSubtotal: 10, PriceTax: 0, PriceTotal: 10}}

	if _, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher); eligible {
		t.Fatalf("quantity below minimum_qty should not be eligible")
	}
}

func TestComputeProgramPointsRequiresMatchingProduct(t *testing.T) {
	rule := testRule(RewardPointModeOrder, 1)
	rule.MinimumQty = 0
	program := testProgram(ProgramTypePromotion, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{{ProductID: 9, Qty: 2, PriceSubtotal: 50, PriceTax: 0, PriceTotal: 50}}

	if _, eligible := ComputeProgramPoints(program, lines, func(found bool) func(int64) bool {
		return func(int64) bool { return found }
	}(true), func(r *LoyaltyRule, productID int64) bool {
		return productID == 1
	}); eligible {
		t.Fatalf("no order product matching the rule should not be eligible")
	}
}

func TestComputeProgramPointsWithCodeRequiresEnabledRule(t *testing.T) {
	rule := testRule(RewardPointModeOrder, 1)
	rule.Mode = RuleModeWithCode
	rule.Code = "XMAS"
	program := testProgram(ProgramTypePromoCode, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerWithCode, false)

	lines := []EarnLine{{ProductID: 1, Qty: 1, PriceSubtotal: 100, PriceTax: 0, PriceTotal: 100}}

	if _, eligible := ComputeProgramPoints(program, lines, func(int64) bool { return false }, noopMatcher); eligible {
		t.Fatalf("inactive with_code rule must not make program eligible")
	}
	if points, eligible := ComputeProgramPoints(program, lines, func(int64) bool { return true }, noopMatcher); !eligible || points != 1 {
		t.Fatalf("enabled with_code rule should earn 1 point, got points=%v eligible=%v", points, eligible)
	}
}

func TestComputeProgramPointsNoRulesWithCodeCurrentIsApplicable(t *testing.T) {
	program := testProgram(ProgramTypeCoupons, nil, AppliesOnCurrent, TriggerWithCode, false)
	if _, eligible := ComputeProgramPoints(program, nil, nil, noopMatcher); !eligible {
		t.Fatalf("coupon program (no rules, with_code, current) must always be applicable")
	}

	future := testProgram(ProgramTypeCoupons, nil, AppliesOnFuture, TriggerWithCode, false)
	if _, eligible := ComputeProgramPoints(future, nil, nil, noopMatcher); eligible {
		t.Fatalf("future coupon program without rules must not be applicable")
	}
}

func TestComputeProgramPointsEWalletWithoutTriggerProducts(t *testing.T) {
	rule := testRule(RewardPointModeMoney, 1)
	rule.MinimumQty = 0
	program := testProgram(ProgramTypeEWallet, []LoyaltyRule{rule}, AppliesOnFuture, TriggerAuto, false)

	lines := []EarnLine{{ProductID: 1, Qty: 1, PriceSubtotal: 100, PriceTax: 0, PriceTotal: 100}}

	if _, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher); eligible {
		t.Fatalf("eWallet program without trigger products must never earn from a plain order")
	}
}

func TestComputeProgramPointsNominativeSkipsGates(t *testing.T) {
	rule := testRule(RewardPointModeMoney, 1)
	rule.MinimumAmount = 1000
	rule.MinimumQty = 5
	program := testProgram(ProgramTypeLoyalty, []LoyaltyRule{rule}, AppliesOnBoth, TriggerAuto, true)

	lines := []EarnLine{{ProductID: 1, Qty: 1, PriceSubtotal: 10, PriceTax: 0, PriceTotal: 10}}

	if _, eligible := ComputeProgramPoints(program, lines, nil, noopMatcher); !eligible {
		t.Fatalf("nominative programs must skip gate checks and report eligible")
	}
	if points, _ := ComputeProgramPoints(program, lines, nil, noopMatcher); points != 0 {
		t.Fatalf("nominative program below limits should still earn 0, got %v", points)
	}
}

func TestComputeProgramPointsFloorFour(t *testing.T) {
	rule := testRule(RewardPointModeMoney, 1)
	rule.MinimumQty = 0
	program := testProgram(ProgramTypeLoyalty, []LoyaltyRule{rule}, AppliesOnCurrent, TriggerAuto, false)

	lines := []EarnLine{{ProductID: 1, Qty: 1, PriceSubtotal: 1.99999, PriceTax: 0, PriceTotal: 1.99999}}

	points, _ := ComputeProgramPoints(program, lines, nil, noopMatcher)
	if points != 1.9999 { // floor4 truncates the 5th decimal
		t.Fatalf("expected 1.9999 after floor4, got %v", points)
	}
}

// DiscountAmount --------------------------------------------------------------

func TestDiscountAmountPercent(t *testing.T) {
	r := LoyaltyReward{
		RewardType:    RewardTypeDiscount,
		DiscountMode:  DiscountModePercent,
		Discount:      10,
		RequiredPoints: 5,
	}
	amount, cost := r.DiscountAmount(100, 0, 250)
	if amount != 25 || cost != 5 {
		t.Fatalf("expected discount 25 cost 5, got discount=%v cost=%v", amount, cost)
	}
}

func TestDiscountAmountPerOrder(t *testing.T) {
	r := LoyaltyReward{
		RewardType:    RewardTypeDiscount,
		DiscountMode:  DiscountModePerOrder,
		Discount:      20,
		RequiredPoints: 3,
	}
	amount, cost := r.DiscountAmount(100, 0, 999)
	if amount != 20 || cost != 3 {
		t.Fatalf("expected fixed 20 cost 3, got discount=%v cost=%v", amount, cost)
	}
}

func TestDiscountAmountPerPoint(t *testing.T) {
	r := LoyaltyReward{
		RewardType:    RewardTypeDiscount,
		DiscountMode:  DiscountModePerPoint,
		Discount:      1.5,
		RequiredPoints: 1,
	}
	amount, cost := r.DiscountAmount(10, 4, 999)
	if amount != 6 || cost != 4 { // 4 points × 1.5
		t.Fatalf("expected discount 6 cost 4, got discount=%v cost=%v", amount, cost)
	}
}

func TestDiscountAmountPerPointInsufficientBalance(t *testing.T) {
	r := LoyaltyReward{
		RewardType:    RewardTypeDiscount,
		DiscountMode:  DiscountModePerPoint,
		Discount:      1,
		RequiredPoints: 10,
	}
	amount, cost := r.DiscountAmount(5, 5, 999)
	if amount != 0 || cost != 0 {
		t.Fatalf("insufficient available points must yield zero, got discount=%v cost=%v", amount, cost)
	}
}

func TestDiscountAmountMaxAmountCap(t *testing.T) {
	r := LoyaltyReward{
		RewardType:      RewardTypeDiscount,
		DiscountMode:    DiscountModePercent,
		Discount:        50,
		RequiredPoints:  1,
		DiscountMaxAmount: 30,
	}
	amount, cost := r.DiscountAmount(100, 0, 200)
	if amount != 30 || cost != 1 { // 100 would exceed cap of 30
		t.Fatalf("expected capped discount 30 cost 1, got discount=%v cost=%v", amount, cost)
	}
}

func TestDiscountAmountProductRewardCost(t *testing.T) {
	r := LoyaltyReward{
		RewardType:     RewardTypeProduct,
		RewardProductID: int64Ptr(7),
		RewardProductQty: 1,
		RequiredPoints:  2,
	}
	amount, cost := r.DiscountAmount(10, 0, 50)
	if amount != 0 || cost != 2 {
		t.Fatalf("product reward yields no monetary discount and costs required points, got discount=%v cost=%v", amount, cost)
	}
}

// AvailablePoints -------------------------------------------------------------

func TestAvailablePoints(t *testing.T) {
	if got := AvailablePoints(100, 25, 10); got != 115 {
		t.Fatalf("expected 115 available points, got %v", got)
	}
	if got := AvailablePoints(5, 0, 10); got != 0 {
		t.Fatalf("expected 0 when balance turns negative, got %v", got)
	}
}

func int64Ptr(v int64) *int64 { return &v }