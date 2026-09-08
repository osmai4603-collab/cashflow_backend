package loyaltyusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// UseCase implements the loyalty application logic (Odoo `loyalty` + `sale_loyalty`).
type UseCase struct {
	repo        loyalty.Repository
	productRepo product.Repository
	saleRepo    sale.Repository
	logger      *slog.Logger
}

// New creates the loyalty usecase. saleRepo and productRepo may be nil in tests.
func New(repo loyalty.Repository, productRepo product.Repository, saleRepo sale.Repository, logger *slog.Logger) *UseCase {
	return &UseCase{
		repo:        repo,
		productRepo: productRepo,
		saleRepo:    saleRepo,
		logger:      logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Programs
// ─────────────────────────────────────────────────────────────────────────────

type CreateProgramInput struct {
	Name           string
	Active         bool
	Sequence       int
	CompanyID      *int64
	Currency       string
	PricelistIDs   []int64
	ProgramType    loyalty.ProgramType
	DateFrom       *time.Time
	DateTo         *time.Time
	LimitUsage     bool
	MaxUsage       int
	AppliesOn      loyalty.AppliesOn
	Trigger        loyalty.Trigger
	PortalVisible  bool
	PortalPointName string
	// Children
	Rules   []loyalty.LoyaltyRule
	Rewards []loyalty.LoyaltyReward
	Mails   []loyalty.LoyaltyMail
}

func (uc *UseCase) CreateProgram(ctx context.Context, in CreateProgramInput) (*loyalty.LoyaltyProgram, error) {
	program := &loyalty.LoyaltyProgram{
		Name:            in.Name,
		Active:          true,
		Sequence:        in.Sequence,
		CompanyID:       in.CompanyID,
		Currency:        in.Currency,
		PricelistIDs:    in.PricelistIDs,
		ProgramType:     in.ProgramType,
		DateFrom:        in.DateFrom,
		DateTo:          in.DateTo,
		LimitUsage:      in.LimitUsage,
		MaxUsage:        in.MaxUsage,
		AppliesOn:       in.AppliesOn,
		Trigger:         in.Trigger,
		PortalVisible:   in.PortalVisible,
		PortalPointName: in.PortalPointName,
		Rules:           in.Rules,
		Rewards:         in.Rewards,
		Mails:           in.Mails,
		SaleOK:          true,
	}

	// Seeds Odoo's per-type defaults for rules/rewards and derived flags.
	program.ApplyProgramTypeDefaults(true)
	for i := range program.Rules {
		program.Rules[i].ProgramID = 1 // resolved to program.ID on persistence
		program.Rules[i].ResolveMode()
	}
	for i := range program.Rewards {
		program.Rewards[i].ProgramID = 1
	}
	for i := range program.Mails {
		program.Mails[i].ProgramID = 1
	}

	if len(in.Rewards) > 0 {
		// Explicit rewards were supplied: keep them instead of the type defaults.
		program.Rewards = in.Rewards
	}
	if len(in.Rules) > 0 {
		program.Rules = in.Rules
	}

	if err := program.Validate(); err != nil {
		return nil, err
	}
	for i := range program.Rules {
		if err := program.Rules[i].Validate(); err != nil {
			return nil, err
		}
	}
	for i := range program.Rewards {
		if err := program.Rewards[i].Validate(); err != nil {
			return nil, err
		}
	}
	for i := range program.Mails {
		if err := program.Mails[i].Validate(); err != nil {
			return nil, err
		}
	}

	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		program.CompanyID = companyID
	}

	if err := uc.repo.CreateProgram(ctx, program); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "loyalty program created", "id", program.ID, "name", program.Name, "program_type", program.ProgramType)
	return program, nil
}

type UpdateProgramInput struct {
	Name            *string
	Active          *bool
	Sequence        *int
	Currency        *string
	PricelistIDs    []int64
	DateFrom        *time.Time
	DateTo          *time.Time
	LimitUsage      *bool
	MaxUsage        *int
	AppliesOn       *loyalty.AppliesOn
	Trigger         *loyalty.Trigger
	PortalVisible   *bool
	PortalPointName *string
	Rules           []loyalty.LoyaltyRule
	Rewards         []loyalty.LoyaltyReward
	Mails           []loyalty.LoyaltyMail
}

