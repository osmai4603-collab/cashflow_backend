package accountingusecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"text/template"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/domain/partner"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ZatcaProcessor implements the EDIProcessor interface for Saudi ZATCA requirements.
type ZatcaProcessor struct {
	companyRepo company.Repository
	partnerRepo partner.Repository
}

// NewZatcaProcessor creates an instance of the ZATCA EDI processor.
func NewZatcaProcessor(repositories ...any) *ZatcaProcessor {
	processor := &ZatcaProcessor{}
	if len(repositories) > 0 {
		processor.companyRepo, _ = repositories[0].(company.Repository)
	}
	if len(repositories) > 1 {
		processor.partnerRepo, _ = repositories[1].(partner.Repository)
	}
	return processor
}

// GenerateXML constructs the UBL 2.1 XML for ZATCA using templates.
func (p *ZatcaProcessor) GenerateXML(ctx context.Context, move *accounting.AccountMove) ([]byte, error) {
	data, err := p.prepareTemplateData(ctx, move)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("zatca").Parse(zatcaUBLTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ZATCA template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute ZATCA template: %w", err)
	}

	return buf.Bytes(), nil
}

type ZatcaTemplateData struct {
	InvoiceNumber    string
	UUID             string
	IssueDate        string
	IssueTime        string
	InvoiceTypeName  string
	InvoiceTypeCode  string
	Currency         string
	InvoiceCounter   string
	PreviousHash     string
	SellerCRN        string
	SellerStreet     string
	SellerBuildingNo string
	SellerCity       string
	SellerPostalCode string
	SellerState      string
	SellerVAT        string
	SellerName       string
	BuyerName        string
	BuyerVAT         string
	BuyerStreet      string
	BuyerBuildingNo  string
	BuyerCity        string
	BuyerPostalCode  string
	AmountUntaxed    string
	AmountTotal      string
	TaxTotals        []ZatcaTaxTotal
	Lines            []ZatcaLine
}

type ZatcaTaxTotal struct {
	Amount    string
	Subtotals []ZatcaTaxSubtotal
}

type ZatcaTaxSubtotal struct {
	TaxableAmount  string
	TaxAmount      string
	TaxCategoryCode string
	TaxPercent     string
}

type ZatcaLine struct {
	ID                  int64
	Name                string
	Quantity            string
	LineExtensionAmount string
	TaxAmount           string
	TaxInclusiveAmount  string
	TaxCategoryCode     string
	TaxPercent          string
	PriceUnit           string
}

func (p *ZatcaProcessor) prepareTemplateData(ctx context.Context, move *accounting.AccountMove) (*ZatcaTemplateData, error) {
	// Fetch company (Seller)
	// In a real multi-tenant app, move would have CompanyID. For now assume company 1.
	comp := &company.Company{Name: "Default Company"}
	if p.companyRepo != nil {
		loaded, err := p.companyRepo.GetByID(ctx, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get seller company: %w", err)
		}
		comp = loaded
	}

	// Fetch partner (Buyer)
	var buy partner.Partner
	if move.PartnerID != nil {
		if p.partnerRepo != nil {
			b, err := p.partnerRepo.GetByID(ctx, *move.PartnerID)
			if err == nil && b != nil {
				buy = *b
			}
		}
	}

	data := &ZatcaTemplateData{
		InvoiceNumber:   move.Name,
		UUID:            fmt.Sprintf("%d", move.ID), // Should be real UUID
		IssueDate:       move.Date.Format("2006-01-02"),
		IssueTime:       move.Date.Format("15:04:05"),
		Currency:        move.Currency,
		InvoiceCounter:  "1",
		PreviousHash:    "NWZlY2ViOTZmOTk1YTM1NzAzNzAxNzQyOWQ5MDljNzgzYzlkM2Q2NzkwMDNkZDVlMzcwOGlkZTZhMGNmYWQwZQ==",
		SellerName:      comp.Name,
		SellerVAT:       comp.VAT,
		SellerStreet:    comp.Street,
		SellerCity:      comp.City,
		SellerPostalCode: comp.ZipCode,
		SellerState:     comp.State,
		BuyerName:       buy.Name,
		BuyerVAT:        buy.VATNumber,
		BuyerStreet:     buy.Street,
		BuyerCity:       buy.City,
		BuyerPostalCode: buy.ZipCode,
		AmountUntaxed:   fmt.Sprintf("%.2f", move.AmountUntaxed),
		AmountTotal:     fmt.Sprintf("%.2f", move.AmountTotal),
	}

	data.InvoiceTypeCode = "0100000"
	data.InvoiceTypeName = "388"
	if move.IsSimplified {
		data.InvoiceTypeCode = "0200000"
	}
	if move.MoveType == accounting.MoveTypeOutRefund {
		data.InvoiceTypeName = "381"
	}

	// Simple mapping for lines
	for i, l := range move.Lines {
		if l.DisplayType != "" || (l.Debit == 0 && l.Credit == 0) {
			continue
		}
		data.Lines = append(data.Lines, ZatcaLine{
			ID:                  int64(i + 1),
			Name:                l.Name,
			Quantity:            fmt.Sprintf("%.2f", l.Quantity),
			LineExtensionAmount: fmt.Sprintf("%.2f", l.AbsBalance()-l.TaxAmount),
			TaxAmount:           fmt.Sprintf("%.2f", l.TaxAmount),
			TaxInclusiveAmount:  fmt.Sprintf("%.2f", l.AbsBalance()),
			TaxCategoryCode:     "S", // Standard
			TaxPercent:          "15.00",
			PriceUnit:           fmt.Sprintf("%.2f", l.PriceUnit),
		})
	}

	// Populate TaxTotals (Simplified for now)
	data.TaxTotals = []ZatcaTaxTotal{
		{
			Amount: fmt.Sprintf("%.2f", move.AmountTax),
			Subtotals: []ZatcaTaxSubtotal{
				{
					TaxableAmount:   fmt.Sprintf("%.2f", move.AmountUntaxed),
					TaxAmount:       fmt.Sprintf("%.2f", move.AmountTax),
					TaxCategoryCode: "S",
					TaxPercent:      "15.00",
				},
			},
		},
	}

	return data, nil
}

