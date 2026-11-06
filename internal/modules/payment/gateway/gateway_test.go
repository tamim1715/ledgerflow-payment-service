package gateway

import (
	"context"
	"testing"
)

func TestNewGateway(t *testing.T) {
	tests := []struct {
		provider Provider
		name     string
	}{
		{ProviderStripe, "Stripe"},
		{ProviderSSLCommerz, "SSLCommerz"},
		{ProviderPayPal, "PayPal"},
		{ProviderBKash, "bKash"},
		{Provider("UNKNOWN"), "Default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := NewGateway(tt.provider)
			if gw == nil {
				t.Fatalf("expected gateway for %s, got nil", tt.provider)
			}
		})
	}
}

func TestGatewayInitiatePayment(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		provider Provider
	}{
		{ProviderStripe},
		{ProviderSSLCommerz},
		{ProviderPayPal},
		{ProviderBKash},
		{Provider("GENERIC")},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			gw := NewGateway(tt.provider)
			input := PaymentInput{
				PaymentID:      "pay_test123",
				IdempotencyKey: "ik_test123",
				CustomerID:     "cust_123",
				CustomerEmail:  "test@example.com",
				CustomerName:   "Test User",
				Amount:         150.50,
				Currency:       "USD",
			}

			result, err := gw.InitiatePayment(ctx, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ProviderTxID == "" {
				t.Errorf("expected ProviderTxID to be non-empty")
			}
			if result.PaymentURL == nil || *result.PaymentURL == "" {
				t.Errorf("expected PaymentURL to be non-empty")
			}
			if !result.Success {
				t.Errorf("expected Success to be true")
			}
			if result.RawResponse == nil {
				t.Errorf("expected RawResponse to be populated")
			}
		})
	}
}

func TestGatewayProcessRefund(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		provider Provider
	}{
		{ProviderStripe},
		{ProviderSSLCommerz},
		{ProviderPayPal},
		{ProviderBKash},
		{Provider("GENERIC")},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			gw := NewGateway(tt.provider)
			input := RefundInput{
				RefundID:     "ref_test123",
				PaymentID:    "pay_test123",
				ProviderTxID: "tx_orig123",
				Amount:       50.25,
				Currency:     "USD",
				Reason:       "customer requested",
			}

			result, err := gw.ProcessRefund(ctx, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ProviderRefundID == "" {
				t.Errorf("expected ProviderRefundID to be non-empty")
			}
			if !result.Success {
				t.Errorf("expected Success to be true")
			}
			if result.RawResponse == nil {
				t.Errorf("expected RawResponse to be populated")
			}
		})
	}
}
