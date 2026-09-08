package deliveryusecase

import (
	"context"
	"fmt"
	"log/slog"

	"cashflow_backend/internal/domain/delivery"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/sale"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

type UseCase struct {
	repo        delivery.Repository
	saleRepo    sale.Repository
	productRepo product.Repository
	logger      *slog.Logger
}

func NewUseCase(
	repo delivery.Repository,
	saleRepo sale.Repository,
	productRepo product.Repository,
	logger *slog.Logger,
) *UseCase {
	return &UseCase{
		repo:        repo,
		saleRepo:    saleRepo,
		productRepo: productRepo,
		logger:      logger,
	}
}

// CalculateRate computes the shipping cost for a given Sale Order and Carrier.
func (uc *UseCase) CalculateRate(ctx context.Context, orderID int64, carrierID int64) (*delivery.RateResult, error) {
	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	carrier, err := uc.repo.GetCarrierByID(ctx, carrierID)
	if err != nil {
		return nil, err
	}

	// 1. Check Matching (Address, Weight, Volume)
	if !uc.match(ctx, carrier, order) {
		return &delivery.RateResult{
			Success:      false,
			ErrorMessage: "this delivery method is not available for this address or order constraints",
		}, nil
	}

	// 2. Compute Rate
	var price float64
	switch carrier.DeliveryType {
	case delivery.CarrierTypeFixed:
		price = carrier.FixedPrice
	case delivery.CarrierTypeBaseOnRule:
		price, err = uc.calculateBaseOnRule(ctx, carrier, order)
		if err != nil {
			return &delivery.RateResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}
	default:
		return &delivery.RateResult{
			Success:      false,
			ErrorMessage: "unknown delivery type",
		}, nil
	}

	// 3. Apply Margin
	carrierPrice := price
	price = price*(1.0+carrier.Margin) + carrier.FixedMargin

	// 4. Free over rule
	if carrier.FreeOver && order.AmountUntaxed >= carrier.Amount {
		price = 0
	}

	return &delivery.RateResult{
		Success:      true,
		Price:        price,
		CarrierPrice: carrierPrice,
	}, nil
}

func (uc *UseCase) match(ctx context.Context, carrier *delivery.DeliveryCarrier, order *sale.SaleOrder) bool {
	// Address matching would normally require partner details.
	// We assume partner info is available or passed in.
	// For now, let's just implement Weight/Volume matching if set.

	totalWeight := uc.getOrderWeight(ctx, order)
	if carrier.MaxWeight > 0 && totalWeight > carrier.MaxWeight {
		return false
	}

	// Geofencing (Simple check if rules exist)
	// In a real scenario, we'd check order.PartnerShippingID address fields.

	return true
}

func (uc *UseCase) calculateBaseOnRule(ctx context.Context, carrier *delivery.DeliveryCarrier, order *sale.SaleOrder) (float64, error) {
	stats := uc.getOrderStats(ctx, order)

	for _, rule := range carrier.PriceRules {
		value := stats[rule.Variable]
		match := false

		switch rule.Operator {
		case "==": match = value == rule.MaxValue
		case "<=": match = value <= rule.MaxValue
		case "<":  match = value <  rule.MaxValue
		case ">=": match = value >= rule.MaxValue
		case ">":  match = value >  rule.MaxValue
		}

		if match {
			factorValue := stats[rule.VariableFactor]
			return rule.ListBasePrice + (rule.ListPrice * factorValue), nil
		}
	}

	return 0, fmt.Errorf("no matching rule found for this order")
}

func (uc *UseCase) getOrderWeight(ctx context.Context, order *sale.SaleOrder) float64 {
	var totalWeight float64
	for _, line := range order.Lines {
		if line.IsRewardLine || line.IsDelivery {
			continue
		}
		p, err := uc.productRepo.GetVariantByID(ctx, line.ProductID)
		if err != nil {
			continue
		}
		tmpl, err := uc.productRepo.GetTemplateByID(ctx, p.TemplateID)
		if err != nil {
			continue
		}
		if tmpl.Type == product.ProductTypeGoods {
			totalWeight += line.ProductUomQty * tmpl.Weight
		}
	}
	return totalWeight
}

func (uc *UseCase) getOrderStats(ctx context.Context, order *sale.SaleOrder) map[string]float64 {
	stats := make(map[string]float64)
	var weight, volume, quantity float64

	for _, line := range order.Lines {
		if line.IsRewardLine || line.IsDelivery {
			continue
		}
		p, err := uc.productRepo.GetVariantByID(ctx, line.ProductID)
		if err != nil {
			continue
		}
		tmpl, err := uc.productRepo.GetTemplateByID(ctx, p.TemplateID)
		if err != nil {
			continue
		}

		weight += line.ProductUomQty * tmpl.Weight
		volume += line.ProductUomQty * tmpl.Volume
		quantity += line.ProductUomQty
	}

	stats["weight"] = weight
	stats["volume"] = volume
	stats["wv"] = weight * volume
	stats["price"] = order.AmountUntaxed
	stats["quantity"] = quantity

	return stats
}

// ListCarriers returns all active delivery carriers.
func (uc *UseCase) ListCarriers(ctx context.Context) ([]delivery.DeliveryCarrier, error) {
	return uc.repo.ListCarriers(ctx, nil)
}

// AddShippingToOrder adds a shipping line to the sale order.
func (uc *UseCase) AddShippingToOrder(ctx context.Context, orderID int64, carrierID int64) error {
	res, err := uc.CalculateRate(ctx, orderID, carrierID)
	if err != nil {
		return err
	}
	if !res.Success {
		return platformerrors.Conflict(res.ErrorMessage)
	}

	order, err := uc.saleRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	carrier, err := uc.repo.GetCarrierByID(ctx, carrierID)
	if err != nil {
		return err
	}

	// Remove existing delivery lines
	newLines := make([]sale.SaleOrderLine, 0, len(order.Lines))
	for _, l := range order.Lines {
		if !l.IsDelivery || l.QtyInvoiced > 0 {
			newLines = append(newLines, l)
		} else if l.IsDelivery && l.QtyInvoiced > 0 {
			return platformerrors.Conflict("cannot change shipping; already invoiced")
		}
	}
	order.Lines = newLines

	// Add new delivery line
	deliveryLine := sale.SaleOrderLine{
		OrderID:       order.ID,
		ProductID:     carrier.ProductID,
		Name:          i18n.NewTranslation(carrier.Name),
		ProductUomQty: 1,
		UnitPrice:     res.Price,
		IsDelivery:    true,
	}

	// Note: In a real system we would fetch taxes for the delivery product.

	order.Lines = append(order.Lines, deliveryLine)
	order.CarrierID = &carrier.ID
	order.RecomputeTotals()

	return uc.saleRepo.UpdateOrder(ctx, order)
}
