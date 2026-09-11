package stockusecase_test

import (
	"context"
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/stock"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// mockAccountingGateway records journal entries created by stock operations.
type mockAccountingGateway struct {
	createdEntries []accountingusecase.CreateJournalEntryInput
	postedEntries  []int64
	nextID         int64
}

func (m *mockAccountingGateway) CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error) {
	m.nextID++
	m.createdEntries = append(m.createdEntries, in)
	return &accounting.AccountMove{
		ID:        m.nextID,
		JournalID: in.JournalID,
		Date:      in.Date,
		Ref:       in.Ref,
		State:     accounting.MoveStateDraft,
	}, nil
}

func (m *mockAccountingGateway) PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	m.postedEntries = append(m.postedEntries, id)
	return &accounting.AccountMove{
		ID:    id,
		State: accounting.MoveStatePosted,
	}, nil
}

func (m *mockAccountingGateway) GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: id}, nil
}

// inMemoryStockRepo implements minimum methods needed for stock account tests.
type inMemoryStockRepo struct {
	locations map[int64]*stock.StockLocation
	moves     map[int64]*stock.StockMove
	stock.Repository
}

func (r *inMemoryStockRepo) GetLocationByID(ctx context.Context, id int64) (*stock.StockLocation, error) {
	if loc, ok := r.locations[id]; ok {
		return loc, nil
	}
	return nil, nil
}

func (r *inMemoryStockRepo) UpdateMoveValue(ctx context.Context, move *stock.StockMove) error {
	r.moves[move.ID] = move
	return nil
}

func TestStockAccountService_CreateValuationEntries(t *testing.T) {
	ctx := context.Background()

	locations := map[int64]*stock.StockLocation{
		1: {ID: 1, Name: "Suppliers", Usage: stock.LocationUsageSupplier},
		2: {ID: 2, Name: "Stock", Usage: stock.LocationUsageInternal},
		3: {ID: 3, Name: "Customers", Usage: stock.LocationUsageCustomer},
	}

	t.Run("Incoming Receipt Generates Debit Valuation and Credit Input", func(t *testing.T) {
		repo := &inMemoryStockRepo{
			locations: locations,
			moves:     make(map[int64]*stock.StockMove),
		}
		acctGateway := &mockAccountingGateway{}
		svc := stockusecase.NewStockAccountService(repo, acctGateway, nil, nil)

		now := time.Now().UTC()
		picking := &stock.StockPicking{
			ID:          101,
			Name:        "WH/IN/00001",
			PickingType: stock.PickingTypeIncoming,
			DateDone:    &now,
			Moves: []stock.StockMove{
				{
					ID:             1,
					Name:           "Item A Receipt",
					ProductID:      50,
					ProductQty:     10,
					QuantityDone:   10,
					StandardPrice:  25.0,
					LocationID:     1, // Supplier
					LocationDestID: 2, // Internal
				},
			},
		}

		if err := svc.OnPickingValidated(ctx, picking); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(acctGateway.createdEntries) != 1 {
			t.Fatalf("expected 1 journal entry, got %d", len(acctGateway.createdEntries))
		}

		entry := acctGateway.createdEntries[0]
		if len(entry.Lines) != 2 {
			t.Fatalf("expected 2 lines in journal entry, got %d", len(entry.Lines))
		}

		// Debit: Inventory (5), Credit: Stock Input (16), Amount: 250
		debitLine := entry.Lines[0]
		creditLine := entry.Lines[1]

		if debitLine.AccountID != 5 || debitLine.Debit != 250.0 {
			t.Errorf("expected debit 250 on account 5, got account %d debit %.2f", debitLine.AccountID, debitLine.Debit)
		}
		if creditLine.AccountID != 16 || creditLine.Credit != 250.0 {
			t.Errorf("expected credit 250 on account 16, got account %d credit %.2f", creditLine.AccountID, creditLine.Credit)
		}

		if len(acctGateway.postedEntries) != 1 {
			t.Errorf("expected entry to be posted")
		}

		// Move should have AccountMoveID set
		if picking.Moves[0].AccountMoveID == nil || *picking.Moves[0].AccountMoveID != 1 {
			t.Errorf("expected move AccountMoveID to be 1")
		}
	})

	t.Run("Outgoing Delivery Generates Debit COGS and Credit Valuation", func(t *testing.T) {
		repo := &inMemoryStockRepo{
			locations: locations,
			moves:     make(map[int64]*stock.StockMove),
		}
		acctGateway := &mockAccountingGateway{}
		svc := stockusecase.NewStockAccountService(repo, acctGateway, nil, nil)

		now := time.Now().UTC()
		picking := &stock.StockPicking{
			ID:          102,
			Name:        "WH/OUT/00001",
			PickingType: stock.PickingTypeOutgoing,
			DateDone:    &now,
			Moves: []stock.StockMove{
				{
					ID:             2,
					Name:           "Item A Delivery",
					ProductID:      50,
					ProductQty:     4,
					QuantityDone:   4,
					StandardPrice:  25.0,
					LocationID:     2, // Internal
					LocationDestID: 3, // Customer
				},
			},
		}

		if err := svc.OnPickingValidated(ctx, picking); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(acctGateway.createdEntries) != 1 {
			t.Fatalf("expected 1 journal entry, got %d", len(acctGateway.createdEntries))
		}

		entry := acctGateway.createdEntries[0]
		// Debit: COGS (12), Credit: Inventory (5), Amount: 100
		debitLine := entry.Lines[0]
		creditLine := entry.Lines[1]

		if debitLine.AccountID != 12 || debitLine.Debit != 100.0 {
			t.Errorf("expected debit 100 on COGS (12), got account %d debit %.2f", debitLine.AccountID, debitLine.Debit)
		}
		if creditLine.AccountID != 5 || creditLine.Credit != 100.0 {
			t.Errorf("expected credit 100 on Inventory (5), got account %d credit %.2f", creditLine.AccountID, creditLine.Credit)
		}
	})

	t.Run("Internal Transfer Produces No Accounting Entry", func(t *testing.T) {
		repo := &inMemoryStockRepo{
			locations: map[int64]*stock.StockLocation{
				2:  {ID: 2, Name: "Stock A", Usage: stock.LocationUsageInternal},
				20: {ID: 20, Name: "Stock B", Usage: stock.LocationUsageInternal},
			},
			moves: make(map[int64]*stock.StockMove),
		}
		acctGateway := &mockAccountingGateway{}
		svc := stockusecase.NewStockAccountService(repo, acctGateway, nil, nil)

		picking := &stock.StockPicking{
			ID:          103,
			Name:        "WH/INT/00001",
			PickingType: stock.PickingTypeInternal,
			Moves: []stock.StockMove{
				{
					ID:             3,
					Name:           "Internal move",
					ProductID:      50,
					ProductQty:     5,
					QuantityDone:   5,
					StandardPrice:  25.0,
					LocationID:     2,
					LocationDestID: 20,
				},
			},
		}

		if err := svc.OnPickingValidated(ctx, picking); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(acctGateway.createdEntries) != 0 {
			t.Errorf("expected 0 entries for internal transfer, got %d", len(acctGateway.createdEntries))
		}
	})
}
