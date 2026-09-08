package accountingusecase_test

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/usecase/accounting"
)

type mockEDIProcessor struct {
	genXMLCalled bool
}

func (m *mockEDIProcessor) GenerateXML(ctx context.Context, move *accounting.AccountMove) ([]byte, error) {
	m.genXMLCalled = true
	return []byte("<xml/>"), nil
}

func (m *mockEDIProcessor) ValidateXML(ctx context.Context, xmlContent []byte) error {
	return nil
}

func (m *mockEDIProcessor) SignXML(ctx context.Context, xmlContent []byte, cert *accounting.EDICertificate) ([]byte, error) {
	return xmlContent, nil
}

func (m *mockEDIProcessor) GenerateQRCode(ctx context.Context, move *accounting.AccountMove, xmlContent []byte) (string, error) {
	return "qr-code", nil
}

func (m *mockEDIProcessor) GetTransactionType(ctx context.Context, move *accounting.AccountMove) accounting.EDITransactionType {
	return accounting.EDITransactionStandard
}

func (m *mockEDIProcessor) EmbedXMLInPDF(ctx context.Context, pdfPath string, xmlContent []byte) error {
	return nil
}

func TestUseCase_RegisterEDIProcessor(t *testing.T) {
	uc := accountingusecase.New(nil, nil)
	mock := &mockEDIProcessor{}

	format := accounting.EDIFormat("test_format")
	uc.RegisterEDIProcessor(format, mock)

	// Since ediProcessors is private, we'd ideally test a usecase method that uses it.
	// For now, this confirms the registration interface is available.
}
