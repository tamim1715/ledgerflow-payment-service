package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type paypalGateway struct{}

func (g *paypalGateway) InitiatePayment(ctx context.Context, input PaymentInput) (*GatewayInitiateResult, error) {
	token := fmt.Sprintf("EC-%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://www.sandbox.paypal.com/checkoutnow?token=%s", token)
	orderID := fmt.Sprintf("PAYID-%s", uuid.New().String()[:14])

	payerEmail := input.CustomerEmail
	if payerEmail == "" {
		payerEmail = "unknown@customer.com"
	}

	rawResponse := map[string]any{
		"id":            orderID,
		"intent":        "CAPTURE",
		"status":        "COMPLETED",
		"token":         token,
		"payer_email":   payerEmail,
		"gross_amount":  input.Amount,
		"currency_code": input.Currency,
	}

	return &GatewayInitiateResult{
		ProviderTxID: orderID,
		PaymentURL:   &paymentURL,
		Success:      true,
		Pending:      false,
		RawResponse:  rawResponse,
	}, nil
}

func (g *paypalGateway) ProcessRefund(ctx context.Context, input RefundInput) (*GatewayRefundResult, error) {
	refundID := fmt.Sprintf("paypal_ref_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"id":     refundID,
		"status": "COMPLETED",
		"amount": map[string]any{
			"value":         fmt.Sprintf("%.2f", input.Amount),
			"currency_code": input.Currency,
		},
		"create_time": time.Now().Format(time.RFC3339),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundID,
		Success:          true,
		Pending:          false,
		RawResponse:      rawResponse,
	}, nil
}
