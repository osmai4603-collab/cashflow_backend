package edi

import (
	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/infrastructure/edi/signing"
	"context"
	"testing"
	"time"
)

func TestServiceGeneratesValidatedEDI(t *testing.T) {
	certificate, privateKey, err := signing.GenerateTestCertificate()
	if err != nil {
		t.Fatal(err)
	}
	move := &accounting.AccountMove{ID: 7, Name: "INV/2026/0001", Date: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), Currency: "SAR", AmountUntaxed: 100, AmountTax: 15, AmountTotal: 115, Lines: []accounting.AccountMoveLine{{Name: "Product", Quantity: 1, Balance: 100}}}
	service, err := NewService()
	if err != nil {
		t.Fatal(err)
	}
	document, err := service.Generate(context.Background(), move, accounting.EDIFormatZatcaPhase2, &accounting.EDICertificate{CertContent: certificate, PrivateKey: privateKey})
	if err != nil {
		t.Fatal(err)
	}
	if document.Hash == "" || document.QRCode == "" || len(document.XMLContent) == 0 {
		t.Fatalf("incomplete EDI document: %+v", document)
	}
}
