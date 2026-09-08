// Package loyalty models the Loyalty & Rewards subsystem aligned to Odoo 19.0
// (loyalty.program, loyalty.rule, loyalty.reward, loyalty.card, loyalty.history,
// loyalty.mail and sale.order.coupon.points).
package loyalty

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ProgramType enumerates the supported loyalty program kinds (loyalty.program.program_type).
type ProgramType string

const (
	ProgramTypeCoupons           ProgramType = "coupons"            // Coupons
	ProgramTypeGiftCard          ProgramType = "gift_card"          // Gift Card
	ProgramTypeLoyalty           ProgramType = "loyalty"            // Loyalty Cards
	ProgramTypePromotion         ProgramType = "promotion"          // Promotions
	ProgramTypeEWallet           ProgramType = "ewallet"            // eWallet
	ProgramTypePromoCode         ProgramType = "promo_code"         // Discount Code
	ProgramTypeBuyXGetY          ProgramType = "buy_x_get_y"        // Buy X Get Y
	ProgramTypeNextOrderCoupons  ProgramType = "next_order_coupons" // Next Order Coupons
)

// AppliesOn dictates when earned points can be used (loyalty.program.applies_on).
type AppliesOn string

const (
	AppliesOnCurrent AppliesOn = "current" // Current order
	AppliesOnFuture  AppliesOn = "future"  // Future orders
	AppliesOnBoth    AppliesOn = "both"    // Current & Future orders
)

// Trigger dictates whether a program is applied automatically or via code.
type Trigger string

const (
	TriggerAuto     Trigger = "auto"
	TriggerWithCode Trigger = "with_code"
)

// RuleMode is the application mode of a loyalty rule (loyalty.rule.mode).
type RuleMode string

const (
	RuleModeAuto     RuleMode = "auto"
	RuleModeWithCode RuleMode = "with_code"
)

// RewardPointMode defines the earning unit of a rule (loyalty.rule.reward_point_mode).
type RewardPointMode string

const (
	RewardPointModeOrder RewardPointMode = "order" // per order
	RewardPointModeMoney RewardPointMode = "money" // per currency spent
	RewardPointModeUnit  RewardPointMode = "unit"  // per unit paid
)

// TaxMode indicates whether minimum amounts are tax-inclusive.
type TaxMode string

const (
	TaxModeIncl TaxMode = "incl" // Tax included
	TaxModeExcl TaxMode = "excl" // Tax excluded
)

// RewardType defines the nature of a reward (loyalty.reward.reward_type).
type RewardType string

const (
	RewardTypeDiscount RewardType = "discount" // Discount
	RewardTypeProduct  RewardType = "product"  // Free product
)

// DiscountMode defines how a discount reward is applied (loyalty.reward.discount_mode).
type DiscountMode string

const (
	DiscountModePercent  DiscountMode = "percent"   // %
	DiscountModePerOrder DiscountMode = "per_order" // fixed currency per order
	DiscountModePerPoint DiscountMode = "per_point" // currency per point
)

// DiscountApplicability defines the base of a discount reward.
type DiscountApplicability string

const (
	DiscountApplicabilityOrder    DiscountApplicability = "order"    // whole order
	DiscountApplicabilityCheapest DiscountApplicability = "cheapest" // cheapest product
	DiscountApplicabilitySpecific DiscountApplicability = "specific" // specific products
)

// MailTrigger defines when a loyalty mail is scheduled (loyalty.mail.trigger).
type MailTrigger string

const (
	MailTriggerCreate       MailTrigger = "create"        // At creation
	MailTriggerPointsReach  MailTrigger = "points_reach"  // When reaching points
)

// ProgramTypes is the ordered catalog of supported program types.
var ProgramTypes = []ProgramType{
	ProgramTypeCoupons,
	ProgramTypeGiftCard,
	ProgramTypeLoyalty,
	ProgramTypePromotion,
	ProgramTypeEWallet,
	ProgramTypePromoCode,
	ProgramTypeBuyXGetY,
	ProgramTypeNextOrderCoupons,
}

