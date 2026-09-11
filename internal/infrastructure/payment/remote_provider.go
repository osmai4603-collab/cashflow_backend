package paymentinfra

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cashflow_backend/internal/domain/payment"
)

// RemoteProvider adapts a JSON payment gateway without coupling the domain to its SDK.
type RemoteProvider struct {
	Code          string
	BaseURL       string
	AuthToken     string
	WebhookSecret string
	ProviderID    int64
	HTTPClient    *http.Client
}

func (p *RemoteProvider) GetCode() string { return p.Code }

func (p *RemoteProvider) InitiatePayment(ctx context.Context, tx *payment.PaymentTransaction) (*payment.PaymentInitResult, error) {
	if tx == nil {
		return nil, fmt.Errorf("payment transaction is required")
	}
	var result payment.PaymentInitResult
	if err := p.request(ctx, http.MethodPost, "/payments", map[string]any{"reference": tx.Reference, "amount": tx.Amount, "currency": tx.Currency, "return_url": tx.ReturnURL}, &result); err != nil {
		return nil, err
	}
	if result.TransactionRef == "" {
		result.TransactionRef = tx.Reference
	}
	return &result, nil
}

func (p *RemoteProvider) CapturePayment(ctx context.Context, tx *payment.PaymentTransaction) error {
	return p.action(ctx, tx, "capture")
}
func (p *RemoteProvider) VoidPayment(ctx context.Context, tx *payment.PaymentTransaction) error {
	return p.action(ctx, tx, "void")
}

func (p *RemoteProvider) action(ctx context.Context, tx *payment.PaymentTransaction, action string) error {
	if tx == nil || tx.ProviderReference == "" {
		return fmt.Errorf("provider transaction reference is required")
	}
	return p.request(ctx, http.MethodPost, "/payments/"+url.PathEscape(tx.ProviderReference)+"/"+action, map[string]any{}, &struct{}{})
}

func (p *RemoteProvider) Refund(ctx context.Context, tx *payment.PaymentTransaction, amount float64) (*payment.PaymentRefund, error) {
	if tx == nil || tx.ProviderReference == "" {
		return nil, fmt.Errorf("provider transaction reference is required")
	}
	result := &payment.PaymentRefund{OriginalTxID: tx.ID, Amount: amount, Currency: tx.Currency, CompanyID: tx.CompanyID, State: "pending", CreatedAt: time.Now().UTC()}
	if err := result.Validate(tx); err != nil {
		return nil, err
	}
	var response struct {
		ProviderReference string `json:"provider_reference"`
	}
	if err := p.request(ctx, http.MethodPost, "/payments/"+url.PathEscape(tx.ProviderReference)+"/refunds", map[string]any{"amount": amount, "currency": tx.Currency}, &response); err != nil {
		return nil, err
	}
	result.ProviderReference = response.ProviderReference
	result.State = "done"
	return result, nil
}

func (p *RemoteProvider) HandleWebhook(_ context.Context, payload []byte, headers map[string]string) (*payment.WebhookResult, error) {
	if p.WebhookSecret != "" {
		provided := strings.TrimSpace(headers["X-Signature"])
		mac := hmac.New(sha256.New, []byte(p.WebhookSecret))
		_, _ = mac.Write(payload)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(strings.ToLower(provided)), []byte(expected)) {
			return nil, fmt.Errorf("invalid %s webhook signature", p.Code)
		}
	}
	var result payment.WebhookResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s webhook: %w", p.Code, err)
	}
	result.RawPayload = append([]byte(nil), payload...)
	return &result, nil
}

func (p *RemoteProvider) Tokenize(ctx context.Context, partnerID int64, tokenData map[string]string) (*payment.PaymentToken, error) {
	var result payment.PaymentToken
	if err := p.request(ctx, http.MethodPost, "/tokens", map[string]any{"partner_id": partnerID, "token": tokenData}, &result); err != nil {
		return nil, err
	}
	result.PartnerID = partnerID
	result.ProviderID = p.ProviderID
	result.Active = true
	return &result, nil
}

func (p *RemoteProvider) GetPaymentMethods(ctx context.Context) ([]payment.ProviderPaymentMethod, error) {
	var result []payment.ProviderPaymentMethod
	if err := p.request(ctx, http.MethodGet, "/payment-methods", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *RemoteProvider) request(ctx context.Context, method, path string, body any, result any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(p.BaseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.AuthToken)
	}
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s provider returned %d: %s", p.Code, response.StatusCode, string(data))
	}
	if result != nil && len(data) > 0 {
		return json.Unmarshal(data, result)
	}
	return nil
}
