package accountingusecase

import (
	"context"
	"cashflow_backend/internal/domain/accounting"
)

// EDIProcessor defines the interface for generating and signing electronic documents.
// This interface allows for different implementations (ZATCA, PEPPOL, etc.)
type EDIProcessor interface {
	// GenerateXML creates the UBL XML content for the given move.
	GenerateXML(ctx context.Context, move *accounting.AccountMove) ([]byte, error)

	// ValidateXML checks the XML against the specific format's rules (XSD/Schematron).
	ValidateXML(ctx context.Context, xmlContent []byte) error

	// SignXML applies a digital signature to the XML.
	SignXML(ctx context.Context, xmlContent []byte, cert *accounting.EDICertificate) ([]byte, error)

	// GenerateQRCode generates the QR code content (Base64 for ZATCA).
	GenerateQRCode(ctx context.Context, move *accounting.AccountMove, xmlContent []byte) (string, error)

	// GetTransactionType determines the Odoo-style transaction type (Standard, Simplified, etc.)
	GetTransactionType(ctx context.Context, move *accounting.AccountMove) accounting.EDITransactionType

	// EmbedXMLInPDF attaches the XML content to a PDF file (Factur-X / Hybrid PDF).
	EmbedXMLInPDF(ctx context.Context, pdfPath string, xmlContent []byte) error
}
