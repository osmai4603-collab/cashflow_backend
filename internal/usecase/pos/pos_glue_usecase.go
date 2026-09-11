package posusecase

import (
	"context"

	"cashflow_backend/internal/domain/pos"
)

type GlueUseCase struct {
	stock      StockSyncer
	accounting AccountingSyncer
	loyalty    LoyaltySyncer
}

type StockSyncer interface {
	SyncPOSSession(ctx context.Context, session *pos.PosSession) error
}

type AccountingSyncer interface {
	SyncPOSSession(ctx context.Context, session *pos.PosSession) error
}

type LoyaltySyncer interface {
	ProcessPOSOrder(ctx context.Context, order *pos.PosOrder) error
}

func NewGlueUseCase(stock StockSyncer, accounting AccountingSyncer, loyalty LoyaltySyncer) *GlueUseCase {
	return &GlueUseCase{stock: stock, accounting: accounting, loyalty: loyalty}
}

func (glue *GlueUseCase) SyncOrdersToStock(ctx context.Context, session *pos.PosSession) error {
	if glue.stock == nil {
		return nil
	}
	return glue.stock.SyncPOSSession(ctx, session)
}

func (glue *GlueUseCase) SyncSessionToAccounting(ctx context.Context, session *pos.PosSession) error {
	if glue.accounting == nil {
		return nil
	}
	return glue.accounting.SyncPOSSession(ctx, session)
}

func (glue *GlueUseCase) ProcessLoyaltyForOrder(ctx context.Context, order *pos.PosOrder) error {
	if glue.loyalty == nil {
		return nil
	}
	return glue.loyalty.ProcessPOSOrder(ctx, order)
}
