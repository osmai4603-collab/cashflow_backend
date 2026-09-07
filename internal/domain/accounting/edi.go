package accounting

import (
	"time"
)

// EDIFormat defines the electronic invoicing standard.
type EDIFormat string

const (
	EDIFormatZatcaPhase1 EDIFormat = "zatca_phase_1"
	EDIFormatZatcaPhase2 EDIFormat = "zatca_phase_2"
	EDIFormatUBL21       EDIFormat = "ubl_2_1"
	EDIFormatPeppol      EDIFormat = "peppol"
)

// EDIState represents the status of an electronic document.
type EDIState string

const (
	EDIStateToSend    EDIState = "to_send"
	EDIStateSent      EDIState = "sent"
	EDIStateToCancel  EDIState = "to_cancel"
	EDIStateCancelled EDIState = "cancelled"
	EDIStateError     EDIState = "error"
)

// EDITransactionType classifies the type of EDI transaction.
type EDITransactionType string

const (
	EDITransactionStandard    EDITransactionType = "standard"    // B2B
	EDITransactionSimplified  EDITransactionType = "simplified"  // B2C
	EDITransactionSelfBilling EDITransactionType = "self_billing"
	EDITransactionThirdParty  EDITransactionType = "third_party"
	EDITransactionExport      EDITransactionType = "export"
	EDITransactionNominal     EDITransactionType = "nominal"
	EDITransactionSummary     EDITransactionType = "summary"
)

// EDIDocument represents an electronic document linked to an AccountMove.
type EDIDocument struct {
	ID              int64              `json:"id"`
	MoveID          int64              `json:"move_id"`
	Format          EDIFormat          `json:"format"`
	TransactionType EDITransactionType `json:"transaction_type"`
	State           EDIState           `json:"state"`
	XMLContent      []byte             `json:"xml_content"`
	Hash            string             `json:"hash"` // Previous Invoice Hash (PIH) for chaining
	QRCode          string             `json:"qr_code"`
	ErrorMsg        string             `json:"error_msg,omitempty"`
	SentAt          *time.Time         `json:"sent_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// EDICertificate manages security artifacts for ZATCA Phase 2 signing.
type EDICertificate struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	CertContent    []byte    `json:"cert_content"` // Public Certificate
	PrivateKey     []byte    `json:"private_key"`  // Encrypted Private Key
	PublicKey      []byte    `json:"public_key"`
	CSR            string    `json:"csr"`    // Certificate Signing Request
	CSID           string    `json:"csid"`   // Binary Security Token / CSID
	Secret         string    `json:"secret"` // Shared Secret / Password
	CompanyID      int64     `json:"company_id"`
	IsProduction   bool      `json:"is_production"`
	ExpirationDate time.Time `json:"expiration_date"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}