func (uc *UseCase) UpdateProgram(ctx context.Context, id int64, in UpdateProgramInput) (*loyalty.LoyaltyProgram, error) {
	program, err := uc.repo.GetProgramByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		program.Name = *in.Name
	}
	if in.Active != nil {
		program.Active = *in.Active
	}
	if in.Sequence != nil {
		program.Sequence = *in.Sequence
	}
	if in.Currency != nil {
		program.Currency = *in.Currency
	}
	if in.DateFrom != nil || in.DateTo != nil {
		program.DateFrom = in.DateFrom
		program.DateTo = in.DateTo
	}
	if in.LimitUsage != nil {
		program.LimitUsage = *in.LimitUsage
	}
	if in.MaxUsage != nil {
		program.MaxUsage = *in.MaxUsage
	}
	if in.AppliesOn != nil {
		program.AppliesOn = *in.AppliesOn
		if program.ProgramType == loyalty.ProgramTypeCoupons {
			program.AppliesOn = loyalty.AppliesOnCurrent
		}
	}
	if in.Trigger != nil {
		program.Trigger = *in.Trigger
	}
	if in.PortalVisible != nil {
		program.PortalVisible = *in.PortalVisible
	}
	if in.PortalPointName != nil {
		program.PortalPointName = *in.PortalPointName
	}
	if in.PricelistIDs != nil {
		program.PricelistIDs = in.PricelistIDs
	}
	if in.Rules != nil {
		program.Rules = in.Rules
	}
	if in.Rewards != nil {
		program.Rewards = in.Rewards
	}
	if in.Mails != nil {
		program.Mails = in.Mails
	}

	if err := program.Validate(); err != nil {
		return nil, err
	}
	for i := range program.Rules {
		if err := program.Rules[i].Validate(); err != nil {
			return nil, err
		}
	}
	for i := range program.Rewards {
		if err := program.Rewards[i].Validate(); err != nil {
			return nil, err
		}
	}
	for i := range program.Mails {
		if err := program.Mails[i].Validate(); err != nil {
			return nil, err
		}
	}

	if err := uc.repo.UpdateProgram(ctx, program); err != nil {
		return nil, err
	}
	return program, nil
}

func (uc *UseCase) GetProgram(ctx context.Context, id int64) (*loyalty.LoyaltyProgram, error) {
	return uc.repo.GetProgramByID(ctx, id)
}

func (uc *UseCase) ListPrograms(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[loyalty.LoyaltyProgram], error) {
	return uc.repo.ListPrograms(ctx, f, page)
}

func (uc *UseCase) DeleteProgram(ctx context.Context, id int64) error {
	return uc.repo.DeleteProgram(ctx, id)
}

