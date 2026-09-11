package repair

import (
	"context"
	"fmt"
)

type RepairBillingService struct {
	stockPort StockPort
	movePort  InvoicePort
}

func NewRepairBillingService(stockPort StockPort, movePort InvoicePort) *RepairBillingService {
	return &RepairBillingService{
		stockPort: stockPort,
		movePort:  movePort,
	}
}

func (s *RepairBillingService) ProcessRepairCompletion(ctx context.Context, order *RepairOrder) error {
	order.CalculateTotal()

	// 1. Consume stock for repair parts
	for _, line := range order.Lines {
		err := s.stockPort.ConsumeStockForRepair(ctx, order.LocationID, line.ProductID, line.Quantity)
		if err != nil {
			return fmt.Errorf("failed to consume stock for product %d: %w", line.ProductID, err)
		}
	}

	// 2. Billing: If warranty check is true, the amount total billed is zero or ignored, otherwise bill labor/parts fee
	if order.WarrantyCheck {
		fmt.Printf("Repair order %s is under warranty. No customer move generated.\n", order.Name)
		order.State = StateDone
		return nil
	}

	// Generate invoice / account move for parts & labor
	moveID, err := s.movePort.CreateRepairInvoice(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to generate invoice for repair order %s: %w", order.Name, err)
	}

	order.AccountMoveID = &moveID
	order.State = StateDone
	return nil
}
