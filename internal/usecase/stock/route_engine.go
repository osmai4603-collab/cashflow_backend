package stockusecase

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/stock"
)

// RunProcurement executes stock rules based on route and demand.
// This is the core engine for Odoo 19-style supply chain automation.
func (uc *UseCase) RunProcurement(ctx context.Context, routeID int64, productID int64, qty float64, locationDestID int64, groupID *int64) error {
	rule, err := uc.repo.FindRule(ctx, routeID, locationDestID)
	if err != nil {
		return fmt.Errorf("failed to find rule for route %d and destination %d: %w", routeID, locationDestID, err)
	}

	switch rule.Action {
	case stock.ActionPull:
		return uc.runPull(ctx, rule, productID, qty, groupID)
	case stock.ActionBuy:
		return uc.runBuy(ctx, rule, productID, qty, groupID)
	case stock.ActionManufacture:
		return uc.runManufacture(ctx, rule, productID, qty, groupID)
	default:
		return fmt.Errorf("unsupported rule action: %s", rule.Action)
	}
}

func (uc *UseCase) runPull(ctx context.Context, rule *stock.StockRule, productID int64, qty float64, groupID *int64) error {
	if rule.LocationSrcID == nil {
		return fmt.Errorf("pull rule %d missing source location", rule.ID)
	}

	// Create a picking/move from src to dest
	picking, err := uc.CreatePicking(ctx, CreatePickingInput{
		PickingType:    stock.PickingTypeInternal,
		LocationID:     rule.LocationSrcID,
		LocationDestID: &rule.LocationDestID,
		Origin:         fmt.Sprintf("Procurement Rule %d", rule.ID),
		CompanyID:      &rule.CompanyID,
		Moves: []CreateMoveInput{
			{
				ProductID:  productID,
				ProductQty: qty,
			},
		},
	})
	if err != nil {
		return err
	}

	// If rule is make_to_order (MTO), it should trigger another procurement for its source.
	if rule.ProcureMethod == stock.ProcureFromRule {
		// In Odoo, this would look up the routes again for rule.LocationSrcID
		uc.logger.InfoContext(ctx, "MTO rule chain triggered", "src_loc", *rule.LocationSrcID)
	}

	uc.logger.InfoContext(ctx, "pull rule executed", "rule_id", rule.ID, "picking_id", picking.ID)
	return nil
}

func (uc *UseCase) runBuy(ctx context.Context, rule *stock.StockRule, productID int64, qty float64, groupID *int64) error {
	// Integration point for Purchase module.
	// In a real implementation, this would call purchaseUseCase.CreateRFQ()
	uc.logger.InfoContext(ctx, "buy action triggered by stock rule", "product_id", productID, "qty", qty, "rule_id", rule.ID)
	return nil
}

func (uc *UseCase) runManufacture(ctx context.Context, rule *stock.StockRule, productID int64, qty float64, groupID *int64) error {
	// Integration point for MRP module.
	// In a real implementation, this would call mrpUseCase.CreateProduction()
	uc.logger.InfoContext(ctx, "manufacture action triggered by stock rule", "product_id", productID, "qty", qty, "rule_id", rule.ID)
	return nil
}