// LoyaltyProgram is a loyalty program (loyalty.program).
type LoyaltyProgram struct {
	ID                int64              `json:"id"`
	Name              string             `json:"name"`
	Active            bool               `json:"active"`
	Sequence          int                `json:"sequence"`
	CompanyID         *int64             `json:"company_id,omitempty"`
	Currency          string             `json:"currency"`
	PricelistIDs      []int64            `json:"pricelist_ids,omitempty"`
	ProgramType       ProgramType        `json:"program_type"`
	DateFrom          *time.Time         `json:"date_from,omitempty"`
	DateTo            *time.Time         `json:"date_to,omitempty"`
	LimitUsage        bool               `json:"limit_usage"`
	MaxUsage          int                `json:"max_usage"`
	AppliesOn         AppliesOn          `json:"applies_on"`
	Trigger           Trigger            `json:"trigger"`
	PortalVisible     bool               `json:"portal_visible"`
	PortalPointName   string             `json:"portal_point_name"`
	IsNominative      bool               `json:"is_nominative"`
	IsPaymentProgram  bool               `json:"is_payment_program"`
	SaleOK            bool               `json:"sale_ok"`
	CouponCount       int                `json:"coupon_count,omitempty"`
	TotalOrderCount   int                `json:"total_order_count,omitempty"`
	Rules             []LoyaltyRule      `json:"rules,omitempty"`
	Rewards           []LoyaltyReward    `json:"rewards,omitempty"`
	Mails             []LoyaltyMail      `json:"mails,omitempty"`
	Audit             audit.Fields       `json:"audit"`
}

// Validate verifies program invariants.
func (p *LoyaltyProgram) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return platformerrors.Validation("program name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if p.ProgramType == "" {
		p.ProgramType = ProgramTypePromotion
	}
	if !isValidProgramType(p.ProgramType) {
		return platformerrors.Validation("invalid program type", map[string]string{
			"program_type": fmt.Sprintf("must be one of %v", ProgramTypes),
		})
	}
	if p.AppliesOn == "" {
		p.AppliesOn = AppliesOnCurrent
	}
	if p.AppliesOn != AppliesOnCurrent && p.AppliesOn != AppliesOnFuture && p.AppliesOn != AppliesOnBoth {
		return platformerrors.Validation("invalid applies_on value", map[string]string{
			"applies_on": "must be current, future, or both",
		})
	}
	if p.Trigger == "" {
		p.Trigger = TriggerAuto
	}
	if p.Trigger != TriggerAuto && p.Trigger != TriggerWithCode {
		return platformerrors.Validation("invalid trigger value", map[string]string{
			"trigger": "must be auto or with_code",
		})
	}
	if p.DateFrom != nil && p.DateTo != nil && p.DateTo.Before(*p.DateFrom) {
		return platformerrors.Validation("invalid validity period", map[string]string{
			"date_to": "cannot be before date_from",
		})
	}
	if p.LimitUsage && p.MaxUsage <= 0 {
		return platformerrors.Validation("max usage must be strictly positive when limited", map[string]string{
			"max_usage": "must be > 0 when limit_usage is true",
		})
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if len(p.Rewards) == 0 {
		return platformerrors.Validation("a program must have at least one reward", map[string]string{
			"rewards": "at least one reward is required",
		})
	}
	if p.PortalPointName == "" {
		p.PortalPointName = "Points"
	}
	if !p.Active {
		p.Active = true
	}
	return nil
}

func isValidProgramType(t ProgramType) bool {
	for _, candidate := range ProgramTypes {
		if t == candidate {
			return true
		}
	}
	return false
}

