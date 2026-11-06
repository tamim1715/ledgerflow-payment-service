package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type stripeGateway struct{}

func (g *stripeGateway) InitiatePayment(ctx context.Context, input PaymentInput) (*GatewayInitiateResult, error) {
	sessionID := fmt.Sprintf("cs_live_%s", uuid.New().String()[:18])
	chargeID := fmt.Sprintf("ch_stripe_%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID)

	stripeCustID := "cus_guest"
	if input.StripeCustomerID != "" {
		stripeCustID = input.StripeCustomerID
	}

	rawResponse := map[string]any{
		"id":               chargeID,
		"object":           "charge",
		"amount":           input.Amount * 100, // cents
		"currency":         input.Currency,
		"status":           "succeeded",
		"paid":             true,
		"customer":         stripeCustID,
		"payment_method":   "pm_card_visa",
		"checkout_session": sessionID,
		"created":          time.Now().Unix(),
	}

	return &GatewayInitiateResult{
		ProviderTxID: chargeID,
		PaymentURL:   &paymentURL,
		Success:      true,
		Pending:      false,
		RawResponse:  rawResponse,
	}, nil
}

func (g *stripeGateway) ProcessRefund(ctx context.Context, input RefundInput) (*GatewayRefundResult, error) {
	refundID := fmt.Sprintf("re_stripe_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"id":         refundID,
		"object":     "refund",
		"amount":     input.Amount * 100,
		"currency":   input.Currency,
		"payment_id": input.PaymentID,
		"status":     "succeeded",
		"created":    time.Now().Unix(),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundID,
		Success:          true,
		Pending:          false,
		RawResponse:      rawResponse,
	}, nil
}
