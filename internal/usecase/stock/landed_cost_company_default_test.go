package stockusecase

import (
	"context"
	"testing"
	"time"

	companystorage "cashflow_backend/internal/adapters/storage/company"
	productstorage "cashflow_backend/internal/adapters/storage/product"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/accounting"
	companydomain "cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/stock"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

func TestResolveLandedCostJournalUsesCompanyDefault(t *testing.T) {
	ctx := context.Background()
	repo := companystorage.NewMemoryRepo()
	journalID := int64(42)
	company := &companydomain.Company{
		Name:               "ACME",
		CurrencyID:         1,
		LandedCostJournalID: &journalID,
	}
	if err := repo.Create(ctx, company); err != nil {
		t.Fatalf("Create company failed: %v", err)
	}

	uc := &UseCase{companyRepo: repo, repo: stockstorage.NewMemoryRepo()}
	lc := &stock.LandedCost{CompanyID: company.ID}

	resolved, err := uc.resolveLandedCostJournal(ctx, lc)
	if err != nil {
		t.Fatalf("resolveLandedCostJournal failed: %v", err)
	}
	if resolved != journalID {
		t.Fatalf("expected default landed-cost journal %d, got %d", journalID, resolved)
	}
}

func TestCreateLandedCostUsesCompanyDefaultJournal(t *testing.T) {
	ctx := context.Background()
	companyRepo := companystorage.NewMemoryRepo()
	journalID := int64(88)
	company := &companydomain.Company{
		Name:               "Omega",
		CurrencyID:         1,
		LandedCostJournalID: &journalID,
	}
	if err := companyRepo.Create(ctx, company); err != nil {
		t.Fatalf("Create company failed: %v", err)
	}

	uc := New(stockstorage.NewMemoryRepo(), nil, nil, nil, nil, companyRepo)
	_, err := uc.CreateLandedCost(ctx, CreateLandedCostInput{
		PickingIDs: []int64{1},
		CompanyID:  company.ID,
		CostLines: []CreateLandedCostLineInput{{
			Name:        "Freight",
			ProductID:   1,
			AccountID:   17,
			PriceUnit:   50,
			SplitMethod: stock.SplitEqual,
		}},
	})
	if err != nil {
		t.Fatalf("CreateLandedCost failed: %v", err)
	}
	lc, err := uc.repo.GetLandedCostByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetLandedCostByID failed: %v", err)
	}
	if lc.JournalID != journalID {
		t.Fatalf("expected journal %d from company default, got %d", journalID, lc.JournalID)
	}
}

func TestValidateLandedCostUsesQuantityWhenRemainingQtyUnset(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	uc := &UseCase{repo: repo, accountingSvc: noopAccountingGateway{}}

	move := &stock.StockMove{
		ID:             1,
		ProductID:      10,
		ProductQty:     10,
		QuantityDone:   10,
		LocationID:     1,
		LocationDestID: 2,
		State:          stock.MoveStateDone,
		StandardPrice:  5,
		Value:          50,
		RemainingQty:   0,
		RemainingValue: 0,
	}
	if err := repo.CreateMove(ctx, move); err != nil {
		t.Fatalf("CreateMove failed: %v", err)
	}

	lc := &stock.LandedCost{
		ID:         1,
		Name:       "LC/2026/00001",
		Date:       time.Now().UTC(),
		State:      stock.LandedCostDraft,
		PickingIDs: []int64{1},
		JournalID:  6,
		CompanyID:  1,
		CostLines: []stock.LandedCostLine{{
			ID:           1,
			LandedCostID: 1,
			Name:         "Freight",
			AccountID:    17,
			PriceUnit:    10,
			SplitMethod:  stock.SplitEqual,
		}},
		ValuationAdjustments: []stock.ValuationAdjustment{{
			ID:               1,
			LandedCostID:     1,
			CostLineID:       1,
			MoveID:           1,
			ProductID:        10,
			Quantity:         10,
			Weight:           0,
			Volume:           0,
			FormerCost:       5,
			AdditionalCost:   1,
			FinalCost:        6,
			MoveRemainingQty: 10,
		}},
	}
	if err := repo.CreateLandedCost(ctx, lc); err != nil {
		t.Fatalf("CreateLandedCost failed: %v", err)
	}

	validated, err := uc.ValidateLandedCost(ctx, lc.ID)
	if err != nil {
		t.Fatalf("ValidateLandedCost failed: %v", err)
	}
	if validated.AccountMoveID == nil || *validated.AccountMoveID != 1 {
		t.Fatal("expected validation to create a posted journal entry for the landed-cost revaluation")
	}
	updatedMove, err := repo.GetMoveByID(ctx, move.ID)
	if err != nil {
		t.Fatalf("GetMoveByID failed: %v", err)
	}
	if updatedMove.RemainingValue != 1 {
		t.Fatalf("expected move remaining value to reflect the booked landed-cost amount, got %v", updatedMove.RemainingValue)
	}
}

