package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type bkashGateway struct{}

func (g *bkashGateway) InitiatePayment(ctx context.Context, input PaymentInput) (*GatewayInitiateResult, error) {
	paymentID := fmt.Sprintf("BKASH_%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://checkout.sandbox.bka.sh/v1.2.0-beta/tokenized/checkout?paymentID=%s", paymentID)
	trxID := fmt.Sprintf("TRX_%s", uuid.New().String()[:10])

	invoiceNumber := input.IdempotencyKey
	if invoiceNumber == "" {
		invoiceNumber = input.PaymentID
	}

	rawResponse := map[string]any{
		"paymentID":             paymentID,
		"trxID":                 trxID,
		"transactionStatus":     "Completed",
		"amount":                fmt.Sprintf("%.2f", input.Amount),
		"currency":              input.Currency,
		"intent":                "sale",
		"merchantInvoiceNumber": invoiceNumber,
		"paymentExecuteTime":    time.Now().Format(time.RFC3339),
	}

	return &GatewayInitiateResult{
		ProviderTxID: trxID,
		PaymentURL:   &paymentURL,
		Success:      true,
		Pending:      false,
		RawResponse:  rawResponse,
	}, nil
}

func (g *bkashGateway) ProcessRefund(ctx context.Context, input RefundInput) (*GatewayRefundResult, error) {
	refundTrxID := fmt.Sprintf("BKR_%s", uuid.New().String()[:10])

	rawResponse := map[string]any{
		"statusCode":    "0000",
		"statusMessage": "Successful",
		"originalTrxID": input.PaymentID,
		"refundTrxID":   refundTrxID,
		"amount":        fmt.Sprintf("%.2f", input.Amount),
		"currency":      input.Currency,
		"completedTime": time.Now().Format(time.RFC3339),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundTrxID,
		Success:          true,
		Pending:          false,
		RawResponse:      rawResponse,
	}, nil
}
