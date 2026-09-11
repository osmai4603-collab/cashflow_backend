package stock

import "testing"

func TestPackagingValidation(t *testing.T) {
	packaging := &ProductPackaging{Name: "Box of 12", ProductID: 10, Qty: 12}
	if err := packaging.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (&ProductPackaging{Name: "", ProductID: 10, Qty: 0}).Validate(); err == nil {
		t.Fatal("expected invalid packaging")
	}
}

func TestBarcodeParserAndGS1(t *testing.T) {
	parser := NewBarcodeParser(&BarcodeNomenclature{Rules: []BarcodeRule{{Type: BarcodeRuleProduct, Pattern: `^P-[0-9]+$`}}})
	result, err := parser.Parse("P-123")
	if err != nil || result.Type != BarcodeRuleProduct {
		t.Fatalf("unexpected barcode result: %+v, %v", result, err)
	}
	gs1, err := (&GS1Parser{}).ParseGS1("(01)09501101530003(17)260930(10)BATCH123")
	if err != nil || len(gs1) != 3 || gs1[2].Value != "BATCH123" {
		t.Fatalf("unexpected GS1 result: %+v, %v", gs1, err)
	}
}
