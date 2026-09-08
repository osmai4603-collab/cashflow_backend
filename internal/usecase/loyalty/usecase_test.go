package loyaltyusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"cashflow_backend/internal/adapters/storage/loyalty"
	"cashflow_backend/internal/adapters/storage/sale"
	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/usecase/loyalty"
)

func newTestEnv(t *testing.T) (*loyaltyusecase.UseCase, *loyaltystorage.MemoryRepo, *salestorage.MemoryRepo) {
	t.Helper()
	loyaltyRepo := loyaltystorage.NewMemoryRepo()
	saleRepo := salestorage.NewMemoryRepo()
	uc := loyaltyusecase.New(loyaltyRepo, nil, saleRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return uc, loyaltyRepo, saleRepo
}

func newSaleOrder() *sale.SaleOrder {
	return &sale.SaleOrder{
		Name:       "SO-TEST-001",
		PartnerID:  1,
		DateOrder:  time.Now().UTC(),
		State:      sale.OrderStateDraft,
		Currency:   "USD",
		AppliedCouponIDs:   []int64{},
		CodeEnabledRuleIDs: []int64{},
		Lines: []sale.SaleOrderLine{
			{Sequence: 10, ProductID: 100, Name: "Widget", ProductUomQty: 1, UnitPrice: 100, PriceSubtotal: 100, PriceTax: 0, PriceTotal: 100},
			{Sequence: 20, ProductID: 101, Name: "Gadget", ProductUomQty: 1, UnitPrice: 100, PriceSubtotal: 100, PriceTax: 0, PriceTotal: 100},
		},
	}
}

func TestCreateProgramAppliesTypeDefaults(t *testing.T) {
	ctx := context.Background()
	uc, _, _ := newTestEnv(t)

	program, err := uc.CreateProgram(ctx, loyaltyusecase.CreateProgramInput{Name: "Loyalty Test", ProgramType: loyalty.ProgramTypeLoyalty})
	if err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	if program.ID != 1 {
		t.Fatalf("expected program id 1, got %d", program.ID)
	}
	if program.AppliesOn != loyalty.AppliesOnBoth || program.Trigger != loyalty.TriggerAuto {
		t.Fatalf("unexpected defaults: applies_on=%s trigger=%s", program.AppliesOn, program.Trigger)
	}
	if len(program.Rules) != 1 || len(program.Rewards) != 1 {
		t.Fatalf("expected 1 default rule and 1 default reward, got %d rules %d rewards", len(program.Rules), len(program.Rewards))
	}
	if program.Rules[0].ID != 1 || program.Rules[0].ProgramID != program.ID {
		t.Fatalf("rule materialization mismatch: id=%d program_id=%d", program.Rules[0].ID, program.Rules[0].ProgramID)
	}
	if program.Rewards[0].ID != 1 || program.Rewards[0].ProgramID != program.ID || program.Rewards[0].RequiredPoints != 200 {
		t.Fatalf("reward materialization mismatch: id=%d program_id=%d required_points=%v", program.Rewards[0].ID, program.Rewards[0].ProgramID, program.Rewards[0].RequiredPoints)
	}

	page, err := uc.ListPrograms(ctx, nil, pagination.PageRequest{Limit: 100})
	if err != nil {
		t.Fatalf("ListPrograms: %v", err)
	}
	if page.TotalItems != 1 {
		t.Fatalf("expected 1 program, got %d", page.TotalItems)
	}
}

func TestGenerateCardsAndCheckCoupon(t *testing.T) {
	ctx := context.Background()
	uc, _, _ := newTestEnv(t)

	if _, err := uc.CreateProgram(ctx, loyaltyusecase.CreateProgramInput{Name: "Coupons Test", ProgramType: loyalty.ProgramTypeCoupons}); err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}

	cards, err := uc.GenerateCards(ctx, loyaltyusecase.GenerateCardsInput{ProgramID: 1, Count: 2})
	if err != nil {
		t.Fatalf("GenerateCards: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	if cards[0].Code == "" || cards[0].Code == cards[1].Code {
		t.Fatalf("expected unique non-empty codes, got %q and %q", cards[0].Code, cards[1].Code)
	}
	if cards[0].ProgramID != 1 {
		t.Fatalf("expected program 1 on card, got %d", cards[0].ProgramID)
	}

	info, err := uc.CheckCoupon(ctx, cards[0].Code)
	if err != nil {
		t.Fatalf("CheckCoupon: %v", err)
	}
	if info.ProgramName != "Coupons Test" || len(info.Rewards) != 1 {
		t.Fatalf("unexpected coupon info: %+v", info)
	}

	if _, err := uc.CheckCoupon(ctx, "NOPE-1234"); err == nil {
		t.Fatal("expected error for unknown code")
	}
}

func TestClaimCouponPreviewAndRejectDuplicate(t *testing.T) {
	ctx := context.Background()
	uc, _, saleRepo := newTestEnv(t)

	if _, err := uc.CreateProgram(ctx, loyaltyusecase.CreateProgramInput{Name: "Loyalty Test", ProgramType: loyalty.ProgramTypeLoyalty}); err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	cards, err := uc.GenerateCards(ctx, loyaltyusecase.GenerateCardsInput{ProgramID: 1, Count: 1})
	if err != nil {
		t.Fatalf("GenerateCards: %v", err)
	}

	order := newSaleOrder()
	if err := saleRepo.CreateOrder(ctx, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	preview, err := uc.PreviewOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("PreviewOrder: %v", err)
	}
	if len(preview.ProgramEarns) == 0 || preview.ProgramEarns[0].Earned != 200 {
		t.Fatalf("expected program earn of 200 points, got %+v", preview.ProgramEarns)
	}

	preview, err = uc.ClaimCoupon(ctx, order.ID, cards[0].Code)
	if err != nil {
		t.Fatalf("ClaimCoupon: %v", err)
	}
	if len(preview.AppliedCoupons) != 1 {
		t.Fatalf("expected 1 applied coupon, got %d", len(preview.AppliedCoupons))
	}
	cp := preview.AppliedCoupons[0]
	if cp.Code != cards[0].Code || cp.Available != 200 {
		t.Fatalf("unexpected coupon preview: %+v", cp)
	}

	persisted, err := saleRepo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrderByID: %v", err)
	}
	if len(persisted.AppliedCouponIDs) != 1 || persisted.AppliedCouponIDs[0] != cards[0].ID {
		t.Fatalf("applied coupon not persisted: %+v", persisted.AppliedCouponIDs)
	}

	if _, err := uc.ClaimCoupon(ctx, order.ID, cards[0].Code); err == nil {
		t.Fatal("expected conflict on duplicate claim")
	}
}

