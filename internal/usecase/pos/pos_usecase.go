package posusecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/pos"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo pos.Repository
	glue pos.GlueService
}

func New(repo pos.Repository) *UseCase {
	return &UseCase{repo: repo}
}

func NewWithGlue(repo pos.Repository, glue pos.GlueService) *UseCase {
	return &UseCase{repo: repo, glue: glue}
}

func (useCase *UseCase) CreateConfig(ctx context.Context, config *pos.PosConfig) (*pos.PosConfig, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateConfig(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (useCase *UseCase) OpenSession(ctx context.Context, configID, userID, companyID int64, openingBalance float64) (*pos.PosSession, error) {
	if _, err := useCase.repo.GetConfig(ctx, configID); err != nil {
		return nil, err
	}
	session := &pos.PosSession{Name: fmt.Sprintf("POS/%d", time.Now().UTC().UnixNano()), ConfigID: configID, UserID: userID, CompanyID: companyID}
	if err := session.Open(openingBalance); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (useCase *UseCase) CloseSession(ctx context.Context, sessionID int64, expectedCash, countedCash float64) (*pos.PosSession, error) {
	session, err := useCase.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := session.BeginClosing(expectedCash); err != nil {
		return nil, err
	}
	if err := session.Close(countedCash); err != nil {
		return nil, err
	}
	if err := useCase.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}
	if useCase.glue != nil {
		if err := useCase.glue.SyncOrdersToStock(ctx, session); err != nil {
			return nil, err
		}
		if err := useCase.glue.SyncSessionToAccounting(ctx, session); err != nil {
			return nil, err
		}
	}
	return session, nil
}

func (useCase *UseCase) CreateOrder(ctx context.Context, order *pos.PosOrder) (*pos.PosOrder, error) {
	if err := order.Validate(); err != nil {
		return nil, err
	}
	order.RecomputeTotals()
	if err := useCase.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (useCase *UseCase) SyncOrders(ctx context.Context, batch *pos.SyncBatch) (*pos.SyncResult, error) {
	batch.IdempotencyKey = strings.TrimSpace(batch.IdempotencyKey)
	if batch.SessionID <= 0 || batch.IdempotencyKey == "" {
		return nil, platformerrors.Validation("sync batch requires session and idempotency key", nil)
	}
	if existing, err := useCase.repo.GetSyncResult(ctx, batch.IdempotencyKey); err == nil {
		existing.Duplicate = true
		return existing, nil
	}
	result := &pos.SyncResult{SessionID: batch.SessionID, IdempotencyKey: batch.IdempotencyKey}
	for index := range batch.Orders {
		order := &batch.Orders[index]
		order.SessionID = batch.SessionID
		if _, err := useCase.CreateOrder(ctx, order); err != nil {
			return nil, fmt.Errorf("sync order %d: %w", index, err)
		}
		result.Accepted++
		result.OrderIDs = append(result.OrderIDs, order.ID)
	}
	if err := useCase.repo.SaveSyncResult(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}
