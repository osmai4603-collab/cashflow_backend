package edi

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/infrastructure/edi/hash"
	"cashflow_backend/internal/infrastructure/edi/qrcode"
	"cashflow_backend/internal/infrastructure/edi/signing"
	edixml "cashflow_backend/internal/infrastructure/edi/xml"
	"cashflow_backend/internal/infrastructure/edi/xsd"
)

type Service struct {
	validator *xsd.XSDValidator
	signer    *signing.XAdESSigner
}

func NewService() (*Service, error) {
	validator, err := xsd.NewXSDValidator()
	if err != nil {
		return nil, err
	}
	return &Service{validator: validator, signer: &signing.XAdESSigner{}}, nil
}

func (s *Service) Generate(ctx context.Context, move *accounting.AccountMove, format accounting.EDIFormat, certificate *accounting.EDICertificate) (*accounting.EDIDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	content, err := edixml.NewUBLInvoiceBuilder(move, format).Build()
	if err != nil {
		return nil, err
	}
	if err := s.validator.Validate(content, format); err != nil {
		return nil, fmt.Errorf("UBL validation failed: %w", err)
	}
	digest, err := hash.ComputeInvoiceHashBase64(content)
	if err != nil {
		return nil, err
	}
	signed := content
	if certificate != nil {
		signed, err = s.signer.Sign(content, certificate)
		if err != nil {
			return nil, err
		}
	}
	qr, err := qrcode.EncodeTLVBase64(map[qrcode.TLVTag]string{qrcode.TLVInvoiceHash: digest})
	if err != nil {
		return nil, err
	}
	transactionType := accounting.EDITransactionStandard
	if move.IsSimplified {
		transactionType = accounting.EDITransactionSimplified
	}
	return &accounting.EDIDocument{MoveID: move.ID, Format: format, TransactionType: transactionType, State: accounting.EDIStateToSend, XMLContent: signed, Hash: digest, QRCode: qr}, nil
}
