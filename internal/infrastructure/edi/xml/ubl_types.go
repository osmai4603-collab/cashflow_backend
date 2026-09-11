package xml

import "encoding/xml"

type UBLInvoice struct {
	XMLName              xml.Name              `xml:"Invoice"`
	XMLNs                string                `xml:"xmlns,attr"`
	XMLNsCac             string                `xml:"xmlns:cac,attr"`
	XMLNsCbc             string                `xml:"xmlns:cbc,attr"`
	ProfileID            string                `xml:"cbc:ProfileID"`
	ID                   string                `xml:"cbc:ID"`
	UUID                 string                `xml:"cbc:UUID"`
	IssueDate            string                `xml:"cbc:IssueDate"`
	IssueTime            string                `xml:"cbc:IssueTime"`
	InvoiceTypeCode      string                `xml:"cbc:InvoiceTypeCode"`
	DocumentCurrencyCode string                `xml:"cbc:DocumentCurrencyCode"`
	TaxTotal             UBLTaxTotal           `xml:"cac:TaxTotal"`
	LegalMonetaryTotal   UBLLegalMonetaryTotal `xml:"cac:LegalMonetaryTotal"`
	InvoiceLines         []UBLInvoiceLine      `xml:"cac:InvoiceLine"`
}

type UBLTaxTotal struct {
	TaxAmount UBLAmount `xml:"cbc:TaxAmount"`
}
type UBLLegalMonetaryTotal struct {
	LineExtension UBLAmount `xml:"cbc:LineExtensionAmount"`
	TaxExclusive  UBLAmount `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusive  UBLAmount `xml:"cbc:TaxInclusiveAmount"`
	Payable       UBLAmount `xml:"cbc:PayableAmount"`
}
type UBLAmount struct {
	CurrencyID string `xml:"currencyID,attr"`
	Value      string `xml:",chardata"`
}
type UBLInvoiceLine struct {
	ID            string    `xml:"cbc:ID"`
	Quantity      string    `xml:"cbc:InvoicedQuantity"`
	LineExtension UBLAmount `xml:"cbc:LineExtensionAmount"`
	Item          UBLItem   `xml:"cac:Item"`
}
type UBLItem struct {
	Name string `xml:"cbc:Name"`
}
