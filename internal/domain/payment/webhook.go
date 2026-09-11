package payment

import "time"

type WebhookResult struct {
	TransactionRef string           `json:"transaction_ref"`
	NewState       TransactionState `json:"new_state"`
	ProviderRef    string           `json:"provider_ref,omitempty"`
	Amount         *float64         `json:"amount,omitempty"`
	ErrorMessage   string           `json:"error_message,omitempty"`
	RawPayload     []byte           `json:"-"`
}

type WebhookLog struct {
	ID             int64      `json:"id"`
	ProviderCode   string     `json:"provider_code"`
	EventType      string     `json:"event_type"`
	Payload        []byte     `json:"payload"`
	Processed      bool       `json:"processed"`
	ProcessError   string     `json:"process_error,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	ReceivedAt     time.Time  `json:"received_at"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
}