// SetProgramType re-applies the type-driven defaults (rules, rewards, flags) to
// an existing program.
func (uc *UseCase) SetProgramType(ctx context.Context, id int64, programType loyalty.ProgramType) (*loyalty.LoyaltyProgram, error) {
	program, err := uc.repo.GetProgramByID(ctx, id)
	if err != nil {
		return nil, err
	}
	program.ProgramType = programType
	program.ApplyProgramTypeDefaults(true)
	for i := range program.Rules {
		program.Rules[i].ResolveMode()
	}
	if err := program.Validate(); err != nil {
		return nil, err
	}
	for i := range program.Rewards {
		if err := program.Rewards[i].Validate(); err != nil {
			return nil, err
		}
	}
	if err := uc.repo.UpdateProgram(ctx, program); err != nil {
		return nil, err
	}
	return program, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Child records: rules, rewards, mails
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateRule(ctx context.Context, programID int64, rule *loyalty.LoyaltyRule) (*loyalty.LoyaltyRule, error) {
	program, err := uc.repo.GetProgramByID(ctx, programID)
	if err != nil {
		return nil, err
	}
	rule.ProgramID = program.ID
	rule.ProgramType = program.ProgramType
	rule.ResolveMode()
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (uc *UseCase) UpdateRule(ctx context.Context, id int64, rule *loyalty.LoyaltyRule) (*loyalty.LoyaltyRule, error) {
	rule.ID = id
	rule.ResolveMode()
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateRule(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (uc *UseCase) DeleteRule(ctx context.Context, id int64) error {
	return uc.repo.DeleteRule(ctx, id)
}

func (uc *UseCase) CreateReward(ctx context.Context, programID int64, reward *loyalty.LoyaltyReward) (*loyalty.LoyaltyReward, error) {
	program, err := uc.repo.GetProgramByID(ctx, programID)
	if err != nil {
		return nil, err
	}
	reward.ProgramID = program.ID
	reward.ProgramType = program.ProgramType
	if err := reward.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateReward(ctx, reward); err != nil {
		return nil, err
	}
	return reward, nil
}

func (uc *UseCase) UpdateReward(ctx context.Context, id int64, reward *loyalty.LoyaltyReward) (*loyalty.LoyaltyReward, error) {
	reward.ID = id
	if err := reward.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateReward(ctx, reward); err != nil {
		return nil, err
	}
	return reward, nil
}

func (uc *UseCase) DeleteReward(ctx context.Context, id int64) error {
	return uc.repo.DeleteReward(ctx, id)
}

func (uc *UseCase) CreateMail(ctx context.Context, programID int64, mail *loyalty.LoyaltyMail) (*loyalty.LoyaltyMail, error) {
	program, err := uc.repo.GetProgramByID(ctx, programID)
	if err != nil {
		return nil, err
	}
	mail.ProgramID = program.ID
	if err := mail.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateMail(ctx, mail); err != nil {
		return nil, err
	}
	return mail, nil
}

func (uc *UseCase) UpdateMail(ctx context.Context, id int64, mail *loyalty.LoyaltyMail) (*loyalty.LoyaltyMail, error) {
	mail.ID = id
	if err := mail.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateMail(ctx, mail); err != nil {
		return nil, err
	}
	return mail, nil
}

func (uc *UseCase) DeleteMail(ctx context.Context, id int64) error {
	return uc.repo.DeleteMail(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cards
// ─────────────────────────────────────────────────────────────────────────────

type GenerateCardsInput struct {
	ProgramID      int64
	PartnerID      *int64
	ExpirationDate *time.Time
	Count          int
}

func (uc *UseCase) GenerateCards(ctx context.Context, in GenerateCardsInput) ([]loyalty.LoyaltyCard, error) {
	if in.Count <= 0 {
		in.Count = 1
	}
	if in.Count > 100 {
		in.Count = 100
	}
	program, err := uc.repo.GetProgramByID(ctx, in.ProgramID)
	if err != nil {
		return nil, err
	}
	if !program.Active {
		return nil, platformerrors.Conflict("cannot generate cards for an inactive program")
	}

	cards := make([]loyalty.LoyaltyCard, 0, in.Count)
	for i := 0; i < in.Count; i++ {
		card := &loyalty.LoyaltyCard{
			ProgramID:      program.ID,
			CompanyID:      program.CompanyID,
			PartnerID:      in.PartnerID,
			Points:         0,
			Code:           loyalty.GenerateCode(),
			ExpirationDate: in.ExpirationDate,
			Active:         true,
		}
		if err := card.Validate(); err != nil {
			return nil, err
		}
		if err := uc.repo.CreateCard(ctx, card); err != nil {
			return nil, err
		}
		cards = append(cards, *card)
	}
	return cards, nil
}

func (uc *UseCase) GetCard(ctx context.Context, id int64) (*loyalty.LoyaltyCard, error) {
	return uc.repo.GetCardByID(ctx, id)
}

func (uc *UseCase) GetCardByCode(ctx context.Context, code string) (*loyalty.LoyaltyCard, error) {
	return uc.repo.GetCardByCode(ctx, code)
}

func (uc *UseCase) ListCards(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[loyalty.LoyaltyCard], error) {
	return uc.repo.ListCards(ctx, f, page)
}

func (uc *UseCase) ArchiveCard(ctx context.Context, id int64) (*loyalty.LoyaltyCard, error) {
	card, err := uc.repo.GetCardByID(ctx, id)
	if err != nil {
		return nil, err
	}
	card.Active = false
	if err := uc.repo.UpdateCard(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (uc *UseCase) ListCardHistory(ctx context.Context, cardID int64) ([]loyalty.LoyaltyHistory, error) {
	if _, err := uc.repo.GetCardByID(ctx, cardID); err != nil {
		return nil, err
	}
	return uc.repo.ListHistoryByCard(ctx, cardID)
}

// ─────────────────────────────────────────────────────────────────────────────
// Order preview
// ─────────────────────────────────────────────────────────────────────────────

// CouponPreview describes an already-applied coupon on an order.
type CouponPreview struct {
	CouponID     int64            `json:"coupon_id"`
	Code         string           `json:"code"`
	ProgramID    int64            `json:"program_id"`
	ProgramName  string           `json:"program_name"`
	ProgramType  loyalty.ProgramType `json:"program_type"`
	AppliesOn    loyalty.AppliesOn   `json:"applies_on"`
	CardPoints   float64          `json:"card_points"`
	Projected    float64          `json:"projected"`
	Consumed     float64          `json:"consumed"`
	Available    float64          `json:"available"`
	Rewards      []loyalty.LoyaltyReward `json:"rewards"`
}

// OrderPreview aggregates the loyalty state of an order.
type OrderPreview struct {
	OrderID        int64                  `json:"order_id"`
	ProgramEarns   []loyalty.ProgramEarn  `json:"program_earns"`
	AppliedCoupons []CouponPreview        `json:"applied_coupons"`
}

func (uc *UseCase) PreviewOrder(ctx context.Context, orderID int64) (*OrderPreview, error) {
	if uc.saleRepo == nil {
		return nil, platformerrors.Internal("sale repository is not configured", nil)
	}
	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	programs, err := uc.repo.ListPrograms(ctx, filter.NewFilter(filter.Criterion{Field: "active", Operator: filter.OpEqual, Value: true}), pagination.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}

	preview := &OrderPreview{OrderID: order.ID, AppliedCoupons: []CouponPreview{}}
	for _, program := range programs.Items {
		points, eligible := func() (float64, bool) {
			lines, err := uc.earnLinesFromOrder(ctx, order)
			if err != nil {
				return 0, false
			}
			resolver, err := uc.buildCategoryResolver(ctx)
			if err != nil {
				return 0, false
			}
			_ = resolver.categorize(ctx, uc.productRepo, order)
			pts, ok := loyalty.ComputeProgramPoints(&program, lines, uc.enabledRuleIDFor(order), resolver.matchesProduct)
			return pts, ok
		}()
		if eligible {
			preview.ProgramEarns = append(preview.ProgramEarns, loyalty.ProgramEarn{
				ProgramID:   program.ID,
				ProgramName: program.Name,
				ProgramType: program.ProgramType,
				Earned:      points,
				Eligible:    true,
			})
		}
	}

	for _, couponID := range order.AppliedCouponIDs {
		card, err := uc.repo.GetCardByID(ctx, couponID)
		if err != nil {
			continue
		}
		program, err := uc.repo.GetProgramByID(ctx, card.ProgramID)
		if err != nil {
			continue
		}
		projected, consumed, err := uc.couponBalance(ctx, order, card, program)
		if err != nil {
			return nil, err
		}
		cp := CouponPreview{
			CouponID:    card.ID,
			Code:        card.Code,
			ProgramID:   program.ID,
			ProgramName: program.Name,
			ProgramType: program.ProgramType,
			AppliesOn:   program.AppliesOn,
			CardPoints:  card.Points,
			Projected:   projected,
			Consumed:    consumed,
			Available:   loyalty.AvailablePoints(card.Points, projected, consumed),
			Rewards:     program.Rewards,
		}
		preview.AppliedCoupons = append(preview.AppliedCoupons, cp)
	}
	return preview, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Coupon claim / removal / code enablement
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) ClaimCoupon(ctx context.Context, orderID int64, code string) (*OrderPreview, error) {
	if uc.saleRepo == nil {
		return nil, platformerrors.Internal("sale repository is not configured", nil)
	}
	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.State != sale.OrderStateDraft && order.State != sale.OrderStateSent {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot claim a coupon on an order in state '%s'; must be draft or sent", order.State))
	}

	card, err := uc.repo.GetCardByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !card.Active {
		return nil, platformerrors.Conflict("coupon is inactive")
	}
	if card.ExpirationDate != nil && time.Now().UTC().After(*card.ExpirationDate) {
		return nil, platformerrors.Conflict("coupon has expired")
	}
	program, err := uc.repo.GetProgramByID(ctx, card.ProgramID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if !program.ApplicableToOrder(now, order.PricelistID) {
		return nil, platformerrors.Conflict("program is not applicable to this order (date window or pricelist mismatch)")
	}
	if program.AppliesOn == loyalty.AppliesOnFuture {
		return nil, platformerrors.Conflict("this coupon can only be used on future orders (earned, not yet redeemable here)")
	}
	if program.LimitUsage && card.UseCount >= program.MaxUsage {
		return nil, platformerrors.Conflict("coupon usage limit reached")
	}
	for _, existing := range order.AppliedCouponIDs {
		if existing == card.ID {
			return nil, platformerrors.Conflict("coupon is already applied to this order")
		}
	}

	// Enforce code-enabled rules for with_code programs.
	if program.Trigger == loyalty.TriggerWithCode && len(program.Rules) > 0 {
		hasEnabled := false
		for _, rule := range program.Rules {
			if order.HasCodeEnabled(rule.ID) {
				hasEnabled = true
				break
			}
		}
		if !hasEnabled {
			return nil, platformerrors.Conflict("this program requires enabling its discount code on the order first")
		}
	}

	order.AppliedCouponIDs = append(order.AppliedCouponIDs, card.ID)

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
	uc.logger.InfoContext(ctx, "loyalty coupon claimed", "order_id", order.ID, "code", code)
	return uc.PreviewOrder(ctx, order.ID)
}

// ApplyCode enables a with_code rule on an order.
func (uc *UseCase) ApplyCode(ctx context.Context, orderID int64, ruleID int64) (*OrderPreview, error) {
	if uc.saleRepo == nil {
		return nil, platformerrors.Internal("sale repository is not configured", nil)
	}
	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	order.AddCodeEnabledRule(ruleID)
	if err := uc.saleRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	return uc.PreviewOrder(ctx, order.ID)
}

// RemoveCoupon removes a coupon and its reward lines from an order.
func (uc *UseCase) RemoveCoupon(ctx context.Context, orderID int64, couponID int64) (*OrderPreview, error) {
	if uc.saleRepo == nil {
		return nil, platformerrors.Internal("sale repository is not configured", nil)
	}
	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	kept := order.AppliedCouponIDs[:0]
	for _, id := range order.AppliedCouponIDs {
		if id != couponID {
			kept = append(kept, id)
		}
	}
	order.AppliedCouponIDs = kept

	var lines []sale.SaleOrderLine
	for _, l := range order.Lines {
		if l.CouponID != nil && *l.CouponID == couponID {
			continue
		}
		lines = append(lines, l)
	}
	order.Lines = lines
	order.RecomputeTotals()

	if err := uc.repo.RemoveCouponPointsByOrder(ctx, orderID); err != nil {
		return nil, err
	}
	if err := uc.saleRepo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	return uc.PreviewOrder(ctx, order.ID)
}

// EarnCoupons generates future-program cards (gift card, eWallet, next order
// coupons, loyalty with future applicability) carrying a projected-points row
// for the order. Balances are credited when the order is confirmed.
func (uc *UseCase) EarnCoupons(ctx context.Context, orderID int64) (*OrderPreview, error) {
	if uc.saleRepo == nil {
		return nil, platformerrors.Internal("sale repository is not configured", nil)
	}
	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.State != sale.OrderStateDraft && order.State != sale.OrderStateSent {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot generate coupons on an order in state '%s'; must be draft or sent", order.State))
	}

	programs, err := uc.repo.ListPrograms(ctx, filter.NewFilter(filter.Criterion{Field: "active", Operator: filter.OpEqual, Value: true}), pagination.PageRequest{Limit: 100})
	if err != nil {
		return nil, err
	}

	for _, program := range programs.Items {
		if program.AppliesOn != loyalty.AppliesOnFuture {
			continue // only future-program coupons are minted here
		}
		points, eligible := func() (float64, bool) {
			lines, err := uc.earnLinesFromOrder(ctx, order)
			if err != nil {
				return 0, false
			}
			resolver, err := uc.buildCategoryResolver(ctx)
			if err != nil {
				return 0, false
			}
			_ = resolver.categorize(ctx, uc.productRepo, order)
			return loyalty.ComputeProgramPoints(&program, lines, uc.enabledRuleIDFor(order), resolver.matchesProduct)
		}()
		if !eligible || points <= 0 {
			continue
		}

		// Avoid duplicating an already-generated card for this order.
		existing, err := uc.repo.ListCards(ctx, filter.NewFilter(
			filter.Criterion{Field: "program_id", Operator: filter.OpEqual, Value: program.ID},
			filter.Criterion{Field: "order_id", Operator: filter.OpEqual, Value: order.ID},
		), pagination.PageRequest{Limit: 1})
		if err != nil {
			return nil, err
		}
		if len(existing.Items) > 0 {
			continue
		}

		card := &loyalty.LoyaltyCard{
			ProgramID: program.ID,
			CompanyID: program.CompanyID,
			PartnerID: &order.PartnerID,
			Points:    0,
			Code:      loyalty.GenerateCode(),
			OrderID:   &order.ID,
			Active:    true,
		}
		if err := card.Validate(); err != nil {
			return nil, err
		}
		if err := uc.repo.CreateCard(ctx, card); err != nil {
			return nil, err
		}
		if err := uc.repo.UpsertCouponPoints(ctx, order.ID, card.ID, points); err != nil {
			return nil, err
		}
		uc.logger.InfoContext(ctx, "loyalty future coupon generated",
			"order_id", order.ID, "program_id", program.ID, "code", card.Code, "points", points)
	}
	return uc.PreviewOrder(ctx, order.ID)
}

// CouponInfo is the public view returned by /loyalty/check/{code}.
type CouponInfo struct {
	*loyalty.LoyaltyCard
	ProgramName string                  `json:"program_name"`
	ProgramType loyalty.ProgramType     `json:"program_type"`
	AppliesOn   loyalty.AppliesOn       `json:"applies_on"`
	Rewards     []loyalty.LoyaltyReward `json:"rewards"`
}

func (uc *UseCase) CheckCoupon(ctx context.Context, code string) (*CouponInfo, error) {
	card, err := uc.repo.GetCardByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	program, err := uc.repo.GetProgramByID(ctx, card.ProgramID)
	if err != nil {
		return nil, err
	}
	if !card.Active {
		return nil, platformerrors.Conflict("coupon is inactive")
	}
	if card.ExpirationDate != nil && nowUTC().After(*card.ExpirationDate) {
		return nil, platformerrors.Conflict("coupon has expired")
	}
	return &CouponInfo{
		LoyaltyCard: card,
		ProgramName: program.Name,
		ProgramType: program.ProgramType,
		AppliesOn:   program.AppliesOn,
		Rewards:     program.Rewards,
	}, nil
}