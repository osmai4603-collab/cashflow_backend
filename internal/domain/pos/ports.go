package pos

import "context"

type Repository interface {
	CreateConfig(ctx context.Context, config *PosConfig) error
	GetConfig(ctx context.Context, id int64) (*PosConfig, error)
	CreateSession(ctx context.Context, session *PosSession) error
	GetSession(ctx context.Context, id int64) (*PosSession, error)
	UpdateSession(ctx context.Context, session *PosSession) error
	CreateOrder(ctx context.Context, order *PosOrder) error
	GetOrderByClientUUID(ctx context.Context, clientUUID string) (*PosOrder, error)
	UpdateOrder(ctx context.Context, order *PosOrder) error
	CreateCashMovement(ctx context.Context, movement *CashInOutMovement) error
	SaveSyncResult(ctx context.Context, result *SyncResult) error
	GetSyncResult(ctx context.Context, idempotencyKey string) (*SyncResult, error)
}

type GlueService interface {
	SyncOrdersToStock(ctx context.Context, session *PosSession) error
	SyncSessionToAccounting(ctx context.Context, session *PosSession) error
	ProcessLoyaltyForOrder(ctx context.Context, order *PosOrder) error
}
