package stockusecase

import (
	"context"

	"cashflow_backend/internal/domain/stock"
)

// StockIntegrationHook defines callbacks triggered on stock picking lifecycle events.
// This allows other domains (accounting, sales, purchasing) to react to warehouse actions
// without tightly coupling the stock domain to external domain use cases.
type StockIntegrationHook interface {
	// OnPickingValidated is invoked immediately after a picking is successfully validated (done).
	OnPickingValidated(ctx context.Context, picking *stock.StockPicking) error

	// OnPickingCancelled is invoked immediately after a picking is cancelled.
	OnPickingCancelled(ctx context.Context, picking *stock.StockPicking) error
}

// CompositeStockIntegrationHook runs a chain of integration hooks in registration order.
type CompositeStockIntegrationHook struct {
	hooks []StockIntegrationHook
}

// NewCompositeStockIntegrationHook creates an aggregate hook from given listeners.
func NewCompositeStockIntegrationHook(hooks ...StockIntegrationHook) *CompositeStockIntegrationHook {
	return &CompositeStockIntegrationHook{hooks: hooks}
}

// Add registers a new hook to the listener chain.
func (c *CompositeStockIntegrationHook) Add(h StockIntegrationHook) {
	if h != nil {
		c.hooks = append(c.hooks, h)
	}
}

// OnPickingValidated broadcasts picking validation to all registered hooks.
func (c *CompositeStockIntegrationHook) OnPickingValidated(ctx context.Context, picking *stock.StockPicking) error {
	for _, h := range c.hooks {
		if err := h.OnPickingValidated(ctx, picking); err != nil {
			return err
		}
	}
	return nil
}

// OnPickingCancelled broadcasts picking cancellation to all registered hooks.
func (c *CompositeStockIntegrationHook) OnPickingCancelled(ctx context.Context, picking *stock.StockPicking) error {
	for _, h := range c.hooks {
		if err := h.OnPickingCancelled(ctx, picking); err != nil {
			return err
		}
	}
	return nil
}
