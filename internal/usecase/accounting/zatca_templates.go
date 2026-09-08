package accountingusecase

const zatcaUBLTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
         xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2">
    <cbc:ProfileID>reporting:1.0</cbc:ProfileID>
    <cbc:ID>{{.InvoiceNumber}}</cbc:ID>
    <cbc:UUID>{{.UUID}}</cbc:UUID>
    <cbc:IssueDate>{{.IssueDate}}</cbc:IssueDate>
    <cbc:IssueTime>{{.IssueTime}}</cbc:IssueTime>
    <cbc:InvoiceTypeCode name="{{.InvoiceTypeName}}">{{.InvoiceTypeCode}}</cbc:InvoiceTypeCode>
    <cbc:DocumentCurrencyCode>{{.Currency}}</cbc:DocumentCurrencyCode>
    <cbc:TaxCurrencyCode>{{.Currency}}</cbc:TaxCurrencyCode>
    <cac:AdditionalDocumentReference>
        <cbc:ID>ICV</cbc:ID>
        <cbc:UUID>{{.InvoiceCounter}}</cbc:UUID>
    </cac:AdditionalDocumentReference>
    <cac:AdditionalDocumentReference>
        <cbc:ID>PIH</cbc:ID>
        <cac:Attachment>
            <cbc:EmbeddedDocumentBinaryObject mimeCode="text/plain">{{.PreviousHash}}</cbc:EmbeddedDocumentBinaryObject>
        </cac:Attachment>
    </cac:AdditionalDocumentReference>
    <cac:AccountingSupplierParty>
        <cac:Party>
            <cac:PartyIdentification>
                <cbc:ID schemeID="CRN">{{.SellerCRN}}</cbc:ID>
            </cac:PartyIdentification>
            <cac:PostalAddress>
                <cbc:StreetName>{{.SellerStreet}}</cbc:StreetName>
                <cbc:BuildingNumber>{{.SellerBuildingNo}}</cbc:BuildingNumber>
                <cbc:CityName>{{.SellerCity}}</cbc:CityName>
                <cbc:PostalZone>{{.SellerPostalCode}}</cbc:PostalZone>
                <cbc:CountrySubentity>{{.SellerState}}</cbc:CountrySubentity>
                <cac:Country>
                    <cbc:IdentificationCode>SA</cbc:IdentificationCode>
                </cac:Country>
            </cac:PostalAddress>
            <cac:PartyTaxScheme>
                <cbc:CompanyID>{{.SellerVAT}}</cbc:CompanyID>
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:PartyTaxScheme>
            <cac:PartyLegalEntity>
                <cbc:RegistrationName>{{.SellerName}}</cbc:RegistrationName>
            </cac:PartyLegalEntity>
        </cac:Party>
    </cac:AccountingSupplierParty>
    <cac:AccountingCustomerParty>
        <cac:Party>
            <cac:PostalAddress>
                <cbc:StreetName>{{.BuyerStreet}}</cbc:StreetName>
                <cbc:BuildingNumber>{{.BuyerBuildingNo}}</cbc:BuildingNumber>
                <cbc:CityName>{{.BuyerCity}}</cbc:CityName>
                <cbc:PostalZone>{{.BuyerPostalCode}}</cbc:PostalZone>
                <cac:Country>
                    <cbc:IdentificationCode>SA</cbc:IdentificationCode>
                </cac:Country>
            </cac:PostalAddress>
            <cac:PartyTaxScheme>
                {{if .BuyerVAT}}<cbc:CompanyID>{{.BuyerVAT}}</cbc:CompanyID>{{end}}
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:PartyTaxScheme>
            <cac:PartyLegalEntity>
                <cbc:RegistrationName>{{.BuyerName}}</cbc:RegistrationName>
            </cac:PartyLegalEntity>
        </cac:Party>
    </cac:AccountingCustomerParty>
    <cac:Delivery>
        <cbc:ActualDeliveryDate>{{.IssueDate}}</cbc:ActualDeliveryDate>
    </cac:Delivery>
    <cac:PaymentMeans>
        <cbc:PaymentMeansCode>10</cbc:PaymentMeansCode>
    </cac:PaymentMeans>
    {{range .TaxTotals}}
    <cac:TaxTotal>
        <cbc:TaxAmount currencyID="{{$.Currency}}">{{.Amount}}</cbc:TaxAmount>
        {{range .Subtotals}}
        <cac:TaxSubtotal>
            <cbc:TaxableAmount currencyID="{{$.Currency}}">{{.TaxableAmount}}</cbc:TaxableAmount>
            <cbc:TaxAmount currencyID="{{$.Currency}}">{{.TaxAmount}}</cbc:TaxAmount>
            <cac:TaxCategory>
                <cbc:ID>{{.TaxCategoryCode}}</cbc:ID>
                <cbc:Percent>{{.TaxPercent}}</cbc:Percent>
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:TaxCategory>
        </cac:TaxSubtotal>
        {{end}}
    </cac:TaxTotal>
    {{end}}
    <cac:LegalMonetaryTotal>
        <cbc:LineExtensionAmount currencyID="{{.Currency}}">{{.AmountUntaxed}}</cbc:LineExtensionAmount>
        <cbc:TaxExclusiveAmount currencyID="{{.Currency}}">{{.AmountUntaxed}}</cbc:TaxExclusiveAmount>
        <cbc:TaxInclusiveAmount currencyID="{{.Currency}}">{{.AmountTotal}}</cbc:TaxInclusiveAmount>
        <cbc:PayableAmount currencyID="{{.Currency}}">{{.AmountTotal}}</cbc:PayableAmount>
    </cac:LegalMonetaryTotal>
    {{range .Lines}}
    <cac:InvoiceLine>
        <cbc:ID>{{.ID}}</cbc:ID>
        <cbc:InvoicedQuantity unitCode="PCE">{{.Quantity}}</cbc:InvoicedQuantity>
        <cbc:LineExtensionAmount currencyID="{{$.Currency}}">{{.LineExtensionAmount}}</cbc:LineExtensionAmount>
        <cac:TaxTotal>
            <cbc:TaxAmount currencyID="{{$.Currency}}">{{.TaxAmount}}</cbc:TaxAmount>
            <cbc:RoundingAmount currencyID="{{$.Currency}}">{{.TaxInclusiveAmount}}</cbc:RoundingAmount>
        </cac:TaxTotal>
        <cac:Item>
            <cbc:Name>{{.Name}}</cbc:Name>
            <cac:ClassifiedTaxCategory>
                <cbc:ID>{{.TaxCategoryCode}}</cbc:ID>
                <cbc:Percent>{{.TaxPercent}}</cbc:Percent>
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:ClassifiedTaxCategory>
        </cac:Item>
        <cac:Price>
            <cbc:PriceAmount currencyID="{{$.Currency}}">{{.PriceUnit}}</cbc:PriceAmount>
        </cac:Price>
    </cac:InvoiceLine>
    {{end}}
</Invoice>`
