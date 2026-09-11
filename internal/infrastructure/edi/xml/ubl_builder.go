package xml

import (
	"encoding/xml"
	"fmt"

	"cashflow_backend/internal/domain/accounting"
)

type UBLInvoiceBuilder struct {
	Invoice *accounting.AccountMove
	Format  accounting.EDIFormat
}

func NewUBLInvoiceBuilder(invoice *accounting.AccountMove, format accounting.EDIFormat) *UBLInvoiceBuilder {
	return &UBLInvoiceBuilder{Invoice: invoice, Format: format}
}

func (b *UBLInvoiceBuilder) Build() ([]byte, error) {
	if b == nil || b.Invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}
	move := b.Invoice
	currency := move.Currency
	if currency == "" {
		currency = "SAR"
	}
	result := UBLInvoice{XMLNs: "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2", XMLNsCac: "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2", XMLNsCbc: "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2", ProfileID: "reporting:1.0", ID: move.Name, UUID: fmt.Sprintf("%d", move.ID), IssueDate: move.Date.Format("2006-01-02"), IssueTime: move.Date.Format("15:04:05"), InvoiceTypeCode: "388", DocumentCurrencyCode: currency}
	result.TaxTotal = UBLTaxTotal{TaxAmount: UBLAmount{CurrencyID: currency, Value: fmt.Sprintf("%.2f", move.AmountTax)}}
	result.LegalMonetaryTotal = UBLLegalMonetaryTotal{LineExtension: UBLAmount{CurrencyID: currency, Value: fmt.Sprintf("%.2f", move.AmountUntaxed)}, TaxExclusive: UBLAmount{CurrencyID: currency, Value: fmt.Sprintf("%.2f", move.AmountUntaxed)}, TaxInclusive: UBLAmount{CurrencyID: currency, Value: fmt.Sprintf("%.2f", move.AmountTotal)}, Payable: UBLAmount{CurrencyID: currency, Value: fmt.Sprintf("%.2f", move.AmountTotal)}}
	for index, line := range move.Lines {
		if line.DisplayType != "" {
			continue
		}
		result.InvoiceLines = append(result.InvoiceLines, UBLInvoiceLine{ID: fmt.Sprintf("%d", index+1), Quantity: fmt.Sprintf("%.4f", line.Quantity), LineExtension: UBLAmount{CurrencyID: currency, Value: fmt.Sprintf("%.2f", line.AbsBalance())}, Item: UBLItem{Name: line.Name}})
	}
	return xml.MarshalIndent(result, "", "  ")
}
