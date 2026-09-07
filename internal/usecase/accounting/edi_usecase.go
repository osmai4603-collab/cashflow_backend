package accountingusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/accounting"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ProcessMoveEDI generates the electronic document for a posted move.
func (uc *UseCase) ProcessMoveEDI(ctx context.Context, moveID int64, format accounting.EDIFormat) (*accounting.EDIDocument, error) {
	move, err := uc.repo.GetMoveWithLines(ctx, moveID)
	if err != nil {
		return nil, err
	}

	if move.State != accounting.MoveStatePosted {
		return nil, platformerrors.Validation("move must be posted to generate EDI", nil)
	}

	processor, ok := uc.ediProcessors[format]
	if !ok {
		return nil, platformerrors.Validation("unsupported EDI format", map[string]string{
			"format": string(format),
		})
	}

	transType := processor.GetTransactionType(ctx, move)
	xml, err := processor.GenerateXML(ctx, move)
	if err != nil {
		return nil, fmt.Errorf("failed to generate XML: %w", err)
	}

	qr, _ := processor.GenerateQRCode(ctx, move, xml)

	doc := &accounting.EDIDocument{
		MoveID:          move.ID,
		Format:          format,
		TransactionType: transType,
		State:           accounting.EDIStateToSend,
		XMLContent:      xml,
		QRCode:          qr,
	}

	// For Phase 2 or local signing formats
	if format == accounting.EDIFormatZatcaPhase2 {
		// In a multi-tenant system, we would use the move's company ID.
		// For now, using a placeholder company ID 1.
		cert, err := uc.repo.GetActiveCertificate(ctx, 1)
		if err == nil && cert != nil {
			signed, err := processor.SignXML(ctx, xml, cert)
			if err == nil {
				doc.XMLContent = signed
				doc.State = accounting.EDIStateSent
				now := time.Now().UTC()
				doc.SentAt = &now
			}
		}
	}

	if err := uc.repo.CreateEDIDocument(ctx, doc); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "EDI document generated", "move_id", move.ID, "format", format, "type", transType)
	return doc, nil
}

// GetMoveEDIDocuments retrieves all EDI documents associated with a move.
func (uc *UseCase) GetMoveEDIDocuments(ctx context.Context, moveID int64) ([]accounting.EDIDocument, error) {
	return uc.repo.GetEDIDocumentsByMoveID(ctx, moveID)
}

// ─────────────────────────────────────────────────────────────────────────────
// EDI Certificate Management
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateEDICertificate(ctx context.Context, cert *accounting.EDICertificate) error {
	if cert.Name == "" {
		cert.Name = "Default Certificate"
	}
	cert.Active = true
	return uc.repo.CreateEDICertificate(ctx, cert)
}

func (uc *UseCase) GetActiveCertificate(ctx context.Context, companyID int64) (*accounting.EDICertificate, error) {
	return uc.repo.GetActiveCertificate(ctx, companyID)
}