// ApplyProgramTypeDefaults resets the type-driven fields and the child rules,
// rewards and mails to the Odoo `_program_type_default_values` per program type.
// It preserves the user-provided Name, Company, currency and pricelists.
func (p *LoyaltyProgram) ApplyProgramTypeDefaults(allowDefaults bool) {
	switch p.ProgramType {
	case ProgramTypePromotion:
		p.AppliesOn = AppliesOnCurrent
		p.Trigger = TriggerAuto
		p.PortalVisible = false
		p.PortalPointName = "Promo point(s)"
		p.Rules = []LoyaltyRule{{
			RewardPointAmount:   1,
			RewardPointMode:     RewardPointModeOrder,
			MinimumAmount:       50,
			MinimumQty:          0,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:   RewardTypeDiscount,
			DiscountMode: DiscountModePercent,
			Discount:     10,
			RequiredPoints: 1,
		}}
		p.Mails = nil
	case ProgramTypeCoupons:
		p.AppliesOn = AppliesOnCurrent
		p.Trigger = TriggerWithCode
		p.PortalVisible = false
		p.PortalPointName = "Coupon point(s)"
		p.Rules = nil
		p.Rewards = []LoyaltyReward{{
			RewardType:   RewardTypeDiscount,
			DiscountMode: DiscountModePercent,
			Discount:     10,
			RequiredPoints: 1,
		}}
		p.Mails = nil
	case ProgramTypeLoyalty:
		p.AppliesOn = AppliesOnBoth
		p.Trigger = TriggerAuto
		p.PortalVisible = true
		p.PortalPointName = "Loyalty point(s)"
		p.Rules = []LoyaltyRule{{
			RewardPointAmount:   1,
			RewardPointMode:     RewardPointModeMoney,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:   RewardTypeDiscount,
			DiscountMode: DiscountModePercent,
			Discount:     5,
			RequiredPoints: 200,
		}}
		p.Mails = nil
	case ProgramTypeEWallet:
		p.AppliesOn = AppliesOnFuture
		p.Trigger = TriggerAuto
		p.PortalVisible = true
		p.PortalPointName = p.Currency
		if p.PortalPointName == "" {
			p.PortalPointName = "USD"
		}
		p.Rules = []LoyaltyRule{{
			RewardPointAmount:   1,
			RewardPointMode:     RewardPointModeMoney,
			RewardPointSplit:    false,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:          RewardTypeDiscount,
			DiscountMode:        DiscountModePerPoint,
			Discount:            1,
			DiscountApplicability: DiscountApplicabilityOrder,
			RequiredPoints:      1,
			Description:         "eWallet",
		}}
		p.Mails = nil
	case ProgramTypeGiftCard:
		p.AppliesOn = AppliesOnFuture
		p.Trigger = TriggerAuto
		p.PortalVisible = true
		p.PortalPointName = p.Currency
		if p.PortalPointName == "" {
			p.PortalPointName = "USD"
		}
		p.Rules = []LoyaltyRule{{
			RewardPointAmount:   1,
			RewardPointMode:     RewardPointModeMoney,
			RewardPointSplit:    true,
			MinimumQty:          0,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:          RewardTypeDiscount,
			DiscountMode:        DiscountModePerPoint,
			Discount:            1,
			DiscountApplicability: DiscountApplicabilityOrder,
			RequiredPoints:      1,
			Description:         "Gift Card",
		}}
		p.Mails = nil
	case ProgramTypePromoCode:
		p.AppliesOn = AppliesOnCurrent
		p.Trigger = TriggerWithCode
		p.PortalVisible = false
		p.PortalPointName = "Discount point(s)"
		p.Rules = []LoyaltyRule{{
			Mode:         RuleModeWithCode,
			Code:         "PROMO_CODE_" + randomShort(),
			RewardPointMode: RewardPointModeOrder,
			RewardPointAmount: 1,
			MinimumQty:    0,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:          RewardTypeDiscount,
			DiscountMode:        DiscountModePercent,
			Discount:            10,
			DiscountApplicability: DiscountApplicabilitySpecific,
		}}
		p.Mails = nil
	case ProgramTypeBuyXGetY:
		p.AppliesOn = AppliesOnCurrent
		p.Trigger = TriggerAuto
		p.PortalVisible = false
		p.PortalPointName = "Credit(s)"
		p.Rules = []LoyaltyRule{{
			RewardPointMode:   RewardPointModeUnit,
			RewardPointAmount: 1,
			MinimumQty:        2,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:   RewardTypeProduct,
			RewardProductQty: 1,
			RequiredPoints: 2,
		}}
		p.Mails = nil
	case ProgramTypeNextOrderCoupons:
		p.AppliesOn = AppliesOnFuture
		p.Trigger = TriggerAuto
		p.PortalVisible = true
		p.PortalPointName = "Coupon point(s)"
		p.Rules = []LoyaltyRule{{
			RewardPointAmount:   1,
			RewardPointMode:     RewardPointModeOrder,
			MinimumAmount:       100,
			MinimumQty:          0,
		}}
		p.Rewards = []LoyaltyReward{{
			RewardType:          RewardTypeDiscount,
			DiscountMode:        DiscountModePercent,
			Discount:            15,
			DiscountApplicability: DiscountApplicabilityOrder,
			RequiredPoints:      1,
		}}
		p.Mails = nil
	default:
		if allowDefaults {
			p.AppliesOn = AppliesOnCurrent
			p.Trigger = TriggerAuto
			p.Rewards = []LoyaltyReward{{
				RewardType:   RewardTypeDiscount,
				DiscountMode: DiscountModePercent,
				Discount:     10,
				RequiredPoints: 1,
			}}
		}
	}

	p.IsNominative = p.AppliesOn == AppliesOnBoth ||
		((p.ProgramType == ProgramTypeEWallet || p.ProgramType == ProgramTypeLoyalty) && p.AppliesOn == AppliesOnFuture)
	p.IsPaymentProgram = p.ProgramType == ProgramTypeGiftCard || p.ProgramType == ProgramTypeEWallet
	if !p.SaleOK {
		p.SaleOK = true
	}
}

