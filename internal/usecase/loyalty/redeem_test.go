package loyaltyusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	loyaltystorage "cashflow_backend/internal/adapters/storage/loyalty"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	"cashflow_backend/internal/domain/loyalty"
	"cashflow_backend/internal/domain/sale"
	loyaltyusecase "cashflow_backend/internal/usecase/loyalty"
	"strings"
)

func setupRedeemEnv(t *testing.T) (*loyaltyusecase.UseCase, *loyaltystorage.MemoryRepo, *salestorage.MemoryRepo, *productstorage.MemoryRepo) {
	t.Helper()
	loyaltyRepo := loyaltystorage.NewMemoryRepo()
	saleRepo := salestorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()
	uc := loyaltyusecase.New(loyaltyRepo, productRepo, saleRepo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return uc, loyaltyRepo, saleRepo, productRepo
}

func TestUseCase_RedeemCoupon_InsufficentPoints(t *testing.T) {
	ctx := context.Background()
	uc, lRepo, sRepo, _ := setupRedeemEnv(t)

	// 1. Create Program
	prog := &loyalty.LoyaltyProgram{
		Name:        "Promo 10",
		ProgramType: loyalty.ProgramTypePromotion,
		Active:      true,
		AppliesOn:   loyalty.AppliesOnCurrent,
	}
	if err := lRepo.CreateProgram(ctx, prog); err != nil {
		t.Fatal(err)
	}

	// 2. Create Reward: 10% discount for 100 points
	reward := &loyalty.LoyaltyReward{
		ProgramID:      prog.ID,
		RewardType:     loyalty.RewardTypeDiscount,
		DiscountMode:   loyalty.DiscountModePercent,
		Discount:       10.0,
		RequiredPoints: 100,
		Active:         true,
	}
	if err := lRepo.CreateReward(ctx, reward); err != nil {
		t.Fatal(err)
	}

	// 3. Create Card with 50 points
	card := &loyalty.LoyaltyCard{
		ProgramID: prog.ID,
		Code:      "SAVE10",
		Points:    50,
		Active:    true,
	}
	if err := lRepo.CreateCard(ctx, card); err != nil {
		t.Fatal(err)
	}

	// 4. Create Sale Order
	order := &sale.SaleOrder{
		PartnerID:        1,
		State:            sale.OrderStateDraft,
		AppliedCouponIDs: []int64{card.ID},
		Lines: []sale.SaleOrderLine{
			{ProductID: 1, UnitPrice: 100, ProductUomQty: 1, PriceSubtotal: 100, PriceTotal: 100},
		},
	}
	if err := sRepo.CreateOrder(ctx, order); err != nil {
		t.Fatal(err)
	}

	// 5. Attempt Redeem
	_, err := uc.RedeemCoupon(ctx, order.ID, loyaltyusecase.RedeemInput{
		Code:     "SAVE10",
		RewardID: reward.ID,
	})

	if err == nil {
		t.Fatal("expected error due to insufficient points, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient points to redeem this reward") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUseCase_RedeemCoupon_Success(t *testing.T) {
	ctx := context.Background()
	uc, lRepo, sRepo, _ := setupRedeemEnv(t)

	prog := &loyalty.LoyaltyProgram{
		Name:        "Loyalty Program",
		ProgramType: loyalty.ProgramTypeLoyalty,
		Active:      true,
		AppliesOn:   loyalty.AppliesOnCurrent,
	}
	lRepo.CreateProgram(ctx, prog)

	reward := &loyalty.LoyaltyReward{
		ProgramID:      prog.ID,
		RewardType:     loyalty.RewardTypeDiscount,
		DiscountMode:   loyalty.DiscountModePercent,
		Discount:       15.0,
		RequiredPoints: 100,
		Active:         true,
		Description:    "15% Discount",
	}
	lRepo.CreateReward(ctx, reward)

	card := &loyalty.LoyaltyCard{
		ProgramID: prog.ID,
		Code:      "LOYALTY-001",
		Points:    200,
		Active:    true,
	}
	lRepo.CreateCard(ctx, card)

	order := &sale.SaleOrder{
		PartnerID:        1,
		State:            sale.OrderStateDraft,
		AppliedCouponIDs: []int64{card.ID},
		Lines: []sale.SaleOrderLine{
			{ProductID: 1, UnitPrice: 200, ProductUomQty: 2, PriceSubtotal: 400, PriceTotal: 400},
		},
	}
	sRepo.CreateOrder(ctx, order)

	// Redeem
	preview, err := uc.RedeemCoupon(ctx, order.ID, loyaltyusecase.RedeemInput{
		Code:     "LOYALTY-001",
		RewardID: reward.ID,
	})

	if err != nil {
		t.Fatalf("RedeemCoupon failed: %v", err)
	}

	// Verify Reward Line
	found := false
	for _, l := range preview.Order.Lines {
		if l.IsRewardLine {
			found = true
			if l.UnitPrice != -60.0 { // 15% of 400
				t.Errorf("expected discount -60.0, got %v", l.UnitPrice)
			}
			if l.PointsCost != 100.0 {
				t.Errorf("expected points cost 100, got %v", l.PointsCost)
			}
		}
	}
	if !found {
		t.Fatal("reward line not found in preview")
	}

	// Verify remaining points in preview (200 initial - 100 cost = 100)
	// Note: If the program also gives points for the current order, they might be added to 'available'.
	// But in this test we didn't add rules for earning.
	if preview.AppliedCoupons[0].Available != 100 {
		t.Errorf("expected 100 points available in preview, got %v", preview.AppliedCoupons[0].Available)
	}
}

func TestUseCase_RedeemCoupon_FreeProduct(t *testing.T) {
	ctx := context.Background()
	uc, lRepo, sRepo, _ := setupRedeemEnv(t)

	prog := &loyalty.LoyaltyProgram{
		Name:        "Buy 1 Get 1",
		ProgramType: loyalty.ProgramTypeBuyXGetY,
		Active:      true,
		AppliesOn:   loyalty.AppliesOnCurrent,
	}
	lRepo.CreateProgram(ctx, prog)

	rewardProductID := int64(999)
	reward := &loyalty.LoyaltyReward{
		ProgramID:        prog.ID,
		RewardType:       loyalty.RewardTypeProduct,
		RewardProductID:  &rewardProductID,
		RewardProductQty: 1,
		RequiredPoints:   50,
		Active:           true,
		Description:      "Free Widget",
	}
	lRepo.CreateReward(ctx, reward)

	card := &loyalty.LoyaltyCard{
		ProgramID: prog.ID,
		Code:      "BOGO",
		Points:    100,
		Active:    true,
	}
	lRepo.CreateCard(ctx, card)

	order := &sale.SaleOrder{
		PartnerID:        1,
		State:            sale.OrderStateDraft,
		AppliedCouponIDs: []int64{card.ID},
		Lines: []sale.SaleOrderLine{
			{ProductID: 1, UnitPrice: 100, ProductUomQty: 1, PriceSubtotal: 100, PriceTotal: 100},
		},
	}
	sRepo.CreateOrder(ctx, order)

	preview, err := uc.RedeemCoupon(ctx, order.ID, loyaltyusecase.RedeemInput{
		Code:     "BOGO",
		RewardID: reward.ID,
	})

	if err != nil {
		t.Fatalf("RedeemCoupon failed: %v", err)
	}

	found := false
	for _, l := range preview.Order.Lines {
		if l.IsRewardLine && l.ProductID == rewardProductID {
			found = true
			if l.ProductUomQty != 1 {
				t.Errorf("expected qty 1, got %v", l.ProductUomQty)
			}
			if l.UnitPrice != 0 {
				t.Errorf("expected unit price 0 for free product, got %v", l.UnitPrice)
			}
		}
	}
	if !found {
		t.Fatal("free product reward line not found")
	}
}
