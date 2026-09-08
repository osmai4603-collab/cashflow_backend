package deliveryusecase

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/delivery"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/adapters/storage/delivery"
	"cashflow_backend/internal/adapters/storage/sale"
	"cashflow_backend/internal/adapters/storage/product"
	"log/slog"
	"os"
)

func TestCalculateRateFixed(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo := delivery.NewMemoryRepository()
	saleRepo := sale.NewMemoryRepo()
	prodRepo := product.NewMemoryRepo()

	uc := NewUseCase(repo, saleRepo, prodRepo, logger)

	// Setup carrier
	carrier := &delivery.DeliveryCarrier{
		Name: "Fixed $10",
		DeliveryType: delivery.CarrierTypeFixed,
		FixedPrice: 10.0,
		ProductID: 1,
		Active: true,
	}
	repo.CreateCarrier(ctx, carrier)

	// Setup order
	order := &sale.SaleOrder{
		PartnerID: 1,
		AmountUntaxed: 100.0,
		Active: true,
	}
	saleRepo.CreateOrder(ctx, order)

	res, err := uc.CalculateRate(ctx, order.ID, carrier.ID)
	if err != nil {
		t.Fatalf("failed to calculate rate: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success, got error: %s", res.ErrorMessage)
	}

	if res.Price != 10.0 {
		t.Errorf("expected price 10.0, got %f", res.Price)
	}
}
