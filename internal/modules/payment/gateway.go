package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GatewayInitiateResult struct {
	ProviderTxID  string
	PaymentURL    *string
	Status        TransactionStatus
	PaymentStatus PaymentStatus
	RawResponse   map[string]any
	ErrorCode     *string
	ErrorMessage  *string
}

type GatewayRefundResult struct {
	ProviderRefundID string
	Status           RefundStatus
	RawResponse      map[string]any
	ErrorCode        *string
	ErrorMessage     *string
}

type IGateway interface {
	InitiatePayment(ctx context.Context, p *Payment, customer *Customer) (*GatewayInitiateResult, error)
	ProcessRefund(ctx context.Context, p *Payment, r *Refund) (*GatewayRefundResult, error)
}

func NewGateway(provider PaymentProvider) IGateway {
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

// ---------------------------------------------------------------------------
// Stripe Gateway Implementation
// ---------------------------------------------------------------------------
type stripeGateway struct{}

func (g *stripeGateway) InitiatePayment(ctx context.Context, p *Payment, customer *Customer) (*GatewayInitiateResult, error) {
	sessionID := fmt.Sprintf("cs_live_%s", uuid.New().String()[:18])
	chargeID := fmt.Sprintf("ch_stripe_%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID)

	stripeCustID := "cus_guest"
	if customer != nil && customer.StripeCustomerID != nil && *customer.StripeCustomerID != "" {
		stripeCustID = *customer.StripeCustomerID
	}

	rawResponse := map[string]any{
		"id":               chargeID,
		"object":           "charge",
		"amount":           p.Amount * 100, // cents
		"currency":         p.Currency,
		"status":           "succeeded",
		"paid":             true,
		"customer":         stripeCustID,
		"payment_method":   "pm_card_visa",
		"checkout_session": sessionID,
		"created":          time.Now().Unix(),
	}

	return &GatewayInitiateResult{
		ProviderTxID:  chargeID,
		PaymentURL:    &paymentURL,
		Status:        TxStatusSuccess,
		PaymentStatus: PaymentStatusSucceeded,
		RawResponse:   rawResponse,
	}, nil
}

func (g *stripeGateway) ProcessRefund(ctx context.Context, p *Payment, r *Refund) (*GatewayRefundResult, error) {
	refundID := fmt.Sprintf("re_stripe_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"id":          refundID,
		"object":      "refund",
		"amount":      r.Amount * 100,
		"currency":    r.Currency,
		"payment_id":  p.ID,
		"status":      "succeeded",
		"created":     time.Now().Unix(),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundID,
		Status:           RefundStatusSucceeded,
		RawResponse:      rawResponse,
	}, nil
}

// ---------------------------------------------------------------------------
// SSLCommerz Gateway Implementation
// ---------------------------------------------------------------------------
type sslcommerzGateway struct{}

func (g *sslcommerzGateway) InitiatePayment(ctx context.Context, p *Payment, customer *Customer) (*GatewayInitiateResult, error) {
	sessionKey := uuid.New().String()
	paymentURL := fmt.Sprintf("https://sandbox.sslcommerz.com/gwprocess/v4/gw.php?Q=pay&Sessionkey=%s", sessionKey)
	tranID := fmt.Sprintf("sslcz_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"status":      "SUCCESS",
		"sessionkey":  sessionKey,
		"tran_id":     tranID,
		"amount":      p.Amount,
		"currency":    p.Currency,
		"bank_tran_id": fmt.Sprintf("SSL_%d", time.Now().Unix()),
		"card_type":   "VISA-SSLCommerz",
	}

	return &GatewayInitiateResult{
		ProviderTxID:  tranID,
		PaymentURL:    &paymentURL,
		Status:        TxStatusSuccess,
		PaymentStatus: PaymentStatusSucceeded,
		RawResponse:   rawResponse,
	}, nil
}

func (g *sslcommerzGateway) ProcessRefund(ctx context.Context, p *Payment, r *Refund) (*GatewayRefundResult, error) {
	refundRef := fmt.Sprintf("ssl_ref_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"status":      "SUCCESS",
		"refund_ref":  refundRef,
		"trans_id":    p.ID,
		"amount":      r.Amount,
		"currency":    r.Currency,
		"refund_date": time.Now().Format(time.RFC3339),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundRef,
		Status:           RefundStatusSucceeded,
		RawResponse:      rawResponse,
	}, nil
}

// ---------------------------------------------------------------------------
// PayPal Gateway Implementation
// ---------------------------------------------------------------------------
type paypalGateway struct{}

func (g *paypalGateway) InitiatePayment(ctx context.Context, p *Payment, customer *Customer) (*GatewayInitiateResult, error) {
	token := fmt.Sprintf("EC-%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://www.sandbox.paypal.com/checkoutnow?token=%s", token)
	orderID := fmt.Sprintf("PAYID-%s", uuid.New().String()[:14])

	rawResponse := map[string]any{
		"id":            orderID,
		"intent":        "CAPTURE",
		"status":        "COMPLETED",
		"token":         token,
		"payer_email":   customerEmail(customer),
		"gross_amount":  p.Amount,
		"currency_code": p.Currency,
	}

	return &GatewayInitiateResult{
		ProviderTxID:  orderID,
		PaymentURL:    &paymentURL,
		Status:        TxStatusSuccess,
		PaymentStatus: PaymentStatusSucceeded,
		RawResponse:   rawResponse,
	}, nil
}

func (g *paypalGateway) ProcessRefund(ctx context.Context, p *Payment, r *Refund) (*GatewayRefundResult, error) {
	refundID := fmt.Sprintf("paypal_ref_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"id":     refundID,
		"status": "COMPLETED",
		"amount": map[string]any{
			"value":         fmt.Sprintf("%.2f", r.Amount),
			"currency_code": r.Currency,
		},
		"create_time": time.Now().Format(time.RFC3339),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundID,
		Status:           RefundStatusSucceeded,
		RawResponse:      rawResponse,
	}, nil
}

// ---------------------------------------------------------------------------
// bKash Gateway Implementation
// ---------------------------------------------------------------------------
type bkashGateway struct{}

func (g *bkashGateway) InitiatePayment(ctx context.Context, p *Payment, customer *Customer) (*GatewayInitiateResult, error) {
	paymentID := fmt.Sprintf("BKASH_%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://checkout.sandbox.bka.sh/v1.2.0-beta/tokenized/checkout?paymentID=%s", paymentID)
	trxID := fmt.Sprintf("TRX_%s", uuid.New().String()[:10])

	rawResponse := map[string]any{
		"paymentID":             paymentID,
		"trxID":                 trxID,
		"transactionStatus":     "Completed",
		"amount":                fmt.Sprintf("%.2f", p.Amount),
		"currency":              p.Currency,
		"intent":                "sale",
		"merchantInvoiceNumber": p.IdempotencyKey,
		"paymentExecuteTime":    time.Now().Format(time.RFC3339),
	}

	return &GatewayInitiateResult{
		ProviderTxID:  trxID,
		PaymentURL:    &paymentURL,
		Status:        TxStatusSuccess,
		PaymentStatus: PaymentStatusSucceeded,
		RawResponse:   rawResponse,
	}, nil
}

func (g *bkashGateway) ProcessRefund(ctx context.Context, p *Payment, r *Refund) (*GatewayRefundResult, error) {
	refundTrxID := fmt.Sprintf("BKR_%s", uuid.New().String()[:10])

	rawResponse := map[string]any{
		"statusCode":            "0000",
		"statusMessage":         "Successful",
		"originalTrxID":         p.ID,
		"refundTrxID":           refundTrxID,
		"amount":                fmt.Sprintf("%.2f", r.Amount),
		"currency":              r.Currency,
		"completedTime":         time.Now().Format(time.RFC3339),
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundTrxID,
		Status:           RefundStatusSucceeded,
		RawResponse:      rawResponse,
	}, nil
}

// ---------------------------------------------------------------------------
// Default Gateway Implementation
// ---------------------------------------------------------------------------
type defaultGateway struct {
	provider PaymentProvider
}

func (g *defaultGateway) InitiatePayment(ctx context.Context, p *Payment, customer *Customer) (*GatewayInitiateResult, error) {
	txID := fmt.Sprintf("generic_tx_%s", uuid.New().String()[:16])
	paymentURL := fmt.Sprintf("https://pay.ledgerflow.io/checkout/%s", p.ID)

	rawResponse := map[string]any{
		"provider": string(g.provider),
		"status":   "PROCESSED",
		"amount":   p.Amount,
		"currency": p.Currency,
	}

	return &GatewayInitiateResult{
		ProviderTxID:  txID,
		PaymentURL:    &paymentURL,
		Status:        TxStatusSuccess,
		PaymentStatus: PaymentStatusSucceeded,
		RawResponse:   rawResponse,
	}, nil
}

func (g *defaultGateway) ProcessRefund(ctx context.Context, p *Payment, r *Refund) (*GatewayRefundResult, error) {
	refundID := fmt.Sprintf("generic_ref_%s", uuid.New().String()[:16])

	rawResponse := map[string]any{
		"provider": string(g.provider),
		"status":   "REFUNDED",
		"amount":   r.Amount,
	}

	return &GatewayRefundResult{
		ProviderRefundID: refundID,
		Status:           RefundStatusSucceeded,
		RawResponse:      rawResponse,
	}, nil
}

func customerEmail(c *Customer) string {
	if c != nil && c.Email != "" {
		return c.Email
	}
	return "unknown@customer.com"
}
