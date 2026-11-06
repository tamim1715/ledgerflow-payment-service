package gateway

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type defaultGateway struct {
	provider Provider
}

func (g *defaultGateway) InitiatePayment(ctx context.Context, input PaymentInput) (*GatewayInitiateResult, error) {
	txID := fmt.Sprintf("generic_tx_%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://pay.ledgerflow.io/checkout/%s", input.PaymentID)

	rawResponse := map[string]any{
		"provider": string(g.provider),
		"status":   "PROCESSED",
		"amount":   input.Amount,
		"currency": input.Currency,
	}

	return &GatewayInitiateResult{
		ProviderTxID: txID,
		PaymentURL:   &paymentURL,
		Success:      true,
		Pending:      false,
		RawResponse:  rawResponse,
	}, nil
}

func (g *defaultGateway) ProcessRefund(ctx context.Context, input RefundInput) (*GatewayRefundResult, error) {
	refundID := fmt.Sprintf("generic_ref_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"provider": string(g.provider),
		"status":   "REFUNDED",
		"amount":   input.Amount,
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundID,
		Success:          true,
		Pending:          false,
		RawResponse:      rawResponse,
	}, nil
}
