package repairusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/repair"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct {
	repo repair.Repository
}

func (u *UseCase) ListOrders(ctx context.Context) ([]repair.RepairOrder, error) {
	if u == nil || u.repo == nil {
		return nil, platformerrors.Internal("repair repository is not configured")
	}
	return u.repo.ListOrders(ctx, 0)
}

func (u *UseCase) CreateOrder(ctx context.Context, o *repair.RepairOrder) (*repair.RepairOrder, error) {
	if o == nil {
		return nil, platformerrors.Validation("repair order is required", nil)
	}
	if o.CompanyID == 0 {
		o.CompanyID = 1
	}
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = o.CreatedAt
	}
	if o.State == "" {
		o.State = repair.StateDraft
	}
	for i := range o.Lines {
		line := &o.Lines[i]
		if line.ProductID <= 0 {
			return nil, platformerrors.Validation("line product is required", nil)
		}
		if line.Quantity <= 0 {
			return nil, platformerrors.Validation("line quantity must be greater than zero", nil)
		}
		line.CalculateLineTotal()
	}
	if err := o.Validate(); err != nil {
		return nil, err
	}
	o.CalculateTotal()
	if err := u.repo.CreateOrder(ctx, o); err != nil {
		return nil, err
	}
	for i := range o.Lines {
		line := &o.Lines[i]
		line.RepairID = o.ID
		if err := u.repo.CreateOrderLine(ctx, line); err != nil {
			return nil, err
		}
	}
	o.UpdatedAt = time.Now().UTC()
	return o, nil
}

func (u *UseCase) ConfirmOrder(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.Validation("repair order id is required", nil)
	}
	order, err := u.repo.GetOrderByID(ctx, id)
	if err != nil {
		return err
	}
	if order == nil {
		return platformerrors.NotFound(fmt.Sprintf("repair order %d not found", id))
	}
	order.State = repair.StateConfirmed
	order.UpdatedAt = time.Now().UTC()
	return u.repo.UpdateOrder(ctx, order)
}

func (u *UseCase) CompleteOrder(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.Validation("repair order id is required", nil)
	}
	order, err := u.repo.GetOrderByID(ctx, id)
	if err != nil {
		return err
	}
	if order == nil {
		return platformerrors.NotFound(fmt.Sprintf("repair order %d not found", id))
	}
	if order.State == repair.StateDone {
		return nil
	}
	order.State = repair.StateDone
	order.UpdatedAt = time.Now().UTC()
	return u.repo.UpdateOrder(ctx, order)
}

func (u *UseCase) CreateInvoice(ctx context.Context, id int64) (int64, error) {
	if id <= 0 {
		return 0, platformerrors.Validation("repair order id is required", nil)
	}
	order, err := u.repo.GetOrderByID(ctx, id)
	if err != nil {
		return 0, err
	}
	if order == nil {
		return 0, platformerrors.NotFound(fmt.Sprintf("repair order %d not found", id))
	}
	if order.State != repair.StateDone {
		return 0, platformerrors.Validation("repair order must be done before invoicing", nil)
	}
	if order.AccountMoveID != nil {
		return *order.AccountMoveID, nil
	}
	invoiceID := id + 1000
	order.AccountMoveID = &invoiceID
	order.UpdatedAt = time.Now().UTC()
	if err := u.repo.UpdateOrder(ctx, order); err != nil {
		return 0, err
	}
	return invoiceID, nil
}

func New(repo repair.Repository) *UseCase {
	return &UseCase{repo: repo}
}
