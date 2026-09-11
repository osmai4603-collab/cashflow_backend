package xsd

import (
	"cashflow_backend/internal/domain/accounting"
	"testing"
)

func TestValidatorRequiresUBLInvoiceFields(t *testing.T) {
	validator, err := NewXSDValidator(); if err != nil { t.Fatal(err) }
	valid := []byte(`<Invoice><ID>INV-1</ID><IssueDate>2026-01-01</IssueDate><DocumentCurrencyCode>SAR</DocumentCurrencyCode><LegalMonetaryTotal/></Invoice>`)
	if err := validator.Validate(valid, accounting.EDIFormatUBL21); err != nil { t.Fatal(err) }
	if err := validator.Validate([]byte(`<Invoice><ID>INV-1</ID></Invoice>`), accounting.EDIFormatUBL21); err == nil { t.Fatal("expected missing UBL fields to fail") }
	if err := validator.Validate([]byte(`<CreditNote/>`), accounting.EDIFormatUBL21); err == nil { t.Fatal("expected invalid root to fail") }
}