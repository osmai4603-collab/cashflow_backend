package purchaseusecase_test

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/purchase"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"
)

type mockPurchaseAccountingCreator struct {
	createdBills []accountingusecase.CreateInvoiceInput
	nextID       int64
}

func (m *mockPurchaseAccountingCreator) CreateInvoice(ctx context.Context, in accountingusecase.CreateInvoiceInput) (*accounting.AccountMove, error) {
	m.nextID++
	m.createdBills = append(m.createdBills, in)
	return &accounting.AccountMove{
		ID:        m.nextID,
		Name:      "BILL/2026/00001",
		MoveType:  in.MoveType,
		PartnerID: &in.PartnerID,
		JournalID: in.JournalID,
		Date:      in.Date,
		State:     accounting.MoveStateDraft,
	}, nil
}

type inMemoryPurchaseRepo struct {
	orders map[int64]*purchase.PurchaseOrder
	links  map[int64][]int64
	purchase.Repository
}

func (r *inMemoryPurchaseRepo) GetOrderByID(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	if o, ok := r.orders[id]; ok {
		return o, nil
	}
	return nil, nil
}

func (r *inMemoryPurchaseRepo) UpdateOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	r.orders[order.ID] = order
	return nil
}

func (r *inMemoryPurchaseRepo) LinkBill(ctx context.Context, orderID int64, moveID int64) error {
	r.links[orderID] = append(r.links[orderID], moveID)
	return nil
}

func TestBillService_CreateBillFromOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("Cannot bill draft purchase order", func(t *testing.T) {
		repo := &inMemoryPurchaseRepo{
			orders: map[int64]*purchase.PurchaseOrder{
				1: {
					ID:    1,
					Name:  "PO001",
					State: purchase.OrderStateDraft,
				},
			},
			links: make(map[int64][]int64),
		}
		acct := &mockPurchaseAccountingCreator{}
		svc := purchaseusecase.NewBillService(repo, acct, nil)

		_, err := svc.CreateBillFromOrder(ctx, 1)
		if err == nil {
			t.Fatal("expected error when billing draft order, got nil")
		}
	})

	t.Run("Full billing on received quantities", func(t *testing.T) {
		repo := &inMemoryPurchaseRepo{
			orders: map[int64]*purchase.PurchaseOrder{
				2: {
					ID:        2,
					Name:      "PO002",
					State:     purchase.OrderStatePurchase,
					PartnerID: 55,
					Lines: []purchase.PurchaseOrderLine{
						{
							ID:          10,
							ProductID:   101,
							Name:        "Raw Material A",
							ProductQty:  20,
							QtyReceived: 20,
							UnitPrice:   15,
							QtyInvoiced: 0,
						},
					},
				},
			},
			links: make(map[int64][]int64),
		}
		acct := &mockPurchaseAccountingCreator{}
		svc := purchaseusecase.NewBillService(repo, acct, nil, purchaseusecase.BillPolicyReceived)

		bill, err := svc.CreateBillFromOrder(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error creating bill: %v", err)
		}

		if bill == nil || bill.ID != 1 {
			t.Fatalf("expected created bill with ID 1")
		}

		order := repo.orders[2]
		if order.Lines[0].QtyInvoiced != 20 {
			t.Errorf("expected line 0 qty_invoiced=20, got %.2f", order.Lines[0].QtyInvoiced)
		}
		if order.InvoiceStatus != purchase.InvoiceStatusInvoiced {
			t.Errorf("expected order status 'invoiced', got %s", order.InvoiceStatus)
		}
		if len(repo.links[2]) != 1 || repo.links[2][0] != 1 {
			t.Errorf("expected bill 1 linked to order 2")
		}

		// Trying to bill again should fail
		_, err = svc.CreateBillFromOrder(ctx, 2)
		if err == nil {
			t.Error("expected conflict error when billing already billed order")
		}
	})

	t.Run("Partial billing based on received quantities", func(t *testing.T) {
		repo := &inMemoryPurchaseRepo{
			orders: map[int64]*purchase.PurchaseOrder{
				3: {
					ID:        3,
					Name:      "PO003",
					State:     purchase.OrderStatePurchase,
					PartnerID: 55,
					Lines: []purchase.PurchaseOrderLine{
						{
							ID:          20,
							ProductID:   101,
							Name:        "Raw Material A",
							ProductQty:  50,
							QtyReceived: 20,
							UnitPrice:   15,
							QtyInvoiced: 0,
						},
					},
				},
			},
			links: make(map[int64][]int64),
		}
		acct := &mockPurchaseAccountingCreator{}
		svc := purchaseusecase.NewBillService(repo, acct, nil, purchaseusecase.BillPolicyReceived)

		_, err := svc.CreateBillFromOrder(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		order := repo.orders[3]
		if order.Lines[0].QtyInvoiced != 20 {
			t.Errorf("expected qty_invoiced=20, got %.2f", order.Lines[0].QtyInvoiced)
		}
		if order.InvoiceStatus != purchase.InvoiceStatusToInvoice {
			t.Errorf("expected order status 'to_invoice' after partial billing, got %s", order.InvoiceStatus)
		}
	})
}
