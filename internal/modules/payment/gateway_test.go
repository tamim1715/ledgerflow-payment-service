package payment

import (
	"context"
	"testing"
)

func TestNewGateway(t *testing.T) {
	tests := []struct {
		provider PaymentProvider
		name     string
	}{
		{ProviderStripe, "Stripe"},
		{ProviderSSLCommerz, "SSLCommerz"},
		{ProviderPayPal, "PayPal"},
		{ProviderBKash, "bKash"},
		{PaymentProvider("UNKNOWN"), "Default"},
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
	customer := &Customer{
		ID:    "cust_123",
		Email: "test@example.com",
	}

	tests := []struct {
		provider PaymentProvider
	}{
		{ProviderStripe},
		{ProviderSSLCommerz},
		{ProviderPayPal},
		{ProviderBKash},
		{PaymentProvider("GENERIC")},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			gw := NewGateway(tt.provider)
			payment := &Payment{
				ID:             "pay_test123",
				IdempotencyKey: "ik_test123",
				CustomerID:     customer.ID,
				Amount:         150.50,
				Currency:       "USD",
				Provider:       tt.provider,
				Status:         PaymentStatusProcessing,
			}

			result, err := gw.InitiatePayment(ctx, payment, customer)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ProviderTxID == "" {
				t.Errorf("expected ProviderTxID to be non-empty")
			}
			if result.PaymentURL == nil || *result.PaymentURL == "" {
				t.Errorf("expected PaymentURL to be non-empty")
			}
			if result.Status != TxStatusSuccess {
				t.Errorf("expected TxStatusSuccess, got %v", result.Status)
			}
			if result.PaymentStatus != PaymentStatusSucceeded {
				t.Errorf("expected PaymentStatusSucceeded, got %v", result.PaymentStatus)
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
		provider PaymentProvider
	}{
		{ProviderStripe},
		{ProviderSSLCommerz},
		{ProviderPayPal},
		{ProviderBKash},
		{PaymentProvider("GENERIC")},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			gw := NewGateway(tt.provider)
			payment := &Payment{
				ID:       "pay_test123",
				Amount:   200.00,
				Currency: "USD",
				Provider: tt.provider,
				Status:   PaymentStatusSucceeded,
			}
			refund := &Refund{
				ID:        "ref_test123",
				PaymentID: payment.ID,
				Amount:    50.00,
				Currency:  "USD",
			}

			result, err := gw.ProcessRefund(ctx, payment, refund)
			if err != nil {
				t.Fatalf("unexpected refund error: %v", err)
			}
			if result.ProviderRefundID == "" {
				t.Errorf("expected ProviderRefundID to be non-empty")
			}
			if result.Status != RefundStatusSucceeded {
				t.Errorf("expected RefundStatusSucceeded, got %v", result.Status)
			}
			if result.RawResponse == nil {
				t.Errorf("expected RawResponse to be populated")
			}
		})
	}
}