func TestSettleOrderCreditsCardAndReverseRestores(t *testing.T) {
	ctx := context.Background()
	uc, loyaltyRepo, saleRepo := newTestEnv(t)

	if _, err := uc.CreateProgram(ctx, loyaltyusecase.CreateProgramInput{Name: "Loyalty Test", ProgramType: loyalty.ProgramTypeLoyalty}); err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	cards, err := uc.GenerateCards(ctx, loyaltyusecase.GenerateCardsInput{ProgramID: 1, Count: 1})
	if err != nil {
		t.Fatalf("GenerateCards: %v", err)
	}

	order := newSaleOrder()
	if err := saleRepo.CreateOrder(ctx, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if _, err := uc.ClaimCoupon(ctx, order.ID, cards[0].Code); err != nil {
		t.Fatalf("ClaimCoupon: %v", err)
	}

	order, err = saleRepo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrderByID: %v", err)
	}

	if err := uc.SettleOrder(ctx, order); err != nil {
		t.Fatalf("SettleOrder: %v", err)
	}
	card, err := loyaltyRepo.GetCardByID(ctx, cards[0].ID)
	if err != nil {
		t.Fatalf("GetCardByID: %v", err)
	}
	if card.Points != 200 || card.UseCount != 1 {
		t.Fatalf("expected card 200 points / 1 use, got %v/%d", card.Points, card.UseCount)
	}
	history, err := loyaltyRepo.ListHistoryByCard(ctx, card.ID)
	if err != nil {
		t.Fatalf("ListHistoryByCard: %v", err)
	}
	if len(history) == 0 || history[0].Issued != 200 {
		t.Fatalf("expected issued history of 200, got %+v", history)
	}

	// Projection rows are kept after settlement so a cancel can reverse them.
	projections, err := loyaltyRepo.ListCouponPointsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("ListCouponPointsByOrder: %v", err)
	}
	if len(projections) != 1 {
		t.Fatalf("expected 1 projection kept after settle, got %d", len(projections))
	}

	if err := uc.ReverseOrder(ctx, order); err != nil {
		t.Fatalf("ReverseOrder: %v", err)
	}
	card, err = loyaltyRepo.GetCardByID(ctx, cards[0].ID)
	if err != nil {
		t.Fatalf("GetCardByID: %v", err)
	}
	if card.Points != 0 {
		t.Fatalf("expected card back to 0 points after reverse, got %v", card.Points)
	}
	projections, err = loyaltyRepo.ListCouponPointsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("ListCouponPointsByOrder: %v", err)
	}
	if len(projections) != 0 {
		t.Fatalf("expected projections cleared after reverse, got %d", len(projections))
	}
}