// ApplicableToOrder checks the program's active and validity window plus its
// pricelist membership for a given date and pricelist ID.
func (p *LoyaltyProgram) ApplicableToOrder(on time.Time, pricelistID *int64) bool {
	if !p.Active || !p.SaleOK {
		return false
	}
	if p.DateFrom != nil && on.Before(*p.DateFrom) {
		return false
	}
	if p.DateTo != nil && on.After(*p.DateTo) {
		return false
	}
	if len(p.PricelistIDs) > 0 {
		if pricelistID == nil {
			return false
		}
		matched := false
		for _, id := range p.PricelistIDs {
			if id == *pricelistID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// LoyaltyRule is a conditional earning rule (loyalty.rule).
type LoyaltyRule struct {
	ID                   int64            `json:"id"`
	Active               bool             `json:"active"`
	ProgramID            int64            `json:"program_id"`
	ProgramType          ProgramType      `json:"program_type,omitempty"`
	CompanyID            *int64           `json:"company_id,omitempty"`
	ProductIDs           []int64          `json:"product_ids,omitempty"`
	ProductCategoryID    *int64           `json:"product_category_id,omitempty"`
	ProductTagID         *int64           `json:"product_tag_id,omitempty"`
	ProductDomain        string           `json:"product_domain,omitempty"`
	RewardPointAmount    float64          `json:"reward_point_amount"`
	RewardPointSplit     bool             `json:"reward_point_split"`
	RewardPointMode      RewardPointMode  `json:"reward_point_mode"`
	MinimumQty           int              `json:"minimum_qty"`
	MinimumAmount        float64          `json:"minimum_amount"`
	MinimumAmountTaxMode TaxMode          `json:"minimum_amount_tax_mode"`
	Mode                 RuleMode         `json:"mode"`
	Code                 string           `json:"code,omitempty"`
	Audit                audit.Fields     `json:"audit,omitempty"`
}

// Validate verifies rule invariants.
func (r *LoyaltyRule) Validate() error {
	if r.ProgramID <= 0 {
		return platformerrors.Validation("program is required for rule", map[string]string{
			"program_id": "must reference a valid program",
		})
	}
	if r.RewardPointMode == "" {
		r.RewardPointMode = RewardPointModeOrder
	}
	switch r.RewardPointMode {
	case RewardPointModeOrder, RewardPointModeMoney, RewardPointModeUnit:
	default:
		return platformerrors.Validation("invalid reward_point_mode", map[string]string{
			"reward_point_mode": "must be order, money, or unit",
		})
	}
	if r.MinimumAmountTaxMode == "" {
		r.MinimumAmountTaxMode = TaxModeIncl
	}
	if r.MinimumAmountTaxMode != TaxModeIncl && r.MinimumAmountTaxMode != TaxModeExcl {
		return platformerrors.Validation("invalid minimum_amount_tax_mode", map[string]string{
			"minimum_amount_tax_mode": "must be incl or excl",
		})
	}
	if r.RewardPointAmount < 0 {
		return platformerrors.Validation("reward point amount cannot be negative", map[string]string{
			"reward_point_amount": "must be >= 0",
		})
	}
	if r.MinimumQty < 0 {
		r.MinimumQty = 0
	}
	if r.MinimumAmount < 0 {
		return platformerrors.Validation("minimum amount cannot be negative", map[string]string{
			"minimum_amount": "must be >= 0",
		})
	}
	if !r.Active {
		r.Active = true
	}
	return nil
}

// ResolveMode derives the rule application mode from its code (Odoo
// `_compute_mode`): a non-empty code implies with_code.
func (r *LoyaltyRule) ResolveMode() {
	if strings.TrimSpace(r.Code) != "" {
		r.Mode = RuleModeWithCode
	} else {
		r.Mode = RuleModeAuto
	}
}

// MatchesProduct determines whether the rule applies to a product. categoryIDs
// must contain the category ID and all of its descendant category IDs.
func (r *LoyaltyRule) MatchesProduct(productID int64, categoryIDs map[int64]bool) bool {
	if len(r.ProductIDs) == 0 && r.ProductCategoryID == nil && r.ProductTagID == nil {
		return true // unrestricted (empty domain)
	}
	for _, id := range r.ProductIDs {
		if id == productID {
			return true
		}
	}
	if r.ProductCategoryID != nil && categoryIDs[*r.ProductCategoryID] {
		return true
	}
	// product.tag is not modeled in this backend; tag constraints are stored
	// but not evaluated.
	return false
}

// LoyaltyReward is a redeemable reward (loyalty.reward).
type LoyaltyReward struct {
	ID                          int64                  `json:"id"`
	Active                      bool                   `json:"active"`
	ProgramID                   int64                  `json:"program_id"`
	ProgramType                 ProgramType            `json:"program_type,omitempty"`
	Description                 string                 `json:"description"`
	RewardType                  RewardType             `json:"reward_type"`
	Discount                    float64                `json:"discount"`
	DiscountMode                DiscountMode           `json:"discount_mode"`
	DiscountApplicability       DiscountApplicability  `json:"discount_applicability"`
	DiscountProductIDs          []int64                `json:"discount_product_ids,omitempty"`
	DiscountProductCategoryID   *int64                 `json:"discount_product_category_id,omitempty"`
	DiscountProductTagID        *int64                 `json:"discount_product_tag_id,omitempty"`
	DiscountMaxAmount           float64                `json:"discount_max_amount"`
	DiscountLineProductID       *int64                 `json:"discount_line_product_id,omitempty"`
	RewardProductID             *int64                 `json:"reward_product_id,omitempty"`
	RewardProductQty            int                    `json:"reward_product_qty"`
	RewardProductUomID          *int64                 `json:"reward_product_uom_id,omitempty"`
	RequiredPoints              float64                `json:"required_points"`
	PointName                   string                 `json:"point_name,omitempty"`
	ClearWallet                 bool                   `json:"clear_wallet"`
	ProductDomain               string                 `json:"product_domain,omitempty"`
	Audit                       audit.Fields           `json:"audit,omitempty"`
}

// Validate verifies reward invariants.
func (r *LoyaltyReward) Validate() error {
	if r.ProgramID <= 0 {
		return platformerrors.Validation("program is required for reward", map[string]string{
			"program_id": "must reference a valid program",
		})
	}
	if r.RewardType == "" {
		r.RewardType = RewardTypeDiscount
	}
	if r.RewardType != RewardTypeDiscount && r.RewardType != RewardTypeProduct {
		return platformerrors.Validation("invalid reward_type", map[string]string{
			"reward_type": "must be discount or product",
		})
	}
	if r.DiscountMode == "" {
		r.DiscountMode = DiscountModePercent
	}
	switch r.DiscountMode {
	case DiscountModePercent, DiscountModePerOrder, DiscountModePerPoint:
	default:
		return platformerrors.Validation("invalid discount_mode", map[string]string{
			"discount_mode": "must be percent, per_order, or per_point",
		})
	}
	if r.DiscountApplicability == "" {
		r.DiscountApplicability = DiscountApplicabilityOrder
	}
	switch r.DiscountApplicability {
	case DiscountApplicabilityOrder, DiscountApplicabilityCheapest, DiscountApplicabilitySpecific:
	default:
		return platformerrors.Validation("invalid discount_applicability", map[string]string{
			"discount_applicability": "must be order, cheapest, or specific",
		})
	}
	if r.RequiredPoints <= 0 {
		return platformerrors.Validation("required points must be strictly positive", map[string]string{
			"required_points": "must be > 0",
		})
	}
	if r.RewardType == RewardTypeProduct {
		if r.RewardProductID == nil || *r.RewardProductID <= 0 {
			return platformerrors.Validation("reward product is required for product rewards", map[string]string{
				"reward_product_id": "must reference a valid product",
			})
		}
		if r.RewardProductQty <= 0 {
			return platformerrors.Validation("reward product quantity must be strictly positive", map[string]string{
				"reward_product_qty": "must be > 0",
			})
		}
	} else if r.Discount <= 0 {
		return platformerrors.Validation("discount must be strictly positive", map[string]string{
			"discount": "must be > 0",
		})
	}
	if r.DiscountMaxAmount < 0 {
		r.DiscountMaxAmount = 0
	}
	if !r.Active {
		r.Active = true
	}
	return nil
}

// DiscountAmount computes the monetary value and the point cost of this reward
// for the given available points, points to use and eligible base amount.
// It mirrors the Odoo discount reward computation with mode-specific logic.
func (r *LoyaltyReward) DiscountAmount(availablePoints, pointsToUse, eligibleAmount float64) (discountAmount, pointsCost float64) {
	if r.RewardType == RewardTypeProduct {
		return 0, r.RequiredPoints
	}

	maxAmount := r.DiscountMaxAmount
	if maxAmount < 0 {
		maxAmount = 0
	}

	switch r.DiscountMode {
	case DiscountModePercent:
		pointsCost = r.RequiredPoints
		discountAmount = eligibleAmount * r.Discount / 100.0
	case DiscountModePerOrder:
		pointsCost = r.RequiredPoints
		discountAmount = r.Discount
	case DiscountModePerPoint:
		if pointsToUse <= 0 || availablePoints < r.RequiredPoints {
			return 0, 0
		}
		if pointsToUse > availablePoints {
			pointsToUse = availablePoints
		}
		pointsCost = pointsToUse
		discountAmount = pointsToUse * r.Discount
	default:
		return 0, 0
	}

	if maxAmount > 0 && discountAmount > maxAmount {
		discountAmount = maxAmount
		if r.DiscountMode == DiscountModePerPoint && r.Discount > 0 {
			pointsCost = discountAmount / r.Discount
		}
	}
	return round4(discountAmount), round4(pointsCost)
}

// LoyaltyCard is a coupon, gift card, eWallet or loyalty card (loyalty.card).
type LoyaltyCard struct {
	ID             int64          `json:"id"`
	ProgramID      int64          `json:"program_id"`
	ProgramType    ProgramType    `json:"program_type,omitempty"`
	CompanyID      *int64         `json:"company_id,omitempty"`
	PartnerID      *int64         `json:"partner_id,omitempty"`
	Points         float64        `json:"points"`
	Code           string         `json:"code"`
	ExpirationDate *time.Time     `json:"expiration_date,omitempty"`
	UseCount       int            `json:"use_count"`
	OrderID        *int64         `json:"order_id,omitempty"`
	Active         bool           `json:"active"`
	Audit          audit.Fields   `json:"audit"`
}

// Validate verifies card invariants.
func (c *LoyaltyCard) Validate() error {
	if c.ProgramID <= 0 {
		return platformerrors.Validation("program is required for card", map[string]string{
			"program_id": "must reference a valid program",
		})
	}
	c.Code = strings.TrimSpace(c.Code)
	if c.Code == "" {
		c.Code = GenerateCode()
	}
	if c.Points < 0 {
		return platformerrors.Validation("points cannot be negative", map[string]string{
			"points": "must be >= 0",
		})
	}
	if !c.Active {
		c.Active = true
	}
	return nil
}

// GenerateCode produces a short unique coupon code (barcode-like).
func GenerateCode() string {
	return "044" + randomHex(8)
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func randomShort() string {
	return strings.ToUpper(randomHex(2))
}

// LoyaltyHistory records a balance movement on a card (loyalty.history).
type LoyaltyHistory struct {
	ID          int64      `json:"id"`
	CardID      int64      `json:"card_id"`
	CompanyID   *int64     `json:"company_id,omitempty"`
	Description string     `json:"description"`
	Issued      float64    `json:"issued"`
	Used        float64    `json:"used"`
	OrderModel  string     `json:"order_model,omitempty"`
	OrderID     *int64     `json:"order_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// LoyaltyMail is configuration-only communication plan (loyalty.mail).
type LoyaltyMail struct {
	ID        int64        `json:"id"`
	Active    bool         `json:"active"`
	ProgramID int64        `json:"program_id"`
	Trigger   MailTrigger  `json:"trigger"`
	Points    float64      `json:"points"`
	Audit     audit.Fields `json:"audit,omitempty"`
}

// Validate verifies mail config invariants.
func (m *LoyaltyMail) Validate() error {
	if m.ProgramID <= 0 {
		return platformerrors.Validation("program is required for mail", map[string]string{
			"program_id": "must reference a valid program",
		})
	}
	if m.Trigger == "" {
		m.Trigger = MailTriggerCreate
	}
	if m.Trigger != MailTriggerCreate && m.Trigger != MailTriggerPointsReach {
		return platformerrors.Validation("invalid trigger", map[string]string{
			"trigger": "must be create or points_reach",
		})
	}
	if m.Trigger == MailTriggerPointsReach && m.Points < 0 {
		return platformerrors.Validation("points cannot be negative", map[string]string{
			"points": "must be >= 0",
		})
	}
	if !m.Active {
		m.Active = true
	}
	return nil
}

// OrderCouponPoints tracks the impact of an order on a coupon (sale.order.coupon.points).
type OrderCouponPoints struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	CouponID  int64     `json:"coupon_id"`
	Points    float64   `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}