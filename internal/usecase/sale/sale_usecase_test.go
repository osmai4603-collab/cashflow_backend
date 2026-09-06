package saleusecase_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	accountingstorage "cashflow_backend/internal/adapters/storage/accounting"
	partnerstorage "cashflow_backend/internal/adapters/storage/partner"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	salestorage "cashflow_backend/internal/adapters/storage/sale"
	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/sale"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	saleusecase "cashflow_backend/internal/usecase/sale"
)

func setupTestEnvironment(t *testing.T) (*saleusecase.UseCase, *salestorage.MemoryRepo, *accountingusecase.UseCase, int64, int64) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	// 1. Repositories
	saleRepo := salestorage.NewMemoryRepo()
	partnerRepo := partnerstorage.NewMemoryRepo()
	productRepo := productstorage.NewMemoryRepo()
	accountingRepo := accountingstorage.NewMemoryRepo()

	// 2. Accounting UseCase
	accountingUC := accountingusecase.New(accountingRepo, logger)

	// 3. Sale UseCase
	saleUC := saleusecase.New(saleRepo, partnerRepo, productRepo, accountingRepo, accountingUC, logger)

	// 4. Seed Partner (Customer)
	customer := &partner.Partner{
		Name:       "Acme Corp",
		IsCustomer: true,
		Active:     true,
	}
	if err := partnerRepo.Create(ctx, customer); err != nil {
		t.Fatalf("failed to seed customer: %v", err)
	}

	// 5. Seed Product
	prod := &product.ProductTemplate{
		Name:      "Ergonomic Chair",
		Type:      product.ProductTypeGoods,
		SalePrice: 150.0,
		CostPrice: 80.0,
		SaleOK:    true,
		Active:    true,
	}
	if err := productRepo.CreateTemplate(ctx, prod); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	return saleUC, saleRepo, accountingUC, customer.ID, prod.ID
}

func TestSaleUseCase_EndToEndSalesFlow(t *testing.T) {
	ctx := context.Background()
	uc, _, _, customerID, productID := setupTestEnvironment(t)

	// 1. Create Quotation
	unitPrice := 150.0
	order, err := uc.CreateOrder(ctx, saleusecase.CreateSaleOrderInput{
		PartnerID: customerID,
		DateOrder: time.Now(),
		Lines: []saleusecase.CreateSaleOrderLineInput{
			{
				ProductID:     productID,
				ProductUomQty: 4.0,
				UnitPrice:     &unitPrice,
				Discount:      10.0, // 10% discount
				TaxIDs:        []int64{1}, // Standard 15% VAT
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create quotation: %v", err)
	}

	if order.State != sale.OrderStateDraft {
		t.Errorf("expected draft state, got %s", order.State)
	}
	// Untaxed: 4 * 150 * 0.9 = 540.0
	if order.AmountUntaxed != 540.0 {
		t.Errorf("expected untaxed amount 540.0, got %f", order.AmountUntaxed)
	}
	// Tax: 540 * 0.15 = 81.0
	if order.AmountTax != 81.0 {
		t.Errorf("expected tax amount 81.0, got %f", order.AmountTax)
	}
	// Total: 540 + 81 = 621.0
	if order.AmountTotal != 621.0 {
		t.Errorf("expected total amount 621.0, got %f", order.AmountTotal)
	}

	// 2. ActionSend
	order, err = uc.ActionSend(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to send quotation: %v", err)
	}
	if order.State != sale.OrderStateSent {
		t.Errorf("expected sent state, got %s", order.State)
	}

	// 3. ConfirmOrder -> Becomes Sale Order
	order, err = uc.ConfirmOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to confirm order: %v", err)
	}
	if order.State != sale.OrderStateSale {
		t.Errorf("expected sale state, got %s", order.State)
	}
	if order.InvoiceStatus != sale.InvoiceStatusToInvoice {
		t.Errorf("expected invoice_status 'to_invoice', got %s", order.InvoiceStatus)
	}

	// 4. Create Partial Invoice (Invoice 2 out of 4 chairs)
	partialInv, err := uc.CreateInvoiceFromOrder(ctx, order.ID, saleusecase.CreateInvoiceFromOrderInput{
		Date: time.Now(),
		Lines: []saleusecase.LineInvoiceQuantityInput{
			{
				LineID:   order.Lines[0].ID,
				Quantity: 2.0,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create partial invoice: %v", err)
	}
	if partialInv == nil || partialInv.ID <= 0 {
		t.Fatalf("expected invoice to be created with valid ID")
	}

	// Verify order status after partial invoice
	order, err = uc.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to fetch order: %v", err)
	}
	if order.Lines[0].QtyInvoiced != 2.0 {
		t.Errorf("expected qty_invoiced 2.0, got %f", order.Lines[0].QtyInvoiced)
	}
	if order.InvoiceStatus != sale.InvoiceStatusToInvoice {
		t.Errorf("expected invoice_status 'to_invoice' after partial invoice, got %s", order.InvoiceStatus)
	}

	// 5. Invoice remaining quantities (Remaining 2 chairs)
	fullInv, err := uc.CreateInvoiceFromOrder(ctx, order.ID, saleusecase.CreateInvoiceFromOrderInput{
		Date: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to invoice remaining quantities: %v", err)
	}
	if fullInv == nil || fullInv.ID <= 0 {
		t.Fatalf("expected second invoice to be created")
	}

	// Verify order status after full invoicing
	order, err = uc.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to fetch order: %v", err)
	}
	if order.Lines[0].QtyInvoiced != 4.0 {
		t.Errorf("expected qty_invoiced 4.0, got %f", order.Lines[0].QtyInvoiced)
	}
	if order.InvoiceStatus != sale.InvoiceStatusInvoiced {
		t.Errorf("expected invoice_status 'invoiced' after full invoice, got %s", order.InvoiceStatus)
	}

	// 6. GetOrderInvoices
	invoices, err := uc.GetOrderInvoices(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get order invoices: %v", err)
	}
	if len(invoices) != 2 {
		t.Errorf("expected 2 linked invoices, got %d", len(invoices))
	}

	// 7. Cannot cancel an invoiced order
	_, err = uc.CancelOrder(ctx, order.ID)
	if err == nil {
		t.Errorf("expected error cancelling fully invoiced order, got nil")
	}
}