func TestRedeemSettleDeductsAndCancelRefunds(t *testing.T) {
	ctx := context.Background()
	uc, loyaltyRepo, saleRepo := newTestEnv(t)

	if _, err := uc.CreateProgram(ctx, loyaltyusecase.CreateProgramInput{Name: "Loyalty Test", ProgramType: loyalty.ProgramTypeLoyalty}); err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	cards, err := uc.GenerateCards(ctx, loyaltyusecase.GenerateCardsInput{ProgramID: 1, Count: 1})
	if err != nil {
		t.Fatalf("GenerateCards: %v", err)
	}

	// Seed the card with an existing balance of 200 points.
	card, err := loyaltyRepo.GetCardByCode(ctx, cards[0].Code)
	if err != nil {
		t.Fatalf("GetCardByCode: %v", err)
	}
	card.Points = 200
	if err := loyaltyRepo.UpdateCard(ctx, card); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}

	order := newSaleOrder()
	if err := saleRepo.CreateOrder(ctx, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if _, err := uc.ClaimCoupon(ctx, order.ID, cards[0].Code); err != nil {
		t.Fatalf("ClaimCoupon: %v", err)
	}

	preview, err := uc.RedeemCoupon(ctx, order.ID, loyaltyusecase.RedeemInput{Code: cards[0].Code, RewardID: 1})
	if err != nil {
		t.Fatalf("RedeemCoupon: %v", err)
	}
	if preview.AppliedCoupons[0].Consumed != 200 || preview.AppliedCoupons[0].Available != 200 {
		t.Fatalf("unexpected balance after redeem: %+v", preview.AppliedCoupons[0])
	}

	order, err = saleRepo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrderByID: %v", err)
	}
	rewardLineFound := false
	for _, l := range order.Lines {
		if l.IsRewardLine {
			rewardLineFound = true
			if l.PointsCost != 200 || l.CouponID == nil || *l.CouponID != cards[0].ID {
				t.Fatalf("unexpected reward line: %+v", l)
			}
		}
	}
	if !rewardLineFound {
		t.Fatal("expected a reward line on the order")
	}

	// Settle: credit 200 earned, deduct 200 consumed -> balance unchanged at 200.
	if err := uc.SettleOrder(ctx, order); err != nil {
		t.Fatalf("SettleOrder: %v", err)
	}
	card, err = loyaltyRepo.GetCardByID(ctx, cards[0].ID)
	if err != nil {
		t.Fatalf("GetCardByID: %v", err)
	}
	if card.Points != 200 {
		t.Fatalf("expected net card balance 200 after settle, got %v", card.Points)
	}

	// Cancel a confirmed order: refund the consumed points, debit the earnings.
	if err := uc.ReverseOrder(ctx, order); err != nil {
		t.Fatalf("ReverseOrder: %v", err)
	}
	card, err = loyaltyRepo.GetCardByID(ctx, cards[0].ID)
	if err != nil {
		t.Fatalf("GetCardByID: %v", err)
	}
	if card.Points != 200 {
		t.Fatalf("expected card to retain original 200 after reverse, got %v", card.Points)
	}
}

func TestReleaseOrderForUnconfirmedCancelLeavesBalanceUntouched(t *testing.T) {
	ctx := context.Background()
	uc, loyaltyRepo, saleRepo := newTestEnv(t)

	if _, err := uc.CreateProgram(ctx, loyaltyusecase.CreateProgramInput{Name: "Loyalty Test", ProgramType: loyalty.ProgramTypeLoyalty}); err != nil {
		t.Fatalf("CreateProgram: %v", err)
	}
	cards, err := uc.GenerateCards(ctx, loyaltyusecase.GenerateCardsInput{ProgramID: 1, Count: 1})
	if err != nil {
		t.Fatalf("GenerateCards: %v", err)
	}

	card, err := loyaltyRepo.GetCardByCode(ctx, cards[0].Code)
	if err != nil {
		t.Fatalf("GetCardByCode: %v", err)
	}
	card.Points = 500
	if err := loyaltyRepo.UpdateCard(ctx, card); err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}

	// A draft order is cancelled before settlement: balances must be untouched.
	order := newSaleOrder()
	if err := saleRepo.CreateOrder(ctx, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if _, err := uc.ClaimCoupon(ctx, order.ID, cards[0].Code); err != nil {
		t.Fatalf("ClaimCoupon: %v", err)
	}

	if err := uc.ReleaseOrder(ctx, order); err != nil {
		t.Fatalf("ReleaseOrder: %v", err)
	}
	card, err = loyaltyRepo.GetCardByID(ctx, cards[0].ID)
	if err != nil {
		t.Fatalf("GetCardByID: %v", err)
	}
	if card.Points != 500 {
		t.Fatalf("expected untouched balance 500, got %v", card.Points)
	}
	projections, err := loyaltyRepo.ListCouponPointsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("ListCouponPointsByOrder: %v", err)
	}
	if len(projections) != 0 {
		t.Fatalf("expected projections cleared, got %d", len(projections))
	}
}