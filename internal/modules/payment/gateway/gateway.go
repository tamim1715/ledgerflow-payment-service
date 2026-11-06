package gateway

import (
	"context"
)

type Provider string

const (
	ProviderStripe     Provider = "STRIPE"
	ProviderSSLCommerz Provider = "SSLCOMMERZ"
	ProviderPayPal     Provider = "PAYPAL"
	ProviderBKash      Provider = "BKASH"
	ProviderDefault    Provider = "DEFAULT"
)

type PaymentInput struct {
	PaymentID        string
	Amount           float64
	Currency         string
	CustomerID       string
	CustomerEmail    string
	CustomerName     string
	CustomerPhone    string
	StripeCustomerID string
	Description      string
	IdempotencyKey   string
	Metadata         map[string]any
}

type RefundInput struct {
	RefundID     string
	PaymentID    string
	ProviderTxID string
	Amount       float64
	Currency     string
	Reason       string
}

type GatewayInitiateResult struct {
	ProviderTxID string
	PaymentURL   *string
	Success      bool
	Pending      bool
	RawResponse  map[string]any
	ErrorCode    *string
	ErrorMessage *string
}

type GatewayRefundResult struct {
	ProviderRefundID string
	Success          bool
	Pending          bool
	RawResponse      map[string]any
	ErrorCode        *string
	ErrorMessage     *string
}

type IGateway interface {
	InitiatePayment(ctx context.Context, input PaymentInput) (*GatewayInitiateResult, error)
	ProcessRefund(ctx context.Context, input RefundInput) (*GatewayRefundResult, error)
}

func NewGateway(provider Provider) IGateway {
	switch provider {
	case ProviderStripe:
		return &stripeGateway{}
	case ProviderSSLCommerz:
		return &sslcommerzGateway{}
	case ProviderPayPal:
		return &paypalGateway{}
	case ProviderBKash:
		return &bkashGateway{}
	default:
		return &defaultGateway{provider: provider}
	}
}
