package purchaseusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/purchase"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// CreatePurchaseRequisitionLineInput is used to populate a requisition line.
type CreateRequisitionLineInput struct {
	ProductID    int64      `json:"product_id"`
	ProductQty   float64    `json:"product_qty"`
	ProductUOMID *int64     `json:"product_uom_id,omitempty"`
	PriceUnit    float64    `json:"price_unit"`
	ScheduleDate *time.Time `json:"schedule_date,omitempty"`
	SupplierID   *int64     `json:"supplier_id,omitempty"`
	Description  string     `json:"description,omitempty"`
}

// CreatePurchaseRequisitionInput creates a requisition.
type CreatePurchaseRequisitionInput struct {
	Name        string                       `json:"name"`
	Type        purchase.RequisitionType     `json:"type"`
	VendorID    *int64                       `json:"vendor_id,omitempty"`
	UserID      int64                        `json:"user_id"`
	DateStart   *time.Time                   `json:"date_start,omitempty"`
	DateEnd     *time.Time                   `json:"date_end,omitempty"`
	CurrencyID  int64                        `json:"currency_id"`
	CompanyID   int64                        `json:"company_id"`
	Description string                       `json:"description,omitempty"`
	Lines       []CreateRequisitionLineInput `json:"lines"`
}

// RequisitionRepository is the persistence contract for purchase requisitions.
type RequisitionRepository interface {
	CreateRequisition(ctx context.Context, requisition *purchase.PurchaseRequisition) error
	GetRequisitionByID(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error)
	ListRequisitions(ctx context.Context, page int, limit int) ([]purchase.PurchaseRequisition, error)
	UpdateRequisition(ctx context.Context, requisition *purchase.PurchaseRequisition) error
	DeleteRequisition(ctx context.Context, id int64) error
}

type SupplierInfoRepository interface {
	CreateForRequisition(ctx context.Context, requisition *purchase.PurchaseRequisition) error
	DeleteForRequisition(ctx context.Context, requisitionID int64) error
	UpdatePricesForRequisition(ctx context.Context, requisition *purchase.PurchaseRequisition) error
}

// RequisitionUseCase models the purchase requisition workflow.
type RequisitionUseCase struct {
	reqRepo          RequisitionRepository
	poRepo           purchase.Repository
	supplierInfoRepo SupplierInfoRepository
}

// NewRequisitionUseCase builds a requisition use case.
func NewRequisitionUseCase(reqRepo RequisitionRepository, poRepo purchase.Repository, supplierInfoRepo ...SupplierInfoRepository) *RequisitionUseCase {
	uc := &RequisitionUseCase{reqRepo: reqRepo, poRepo: poRepo}
	if len(supplierInfoRepo) > 0 {
		uc.supplierInfoRepo = supplierInfoRepo[0]
	}
	return uc
}

