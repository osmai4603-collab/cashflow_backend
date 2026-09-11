package saleusecase_test

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/sale"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	saleusecase "cashflow_backend/internal/usecase/sale"
)

type mockSaleAccountingCreator struct {
	createdInvoices []accountingusecase.CreateInvoiceInput
	nextID          int64
}

func (m *mockSaleAccountingCreator) CreateInvoice(ctx context.Context, in accountingusecase.CreateInvoiceInput) (*accounting.AccountMove, error) {
	m.nextID++
	m.createdInvoices = append(m.createdInvoices, in)
	return &accounting.AccountMove{
		ID:        m.nextID,
		Name:      "INV/2026/00001",
		MoveType:  in.MoveType,
		PartnerID: &in.PartnerID,
		JournalID: in.JournalID,
		Date:      in.Date,
		State:     accounting.MoveStateDraft,
	}, nil
}

type inMemorySaleRepo struct {
	orders map[int64]*sale.SaleOrder
	links  map[int64][]int64
	sale.Repository
}

func (r *inMemorySaleRepo) GetOrderByID(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	if o, ok := r.orders[id]; ok {
		return o, nil
	}
	return nil, nil
}

func (r *inMemorySaleRepo) UpdateOrder(ctx context.Context, order *sale.SaleOrder) error {
	r.orders[order.ID] = order
	return nil
}

func (r *inMemorySaleRepo) LinkInvoice(ctx context.Context, orderID int64, moveID int64) error {
	r.links[orderID] = append(r.links[orderID], moveID)
	return nil
}

func TestInvoiceService_CreateInvoiceFromOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("Cannot invoice draft order", func(t *testing.T) {
		repo := &inMemorySaleRepo{
			orders: map[int64]*sale.SaleOrder{
				1: {
					ID:    1,
					Name:  "SO001",
					State: sale.OrderStateDraft,
				},
			},
			links: make(map[int64][]int64),
		}
		acct := &mockSaleAccountingCreator{}
		svc := saleusecase.NewInvoiceService(repo, acct, nil)

		_, err := svc.CreateInvoiceFromOrder(ctx, 1)
		if err == nil {
			t.Fatal("expected error when invoicing draft order, got nil")
		}
	})

	t.Run("Full invoicing on ordered quantities", func(t *testing.T) {
		repo := &inMemorySaleRepo{
			orders: map[int64]*sale.SaleOrder{
				2: {
					ID:        2,
					Name:      "SO002",
					State:     sale.OrderStateSale,
					PartnerID: 15,
					Lines: []sale.SaleOrderLine{
						{
							ID:            10,
							ProductID:     101,
							Name:          "Product A",
							ProductUomQty: 5,
							UnitPrice:     50,
							QtyInvoiced:   0,
						},
						{
							ID:            11,
							ProductID:     102,
							Name:          "Product B",
							ProductUomQty: 2,
							UnitPrice:     100,
							QtyInvoiced:   0,
						},
					},
				},
			},
			links: make(map[int64][]int64),
		}
		acct := &mockSaleAccountingCreator{}
		svc := saleusecase.NewInvoiceService(repo, acct, nil, saleusecase.InvoicePolicyOrder)

		inv, err := svc.CreateInvoiceFromOrder(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error creating invoice: %v", err)
		}

		if inv == nil || inv.ID != 1 {
			t.Fatalf("expected created invoice with ID 1")
		}

		// Verify line invoiced quantities
		order := repo.orders[2]
		if order.Lines[0].QtyInvoiced != 5 {
			t.Errorf("expected line 0 qty_invoiced=5, got %.2f", order.Lines[0].QtyInvoiced)
		}
		if order.Lines[1].QtyInvoiced != 2 {
			t.Errorf("expected line 1 qty_invoiced=2, got %.2f", order.Lines[1].QtyInvoiced)
		}
		if order.InvoiceStatus != sale.InvoiceStatusInvoiced {
			t.Errorf("expected order status 'invoiced', got %s", order.InvoiceStatus)
		}
		if len(repo.links[2]) != 1 || repo.links[2][0] != 1 {
			t.Errorf("expected invoice 1 linked to order 2")
		}

		// Trying to invoice again should fail (nothing left to invoice)
		_, err = svc.CreateInvoiceFromOrder(ctx, 2)
		if err == nil {
			t.Error("expected conflict error on second invoicing attempt")
		}
	})

	t.Run("Partial invoicing based on delivered quantities", func(t *testing.T) {
		repo := &inMemorySaleRepo{
			orders: map[int64]*sale.SaleOrder{
				3: {
					ID:        3,
					Name:      "SO003",
					State:     sale.OrderStateSale,
					PartnerID: 20,
					Lines: []sale.SaleOrderLine{
						{
							ID:            20,
							ProductID:     101,
							Name:          "Product A",
							ProductUomQty: 10,
							QtyDelivered:  6,
							QtyInvoiced:   0,
							UnitPrice:     30,
						},
					},
				},
			},
			links: make(map[int64][]int64),
		}
		acct := &mockSaleAccountingCreator{}
		svc := saleusecase.NewInvoiceService(repo, acct, nil, saleusecase.InvoicePolicyDelivery)

		_, err := svc.CreateInvoiceFromOrder(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		order := repo.orders[3]
		if order.Lines[0].QtyInvoiced != 6 {
			t.Errorf("expected qty_invoiced=6, got %.2f", order.Lines[0].QtyInvoiced)
		}
		if order.InvoiceStatus != sale.InvoiceStatusToInvoice {
			t.Errorf("expected order status 'to_invoice' after partial delivery invoicing, got %s", order.InvoiceStatus)
		}
	})
}
