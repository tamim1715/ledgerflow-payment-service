package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type sslcommerzGateway struct{}

func (g *sslcommerzGateway) InitiatePayment(ctx context.Context, input PaymentInput) (*GatewayInitiateResult, error) {
	sessionKey := uuid.New().String()
	paymentURL := fmt.Sprintf("https://sandbox.sslcommerz.com/gwprocess/v4/gw.php?Q=pay&Sessionkey=%s", sessionKey)
	tranID := fmt.Sprintf("sslcz_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"status":       "SUCCESS",
		"sessionkey":   sessionKey,
		"tran_id":      tranID,
		"amount":       input.Amount,
		"currency":     input.Currency,
		"bank_tran_id": fmt.Sprintf("SSL_%d", time.Now().Unix()),
		"card_type":    "VISA-SSLCommerz",
	}

	return &GatewayInitiateResult{
		ProviderTxID: tranID,
		PaymentURL:   &paymentURL,
		Success:      true,
		Pending:      false,
		RawResponse:  rawResponse,
	}, nil
}

func (g *sslcommerzGateway) ProcessRefund(ctx context.Context, input RefundInput) (*GatewayRefundResult, error) {
	refundRef := fmt.Sprintf("ssl_ref_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"status":      "SUCCESS",
		"refund_ref":  refundRef,
		"trans_id":    input.PaymentID,
		"amount":      input.Amount,
		"currency":    input.Currency,
		"refund_date": time.Now().Format(time.RFC3339),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundRef,
		Success:          true,
		Pending:          false,
		RawResponse:      rawResponse,
	}, nil
}
