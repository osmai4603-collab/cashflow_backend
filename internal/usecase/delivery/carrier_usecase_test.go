package deliveryusecase

import (
	"context"
	"testing"

	deliverystorage "cashflow_backend/internal/adapters/storage/delivery"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	deliverydomain "cashflow_backend/internal/domain/delivery"
	"cashflow_backend/internal/domain/sale"
	"log/slog"
	"os"
)

func TestCalculateRateFixed(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo := deliverystorage.NewMemoryRepository()
	saleRepo := salestorage.NewMemoryRepo()
	prodRepo := productstorage.NewMemoryRepo()

	uc := NewUseCase(repo, saleRepo, prodRepo, logger)

	// Setup carrier
	carrier := &deliverydomain.DeliveryCarrier{
		Name: "Fixed $10",
		DeliveryType: deliverydomain.CarrierTypeFixed,
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