func TestCreateLandedCostFromVendorBillDraftsCostLines(t *testing.T) {
	ctx := context.Background()
	companyRepo := companystorage.NewMemoryRepo()
	journalID := int64(55)
	company := &companydomain.Company{
		Name:               "LandedCostCo",
		CurrencyID:         1,
		LandedCostJournalID: &journalID,
	}
	if err := companyRepo.Create(ctx, company); err != nil {
		t.Fatalf("Create company failed: %v", err)
	}

	productRepo := productstorage.NewMemoryRepo()
	stockAccount := int64(5)
	priceDiffAccount := int64(17)
	pt := &product.ProductTemplate{
		Name:                     "Freight Item",
		Type:                     product.ProductTypeGoods,
		CostPrice:                10,
		SplitMethodLandedCost:    stock.SplitEqual,
		LandedCostOK:             true,
		StockValuationAccountID:  &stockAccount,
		PriceDifferenceAccountID: &priceDiffAccount,
	}
	if err := productRepo.CreateTemplate(ctx, pt); err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}

	bill := &accounting.AccountMove{
		ID:       7,
		MoveType: accounting.MoveTypeInInvoice,
		JournalID: 2,
		Lines: []accounting.AccountMoveLine{{
			AccountID: 6,
			ProductID: &pt.ID,
			Name:      "Freight line",
			Debit:     120,
			Credit:    0,
			Balance:   120,
		}},
	}

	uc := New(stockstorage.NewMemoryRepo(), nil, productRepo, nil, nil, companyRepo, fakeBillGateway{move: bill})
	lc, err := uc.CreateLandedCostFromVendorBill(ctx, CreateLandedCostFromBillInput{
		VendorBillID: 7,
		PickingIDs:   []int64{99},
		CompanyID:    company.ID,
	})
	if err != nil {
		t.Fatalf("CreateLandedCostFromVendorBill failed: %v", err)
	}
	if lc.JournalID != journalID {
		t.Fatalf("expected vendor-bill landed cost to use company default journal %d, got %d", journalID, lc.JournalID)
	}
	if len(lc.CostLines) != 1 {
		t.Fatalf("expected 1 landed-cost line from vendor bill, got %d", len(lc.CostLines))
	}
	if lc.CostLines[0].PriceUnit != 120 {
		t.Fatalf("expected landed-cost price to be 120, got %v", lc.CostLines[0].PriceUnit)
	}
}

type fakeBillGateway struct {
	move *accounting.AccountMove
}

func (f fakeBillGateway) CreateJournalEntry(context.Context, accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: 1}, nil
}

func (f fakeBillGateway) PostMove(context.Context, int64) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: 1}, nil
}

func (f fakeBillGateway) GetMove(context.Context, int64) (*accounting.AccountMove, error) {
	if f.move == nil {
		return nil, nil
	}
	return f.move, nil
}