// ValidateXML performs XSD and Schematron validation.
func (p *ZatcaProcessor) ValidateXML(ctx context.Context, xmlContent []byte) error {
	// TODO: Integrate with a library that supports Schematron/XSLT validation.
	// For now, simple check for root element
	if !bytes.Contains(xmlContent, []byte("<Invoice")) && !bytes.Contains(xmlContent, []byte("<CreditNote")) {
		return platformerrors.Validation("invalid ZATCA XML: root element not found", nil)
	}
	return nil
}

// SignXML performs digital signature (XAdES) for ZATCA Phase 2.
func (p *ZatcaProcessor) SignXML(ctx context.Context, xmlContent []byte, cert *accounting.EDICertificate) ([]byte, error) {
	// TODO: Implement XML canonicalization (C14N) and SHA-256 signing.
	// This requires a full XAdES implementation. For now, returning as-is with a log.
	fmt.Printf("Signing XML with certificate: %s\n", cert.Name)
	return xmlContent, nil
}

// GenerateQRCode creates the TLV-encoded Base64 QR code required by ZATCA.
func (p *ZatcaProcessor) GenerateQRCode(ctx context.Context, move *accounting.AccountMove, xmlContent []byte) (string, error) {
	comp := &company.Company{Name: "Default Company"}
	if p.companyRepo != nil {
		loaded, err := p.companyRepo.GetByID(ctx, 1)
		if err != nil {
			return "", err
		}
		comp = loaded
	}

	tlv := new(bytes.Buffer)
	p.writeTLV(tlv, 1, comp.Name)
	p.writeTLV(tlv, 2, comp.VAT)
	p.writeTLV(tlv, 3, move.Date.Format(time.RFC3339))
	p.writeTLV(tlv, 4, fmt.Sprintf("%.2f", move.AmountTotal))
	p.writeTLV(tlv, 5, fmt.Sprintf("%.2f", move.AmountTax))

	// Phase 2: Tag 6 (Hash), Tag 7 (Signature), Tag 8 (PublicKey), Tag 9 (Certificate Signature)
	hash := sha256.Sum256(xmlContent)
	p.writeTLV(tlv, 6, base64.StdEncoding.EncodeToString(hash[:]))

	return base64.StdEncoding.EncodeToString(tlv.Bytes()), nil
}

func (p *ZatcaProcessor) writeTLV(buf *bytes.Buffer, tag byte, value string) {
	buf.WriteByte(tag)
	buf.WriteByte(byte(len(value)))
	buf.WriteString(value)
}

// GetTransactionType determines the ZATCA transaction category.
func (p *ZatcaProcessor) GetTransactionType(ctx context.Context, move *accounting.AccountMove) accounting.EDITransactionType {
	if move.IsSelfBilling {
		return accounting.EDITransactionSelfBilling
	}
	if move.IsThirdParty {
		return accounting.EDITransactionThirdParty
	}
	if move.IsSimplified {
		return accounting.EDITransactionSimplified
	}
	return accounting.EDITransactionStandard
}

// EmbedXMLInPDF placeholder for attaching XML to a PDF.
func (p *ZatcaProcessor) EmbedXMLInPDF(ctx context.Context, pdfPath string, xmlContent []byte) error {
	// TODO: Use a PDF library (like gofpdf or similar) to attach the XML.
	return nil
}
