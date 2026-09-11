package posstorage

import (
	"context"
	"fmt"
	"sync"

	"cashflow_backend/internal/domain/pos"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepo struct {
	mu             sync.RWMutex
	configs        map[int64]*pos.PosConfig
	sessions       map[int64]*pos.PosSession
	orders         map[string]*pos.PosOrder
	cashMovements  map[int64]*pos.CashInOutMovement
	syncResults    map[string]*pos.SyncResult
	nextConfigID   int64
	nextSessionID  int64
	nextOrderID    int64
	nextMovementID int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		configs:       make(map[int64]*pos.PosConfig),
		sessions:      make(map[int64]*pos.PosSession),
		orders:        make(map[string]*pos.PosOrder),
		cashMovements: make(map[int64]*pos.CashInOutMovement),
		syncResults:   make(map[string]*pos.SyncResult),
	}
}

func (repo *MemoryRepo) CreateConfig(ctx context.Context, config *pos.PosConfig) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextConfigID++
	config.ID = repo.nextConfigID
	clone := *config
	repo.configs[config.ID] = &clone
	return nil
}

func (repo *MemoryRepo) GetConfig(ctx context.Context, id int64) (*pos.PosConfig, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	config, ok := repo.configs[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("pos config with ID %d not found", id))
	}
	clone := *config
	return &clone, nil
}

func (repo *MemoryRepo) CreateSession(ctx context.Context, session *pos.PosSession) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextSessionID++
	session.ID = repo.nextSessionID
	clone := *session
	repo.sessions[session.ID] = &clone
	return nil
}

func (repo *MemoryRepo) GetSession(ctx context.Context, id int64) (*pos.PosSession, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	session, ok := repo.sessions[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("pos session with ID %d not found", id))
	}
	clone := *session
	return &clone, nil
}

func (repo *MemoryRepo) UpdateSession(ctx context.Context, session *pos.PosSession) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, ok := repo.sessions[session.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("pos session with ID %d not found", session.ID))
	}
	clone := *session
	repo.sessions[session.ID] = &clone
	return nil
}

func (repo *MemoryRepo) CreateOrder(ctx context.Context, order *pos.PosOrder) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, exists := repo.orders[order.ClientUUID]; exists {
		return platformerrors.Conflict("order client UUID already exists")
	}
	repo.nextOrderID++
	order.ID = repo.nextOrderID
	clone := *order
	repo.orders[order.ClientUUID] = &clone
	return nil
}

func (repo *MemoryRepo) GetOrderByClientUUID(ctx context.Context, clientUUID string) (*pos.PosOrder, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	order, ok := repo.orders[clientUUID]
	if !ok {
		return nil, platformerrors.NotFound("pos order not found")
	}
	clone := *order
	return &clone, nil
}

func (repo *MemoryRepo) UpdateOrder(ctx context.Context, order *pos.PosOrder) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, ok := repo.orders[order.ClientUUID]; !ok {
		return platformerrors.NotFound("pos order not found")
	}
	clone := *order
	repo.orders[order.ClientUUID] = &clone
	return nil
}

func (repo *MemoryRepo) CreateCashMovement(ctx context.Context, movement *pos.CashInOutMovement) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextMovementID++
	movement.ID = repo.nextMovementID
	clone := *movement
	repo.cashMovements[movement.ID] = &clone
	return nil
}

func (repo *MemoryRepo) SaveSyncResult(ctx context.Context, result *pos.SyncResult) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	clone := *result
	repo.syncResults[result.IdempotencyKey] = &clone
	return nil
}

func (repo *MemoryRepo) GetSyncResult(ctx context.Context, key string) (*pos.SyncResult, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result, ok := repo.syncResults[key]
	if !ok {
		return nil, platformerrors.NotFound("pos sync result not found")
	}
	clone := *result
	return &clone, nil
}
