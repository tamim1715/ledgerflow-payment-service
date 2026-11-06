package webhook

import (
	"ledgerflow-payment-service/internal/modules/payment"
)

type NormalizedWebhookEvent struct {
	Provider     payment.PaymentProvider
	EventID      string
	EventType    string
	PaymentID    string
	ProviderTxID string
	Status       payment.PaymentStatus
	Amount       float64
	Currency     string
	ErrorCode    *string
	ErrorMessage *string
	Metadata     map[string]any
	RawPayload   map[string]any
}

type WebhookProcessResult struct {
	EventID   string                   `json:"event_id" example:"evt_123456789"`
	Provider  string                   `json:"provider" example:"STRIPE"`
	EventType string                   `json:"event_type" example:"payment_intent.succeeded"`
	Status    string                   `json:"status" example:"PROCESSED"`
	Payment   *payment.PaymentResponse `json:"payment,omitempty"`
	Message   string                   `json:"message" example:"Webhook processed successfully"`
}
