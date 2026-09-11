package paymentusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/payment"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// CreateTransaction initializes an external payment transaction.
func (uc *UseCase) CreateTransaction(ctx context.Context, providerCode string, amount float64, currency string, partnerID int64, companyID int64) (*payment.PaymentTransaction, error) {
	provider, err := uc.repo.GetProviderByCode(ctx, providerCode, companyID)
	if err != nil {
		return nil, err
	}

	tx := &payment.PaymentTransaction{
		Reference:  fmt.Sprintf("TXN-%d", time.Now().UnixNano()),
		Amount:     amount,
		Currency:   currency,
		ProviderID: provider.ID,
		PartnerID:  partnerID,
		State:      payment.TransactionStateDraft,
		CompanyID:  companyID,
	}

	if err := tx.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	if uc.providers != nil {
		gateway, err := uc.providers.Get(provider.Code)
		if err != nil {
			return nil, err
		}
		result, err := gateway.InitiatePayment(ctx, tx)
		if err != nil {
			tx.LastError = err.Error()
			_ = uc.repo.UpdateTransaction(ctx, tx)
			return nil, err
		}
		tx.ProviderReference = result.TransactionRef
		if err := tx.Transition(payment.TransactionStatePending); err != nil {
			return nil, err
		}
		if err := uc.repo.UpdateTransaction(ctx, tx); err != nil {
			return nil, err
		}
	}

	return tx, nil
}

func (uc *UseCase) CaptureTransaction(ctx context.Context, id int64) (*payment.PaymentTransaction, error) {
	return uc.callExternalTransaction(ctx, id, func(gateway payment.PaymentProviderInterface, tx *payment.PaymentTransaction) error {
		return gateway.CapturePayment(ctx, tx)
	}, payment.TransactionStateDone)
}

func (uc *UseCase) VoidTransaction(ctx context.Context, id int64) (*payment.PaymentTransaction, error) {
	return uc.callExternalTransaction(ctx, id, func(gateway payment.PaymentProviderInterface, tx *payment.PaymentTransaction) error {
		return gateway.VoidPayment(ctx, tx)
	}, payment.TransactionStateCancelled)
}

func (uc *UseCase) callExternalTransaction(ctx context.Context, id int64, call func(payment.PaymentProviderInterface, *payment.PaymentTransaction) error, state payment.TransactionState) (*payment.PaymentTransaction, error) {
	tx, err := uc.repo.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if uc.providers == nil {
		return nil, platformerrors.Internal("payment provider registry is not configured", nil)
	}
	providerRepo, ok := uc.repo.(interface {
		GetProviderByID(context.Context, int64) (*payment.PaymentProvider, error)
	})
	if !ok {
		return nil, platformerrors.Internal("provider lookup by ID is not configured", nil)
	}
	provider, err := providerRepo.GetProviderByID(ctx, tx.ProviderID)
	if err != nil {
		return nil, err
	}
	gateway, err := uc.providers.Get(provider.Code)
	if err != nil {
		return nil, err
	}
	if err := call(gateway, tx); err != nil {
		return nil, err
	}
	if err := tx.Transition(state); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// ProcessTransactionWebhook updates transaction state and creates internal payment on success.
func (uc *UseCase) ProcessTransactionWebhook(ctx context.Context, reference string, newState payment.TransactionState, providerRef string) error {
	tx, err := uc.repo.GetTransactionByReference(ctx, reference)
	if err != nil {
		return err
	}

	// Idempotency check: don't process already confirmed transactions
	if tx.State == payment.TransactionStateConfirmed {
		return nil
	}

	if err := tx.Transition(newState); err != nil {
		return err
	}
	tx.ProviderReference = providerRef
	tx.WebhookReceived = true
	tx.UpdatedAt = time.Now().UTC()

	if newState == payment.TransactionStateConfirmed {
		// Create internal account.payment automatically
		// Note: In a real system, we'd map providerCode to a specific bank journal.
		payInput := CreatePaymentInput{
			PartnerID:     tx.PartnerID,
			Amount:        tx.Amount,
			PaymentType:   payment.PaymentTypeInbound,
			PartnerType:   payment.PartnerTypeCustomer,
			PaymentMethod: payment.PaymentMethodBankTransfer,
			JournalID:     1, // Default bank journal
			Date:          time.Now().UTC(),
			Ref:           fmt.Sprintf("Online Payment Ref: %s (Provider: %s)", tx.Reference, providerRef),
			Currency:      tx.Currency,
			CompanyID:     &tx.CompanyID,
			AutoReconcile: true,
		}

		pay, err := uc.CreatePayment(ctx, payInput)
		if err == nil {
			tx.PaymentID = &pay.ID
		} else {
			// Log error but maybe don't fail the webhook if payment creation fails
			// We can retry payment creation later if tx is confirmed.
			uc.logger.ErrorContext(ctx, "failed to create internal payment for confirmed transaction", "tx_ref", tx.Reference, "error", err)
		}
	}

	return uc.repo.UpdateTransaction(ctx, tx)
}