// CreateRequisition creates a new purchase requisition with validation.
func (uc *RequisitionUseCase) CreateRequisition(ctx context.Context, in CreatePurchaseRequisitionInput) (*purchase.PurchaseRequisition, error) {
	if len(in.Lines) == 0 {
		return nil, platformerrors.Validation("requisition requires at least one line", map[string]string{
			"lines": "must contain at least one item",
		})
	}

	if in.Type == "" {
		in.Type = purchase.RequisitionBlanketOrder
	}

	req := &purchase.PurchaseRequisition{
		Name:        in.Name,
		Active:      true,
		Type:        in.Type,
		VendorID:    in.VendorID,
		UserID:      in.UserID,
		DateStart:   in.DateStart,
		DateEnd:     in.DateEnd,
		CurrencyID:  in.CurrencyID,
		CompanyID:   in.CompanyID,
		Description: in.Description,
		State:       purchase.RequisitionDraft,
	}

	for _, line := range in.Lines {
		req.Lines = append(req.Lines, purchase.PurchaseRequisitionLine{
			ProductID:                  line.ProductID,
			ProductQty:                 line.ProductQty,
			ProductUOMID:               line.ProductUOMID,
			PriceUnit:                  line.PriceUnit,
			ScheduleDate:               line.ScheduleDate,
			SupplierID:                 line.SupplierID,
			ProductDescriptionVariants: line.Description,
			Description:                line.Description,
		})
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := uc.reqRepo.CreateRequisition(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// GetRequisitionByID fetches a requisition by ID.
func (uc *RequisitionUseCase) GetRequisitionByID(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error) {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.attachPurchaseOrderIDs(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// UpdateRequisition updates a requisition while preserving its current workflow state.
func (uc *RequisitionUseCase) UpdateRequisition(ctx context.Context, id int64, in CreatePurchaseRequisitionInput) (*purchase.PurchaseRequisition, error) {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.State != purchase.RequisitionDraft {
		return nil, platformerrors.Conflict("only draft requisitions can be updated")
	}
	if len(in.Lines) == 0 {
		return nil, platformerrors.Validation("requisition requires at least one line", map[string]string{
			"lines": "must contain at least one item",
		})
	}

	req.Name = in.Name
	req.Type = in.Type
	if req.Type == "" {
		req.Type = purchase.RequisitionBlanketOrder
	}
	req.VendorID = in.VendorID
	req.UserID = in.UserID
	req.DateStart = in.DateStart
	req.DateEnd = in.DateEnd
	req.CurrencyID = in.CurrencyID
	req.CompanyID = in.CompanyID
	req.Description = in.Description
	req.Lines = make([]purchase.PurchaseRequisitionLine, len(in.Lines))
	for i, line := range in.Lines {
		req.Lines[i] = purchase.PurchaseRequisitionLine{
			ProductID: line.ProductID, ProductQty: line.ProductQty, ProductUOMID: line.ProductUOMID,
			PriceUnit: line.PriceUnit, ScheduleDate: line.ScheduleDate, SupplierID: line.SupplierID,
			ProductDescriptionVariants: line.Description,
			Description:                line.Description,
		}
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := uc.reqRepo.UpdateRequisition(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// DeleteRequisition removes a draft or cancelled requisition.
func (uc *RequisitionUseCase) DeleteRequisition(ctx context.Context, id int64) error {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, id)
	if err != nil {
		return err
	}
	if req.State != purchase.RequisitionDraft && req.State != purchase.RequisitionCancel {
		return platformerrors.Conflict("only draft or cancelled requisitions can be deleted")
	}
	return uc.reqRepo.DeleteRequisition(ctx, id)
}

// ListRequisitions lists requisitions using the project pagination conventions.
func (uc *RequisitionUseCase) ListRequisitions(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[purchase.PurchaseRequisition], error) {
	items, err := uc.reqRepo.ListRequisitions(ctx, page.Page, page.LimitClamped())
	if err != nil {
		return pagination.PageResult[purchase.PurchaseRequisition]{}, err
	}
	result := pagination.NewPageResult(items, int64(len(items)), page)
	for i := range result.Items {
		if err := uc.attachPurchaseOrderIDs(ctx, &result.Items[i]); err != nil {
			return pagination.PageResult[purchase.PurchaseRequisition]{}, err
		}
	}
	return result, nil
}

func (uc *RequisitionUseCase) attachPurchaseOrderIDs(ctx context.Context, req *purchase.PurchaseRequisition) error {
	orders, err := uc.poRepo.ListOrders(ctx, filter.NewFilter().Add("requisition_id", filter.OpEqual, req.ID), pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		return err
	}
	req.PurchaseOrderIDs = make([]int64, len(orders.Items))
	for i, order := range orders.Items {
		req.PurchaseOrderIDs[i] = order.ID
	}
	return nil
}

// ConfirmRequisition confirms a draft requisition.
func (uc *RequisitionUseCase) ConfirmRequisition(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error) {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Confirm(); err != nil {
		return nil, err
	}
	if err := uc.reqRepo.UpdateRequisition(ctx, req); err != nil {
		return nil, err
	}
	if uc.supplierInfoRepo != nil && req.Type == purchase.RequisitionBlanketOrder {
		if err := uc.supplierInfoRepo.CreateForRequisition(ctx, req); err != nil {
			return nil, err
		}
	}
	return req, nil
}

// CloseRequisition closes a confirmed requisition.
func (uc *RequisitionUseCase) CloseRequisition(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error) {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	orders, err := uc.poRepo.ListOrders(ctx, filter.NewFilter().Add("requisition_id", filter.OpEqual, id), pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		return nil, err
	}
	for _, order := range orders.Items {
		if order.State != purchase.OrderStateDone && order.State != purchase.OrderStateCancel {
			return nil, platformerrors.Conflict("cannot close requisition while purchase orders are open")
		}
	}
	if err := req.Close(); err != nil {
		return nil, err
	}
	if err := uc.reqRepo.UpdateRequisition(ctx, req); err != nil {
		return nil, err
	}
	if uc.supplierInfoRepo != nil && req.Type == purchase.RequisitionBlanketOrder {
		if err := uc.supplierInfoRepo.DeleteForRequisition(ctx, req.ID); err != nil {
			return nil, err
		}
	}
	return req, nil
}

// CancelRequisition cancels a requisition.
func (uc *RequisitionUseCase) CancelRequisition(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error) {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := req.Cancel(); err != nil {
		return nil, err
	}
	orders, err := uc.poRepo.ListOrders(ctx, filter.NewFilter().Add("requisition_id", filter.OpEqual, id), pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		return nil, err
	}
	for i := range orders.Items {
		if orders.Items[i].State != purchase.OrderStateDraft {
			continue
		}
		orders.Items[i].State = purchase.OrderStateCancel
		if err := uc.poRepo.UpdateOrder(ctx, &orders.Items[i]); err != nil {
			return nil, err
		}
	}
	if uc.supplierInfoRepo != nil && req.Type == purchase.RequisitionBlanketOrder {
		if err := uc.supplierInfoRepo.DeleteForRequisition(ctx, req.ID); err != nil {
			return nil, err
		}
	}
	if err := uc.reqRepo.UpdateRequisition(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// CreatePurchaseOrderFromRequisition creates a purchase order from the requisition using the same vendor and line details.
func (uc *RequisitionUseCase) CreatePurchaseOrderFromRequisition(ctx context.Context, requisitionID int64) (*purchase.PurchaseOrder, error) {
	req, err := uc.reqRepo.GetRequisitionByID(ctx, requisitionID)
	if err != nil {
		return nil, err
	}
	if req.State == "" {
		req.State = purchase.RequisitionDraft
	}
	if req.State != purchase.RequisitionConfirmed {
		return nil, platformerrors.Conflict("purchase order can only be created from a confirmed requisition")
	}
	if req.VendorID == nil || *req.VendorID <= 0 {
		return nil, platformerrors.Validation("requisition vendor is required", map[string]string{
			"vendor_id": "must reference a valid supplier",
		})
	}
	if len(req.Lines) == 0 {
		return nil, platformerrors.Validation("requisition has no lines", map[string]string{
			"lines": "at least one requisition line is required",
		})
	}

	po := &purchase.PurchaseOrder{
		Name:          "PO/REQ/" + fmt.Sprintf("%d", requisitionID),
		PartnerID:     *req.VendorID,
		DateOrder:     time.Now().UTC(),
		State:         purchase.OrderStateDraft,
		InvoiceStatus: purchase.InvoiceStatusNo,
		Currency:      fmt.Sprintf("%d", req.CurrencyID),
		Note:          req.Description,
		Active:        true,
	}
	po.RequisitionID = &req.ID
	po.RequisitionType = func() *string { s := string(req.Type); return &s }()

	for i, line := range req.Lines {
		desc := line.ProductDescriptionVariants
		if desc == "" {
			desc = line.Description
		}
		if desc == "" {
			desc = fmt.Sprintf("Requisition line %d", i+1)
		}
		po.Lines = append(po.Lines, purchase.PurchaseOrderLine{
			Sequence:  i + 1,
			ProductID: line.ProductID,
			Name:      desc,
			ProductQty: func() float64 {
				if req.Type == purchase.RequisitionBlanketOrder {
					return 0
				}
				return line.ProductQty
			}(),
			ProductUom: line.ProductUOMID,
			UnitPrice:  line.PriceUnit,
		})
	}
	po.RecomputeTotals()
	if err := po.Validate(); err != nil {
		return nil, err
	}
	if err := uc.poRepo.CreateOrder(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}
