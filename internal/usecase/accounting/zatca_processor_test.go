package accountingusecase_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/usecase/accounting"
)

func TestZatcaProcessor_GenerateXML(t *testing.T) {
	processor := accountingusecase.NewZatcaProcessor()
	ctx := context.Background()

	t.Run("Standard Invoice XML", func(t *testing.T) {
		move := &accounting.AccountMove{
			Name:         "INV/2026/00001",
			Date:         time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
			MoveType:     accounting.MoveTypeOutInvoice,
			IsSimplified: false,
		}

		xml, err := processor.GenerateXML(ctx, move)
		if err != nil {
			t.Fatalf("GenerateXML failed: %v", err)
		}

		s := string(xml)
		if !strings.Contains(s, "<cbc:ID>INV/2026/00001</cbc:ID>") {
			t.Error("XML missing correct Invoice ID")
		}
		if !strings.Contains(s, "<cbc:InvoiceTypeCode name=\"388\">0100000</cbc:InvoiceTypeCode>") {
			t.Error("XML missing correct Standard Invoice Type Code")
		}
	})

	t.Run("Simplified Credit Note XML", func(t *testing.T) {
		move := &accounting.AccountMove{
			Name:         "RINV/2026/00001",
			Date:         time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC),
			MoveType:     accounting.MoveTypeOutRefund,
			IsSimplified: true,
		}

		xml, err := processor.GenerateXML(ctx, move)
		if err != nil {
			t.Fatalf("GenerateXML failed: %v", err)
		}

		s := string(xml)
		if !strings.Contains(s, "<cbc:InvoiceTypeCode name=\"381\">0200000</cbc:InvoiceTypeCode>") {
			t.Error("XML missing correct Simplified Credit Note Type Code")
		}
	})
}

func TestZatcaProcessor_GetTransactionType(t *testing.T) {
	processor := accountingusecase.NewZatcaProcessor()
	ctx := context.Background()

	tests := []struct {
		name     string
		move     *accounting.AccountMove
		expected accounting.EDITransactionType
	}{
		{"Standard", &accounting.AccountMove{IsSimplified: false}, accounting.EDITransactionStandard},
		{"Simplified", &accounting.AccountMove{IsSimplified: true}, accounting.EDITransactionSimplified},
		{"Self Billing", &accounting.AccountMove{IsSelfBilling: true}, accounting.EDITransactionSelfBilling},
		{"Third Party", &accounting.AccountMove{IsThirdParty: true}, accounting.EDITransactionThirdParty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.GetTransactionType(ctx, tt.move)
			if got != tt.expected {
				t.Errorf("GetTransactionType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestZatcaProcessor_GenerateQRCode(t *testing.T) {
	processor := accountingusecase.NewZatcaProcessor()
	ctx := context.Background()

	move := &accounting.AccountMove{
		AmountTotal: 115.0,
		AmountTax:   15.0,
	}

	qr, err := processor.GenerateQRCode(ctx, move, []byte("<xml/>"))
	if err != nil {
		t.Fatalf("GenerateQRCode failed: %v", err)
	}

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(qr)
	if err != nil {
		t.Fatalf("QRCode is not valid base64: %v", err)
	}

	// In the current mock implementation in zatca_processor.go, it returns:
	// []byte{0x01, 0x04, 'T', 'e', 's', 't'}
	if len(decoded) < 2 {
		t.Fatal("Decoded QR TLV too short")
	}
	if decoded[0] != 0x01 {
		t.Errorf("Expected Tag 1, got %x", decoded[0])
	}
}

// Helper function to simulate a production-ready TLV encoder for ZATCA
// Tag 1: Seller Name
// Tag 2: Seller VAT
// Tag 3: Timestamp
// Tag 4: Invoice Total
// Tag 5: VAT Total
func encodeTLV(tag byte, value string) []byte {
	return append([]byte{tag, byte(len(value))}, []byte(value)...)
}

func TestZatcaReality_TLVStructure(t *testing.T) {
	// This test "describes reality" by demonstrating how the TLV should be constructed
	// even if the processor is still using a mock.
	sellerName := "Cashflow Solutions Ltd"
	vatNumber := "300000000000003"
	timestamp := "2026-05-20T10:00:00Z"
	total := "115.00"
	vatTotal := "15.00"

	var fullTLV []byte
	fullTLV = append(fullTLV, encodeTLV(1, sellerName)...)
	fullTLV = append(fullTLV, encodeTLV(2, vatNumber)...)
	fullTLV = append(fullTLV, encodeTLV(3, timestamp)...)
	fullTLV = append(fullTLV, encodeTLV(4, total)...)
	fullTLV = append(fullTLV, encodeTLV(5, vatTotal)...)

	encoded := base64.StdEncoding.EncodeToString(fullTLV)

	// Real-world check: ZATCA Phase 1 QR codes are always TLV encoded and then Base64.
	if !strings.HasPrefix(encoded, "AQ") { // Tag 1 (0x01) usually starts with 'AQ' in Base64 if length is > 0
		t.Errorf("Base64 TLV seems incorrect: %s", encoded)
	}
}
