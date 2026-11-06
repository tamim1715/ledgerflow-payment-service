package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ledgerflow-payment-service/internal/modules/payment"
)

var (
	ErrUnsupportedProvider = errors.New("unsupported payment webhook provider")
	ErrInvalidWebhookData  = errors.New("invalid webhook payload")
)

type IWebhookParser interface {
	Parse(headers map[string]string, body []byte) (*NormalizedWebhookEvent, error)
}

func NewWebhookParser(provider payment.PaymentProvider) (IWebhookParser, error) {
	switch provider {
	case payment.ProviderStripe:
		return &StripeWebhookParser{}, nil
	case payment.ProviderSSLCommerz:
		return &SSLCommerzWebhookParser{}, nil
	case payment.ProviderBKash:
		return &BKashWebhookParser{}, nil
	case payment.ProviderPayPal:
		return &PayPalWebhookParser{}, nil
	default:
		return &GenericWebhookParser{provider: provider}, nil
	}
}

// ---------------------------------------------------------------------------
// Stripe Webhook Parser
// ---------------------------------------------------------------------------

type StripeWebhookParser struct{}

func (p *StripeWebhookParser) Parse(headers map[string]string, body []byte) (*NormalizedWebhookEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: stripe json decode: %v", ErrInvalidWebhookData, err)
	}

	eventID, _ := payload["id"].(string)
	if eventID == "" {
		eventID = fmt.Sprintf("evt_stripe_%d", time.Now().UnixNano())
	}

	eventType, _ := payload["type"].(string)
	if eventType == "" {
		eventType = "payment_intent.succeeded"
	}

	var paymentID string
	var providerTxID string
	var amount float64
	var currency string
	status := payment.PaymentStatusSucceeded
	var errCode *string
	var errMsg *string
	metadata := make(map[string]any)

	// Extract object from data.object
	if data, ok := payload["data"].(map[string]any); ok {
		if obj, ok := data["object"].(map[string]any); ok {
			if id, ok := obj["id"].(string); ok {
				providerTxID = id
			}

			// Extract metadata
			if meta, ok := obj["metadata"].(map[string]any); ok {
				metadata = meta
				if pid, ok := meta["payment_id"].(string); ok {
					paymentID = pid
				}
			}

			// Checkout session client reference fallback
			if paymentID == "" {
				if clientRef, ok := obj["client_reference_id"].(string); ok && clientRef != "" {
					paymentID = clientRef
				}
			}

			if cur, ok := obj["currency"].(string); ok {
				currency = strings.ToUpper(cur)
			}

			// Amount in cents
			if amt, ok := obj["amount"].(float64); ok {
				amount = amt / 100.0
			} else if amtTotal, ok := obj["amount_total"].(float64); ok {
				amount = amtTotal / 100.0
			}

			// Error details if failed
			if lastErr, ok := obj["last_payment_error"].(map[string]any); ok {
				if code, ok := lastErr["code"].(string); ok {
					errCode = &code
				}
				if msg, ok := lastErr["message"].(string); ok {
					errMsg = &msg
				}
			}
		}
	}

	if paymentID == "" {
		paymentID = providerTxID
	}

	switch eventType {
	case "payment_intent.succeeded", "checkout.session.completed", "charge.succeeded":
		status = payment.PaymentStatusSucceeded
	case "payment_intent.payment_failed", "charge.failed":
		status = payment.PaymentStatusFailed
	case "payment_intent.canceled":
		status = payment.PaymentStatusCancelled
	case "charge.refunded":
		status = payment.PaymentStatusRefunded
	}

	return &NormalizedWebhookEvent{
		Provider:     payment.ProviderStripe,
		EventID:      eventID,
		EventType:    eventType,
		PaymentID:    paymentID,
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       amount,
		Currency:     currency,
		ErrorCode:    errCode,
		ErrorMessage: errMsg,
		Metadata:     metadata,
		RawPayload:   payload,
	}, nil
}

// ---------------------------------------------------------------------------
// SSLCommerz Webhook Parser (Supports JSON and Form URL-Encoded IPN)
// ---------------------------------------------------------------------------

type SSLCommerzWebhookParser struct{}

func (p *SSLCommerzWebhookParser) Parse(headers map[string]string, body []byte) (*NormalizedWebhookEvent, error) {
	dataMap := make(map[string]any)

	// Try JSON first, fallback to URL-encoded form
	if err := json.Unmarshal(body, &dataMap); err != nil {
		values, formErr := url.ParseQuery(string(body))
		if formErr != nil {
			return nil, fmt.Errorf("%w: sslcommerz payload parse failed: %v", ErrInvalidWebhookData, formErr)
		}
		for k, v := range values {
			if len(v) > 0 {
				dataMap[k] = v[0]
			}
		}
	}

	tranID, _ := dataMap["tran_id"].(string)
	valID, _ := dataMap["val_id"].(string)
	bankTranID, _ := dataMap["bank_tran_id"].(string)
	rawStatus, _ := dataMap["status"].(string)
	currency, _ := dataMap["currency"].(string)
	rawAmount, _ := dataMap["amount"]

	var amount float64
	switch v := rawAmount.(type) {
	case float64:
		amount = v
	case string:
		amount, _ = strconv.ParseFloat(v, 64)
	}

	eventID := valID
	if eventID == "" {
		eventID = fmt.Sprintf("ssl_%s_%s", tranID, rawStatus)
	}

	providerTxID := bankTranID
	if providerTxID == "" {
		providerTxID = valID
	}

	status := payment.PaymentStatusFailed
	eventType := "sslcommerz.ipn." + strings.ToLower(rawStatus)
	var errCode *string
	var errMsg *string

	switch strings.ToUpper(rawStatus) {
	case "VALID", "VALIDATED":
		status = payment.PaymentStatusSucceeded
	case "CANCELLED":
		status = payment.PaymentStatusCancelled
		msg := "Transaction cancelled by user or gateway"
		errMsg = &msg
	default:
		status = payment.PaymentStatusFailed
		code := rawStatus
		errCode = &code
		if errReason, ok := dataMap["error"].(string); ok {
			errMsg = &errReason
		}
	}

	return &NormalizedWebhookEvent{
		Provider:     payment.ProviderSSLCommerz,
		EventID:      eventID,
		EventType:    eventType,
		PaymentID:    tranID,
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       amount,
		Currency:     strings.ToUpper(currency),
		ErrorCode:    errCode,
		ErrorMessage: errMsg,
		Metadata:     dataMap,
		RawPayload:   dataMap,
	}, nil
}

// ---------------------------------------------------------------------------
// bKash Webhook Parser
// ---------------------------------------------------------------------------

type BKashWebhookParser struct{}

func (p *BKashWebhookParser) Parse(headers map[string]string, body []byte) (*NormalizedWebhookEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: bkash json decode: %v", ErrInvalidWebhookData, err)
	}

	paymentID, _ := payload["paymentID"].(string)
	trxID, _ := payload["trxID"].(string)
	txStatus, _ := payload["transactionStatus"].(string)
	invoiceNo, _ := payload["merchantInvoiceNumber"].(string)
	currency, _ := payload["currency"].(string)
	rawAmount, _ := payload["amount"]

	var amount float64
	switch v := rawAmount.(type) {
	case float64:
		amount = v
	case string:
		amount, _ = strconv.ParseFloat(v, 64)
	}

	targetPaymentID := invoiceNo
	if targetPaymentID == "" {
		targetPaymentID = paymentID
	}

	eventID := trxID
	if eventID == "" {
		eventID = fmt.Sprintf("bks_%s_%s", paymentID, txStatus)
	}

	status := payment.PaymentStatusFailed
	eventType := "bkash.payment." + strings.ToLower(txStatus)
	var errCode *string
	var errMsg *string

	switch strings.ToLower(txStatus) {
	case "completed", "success":
		status = payment.PaymentStatusSucceeded
	case "cancelled":
		status = payment.PaymentStatusCancelled
		msg := "Customer cancelled bKash payment"
		errMsg = &msg
	default:
		status = payment.PaymentStatusFailed
		if code, ok := payload["errorCode"].(string); ok {
			errCode = &code
		}
		if msg, ok := payload["errorMessage"].(string); ok {
			errMsg = &msg
		}
	}

	return &NormalizedWebhookEvent{
		Provider:     payment.ProviderBKash,
		EventID:      eventID,
		EventType:    eventType,
		PaymentID:    targetPaymentID,
		ProviderTxID: trxID,
		Status:       status,
		Amount:       amount,
		Currency:     strings.ToUpper(currency),
		ErrorCode:    errCode,
		ErrorMessage: errMsg,
		Metadata:     payload,
		RawPayload:   payload,
	}, nil
}

// ---------------------------------------------------------------------------
// PayPal Webhook Parser
// ---------------------------------------------------------------------------

type PayPalWebhookParser struct{}

func (p *PayPalWebhookParser) Parse(headers map[string]string, body []byte) (*NormalizedWebhookEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: paypal json decode: %v", ErrInvalidWebhookData, err)
	}

	eventID, _ := payload["id"].(string)
	eventType, _ := payload["event_type"].(string)
	if eventID == "" {
		eventID = fmt.Sprintf("wh_paypal_%d", time.Now().UnixNano())
	}

	var paymentID string
	var providerTxID string
	var amount float64
	var currency string
	status := payment.PaymentStatusProcessing

	if resource, ok := payload["resource"].(map[string]any); ok {
		if id, ok := resource["id"].(string); ok {
			providerTxID = id
		}
		if customID, ok := resource["custom_id"].(string); ok && customID != "" {
			paymentID = customID
		} else if invoiceID, ok := resource["invoice_id"].(string); ok && invoiceID != "" {
			paymentID = invoiceID
		}

		if amtMap, ok := resource["amount"].(map[string]any); ok {
			if valStr, ok := amtMap["value"].(string); ok {
				amount, _ = strconv.ParseFloat(valStr, 64)
			}
			if cur, ok := amtMap["currency_code"].(string); ok {
				currency = cur
			}
		}
	}

	if paymentID == "" {
		paymentID = providerTxID
	}

	switch eventType {
	case "PAYMENT.CAPTURE.COMPLETED", "CHECKOUT.ORDER.APPROVED":
		status = payment.PaymentStatusSucceeded
	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		status = payment.PaymentStatusFailed
	case "PAYMENT.CAPTURE.REFUNDED":
		status = payment.PaymentStatusRefunded
	default:
		status = payment.PaymentStatusProcessing
	}

	return &NormalizedWebhookEvent{
		Provider:     payment.ProviderPayPal,
		EventID:      eventID,
		EventType:    eventType,
		PaymentID:    paymentID,
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       amount,
		Currency:     strings.ToUpper(currency),
		Metadata:     payload,
		RawPayload:   payload,
	}, nil
}

// ---------------------------------------------------------------------------
// Generic / Fallback Webhook Parser
// ---------------------------------------------------------------------------

type GenericWebhookParser struct {
	provider payment.PaymentProvider
}

func (p *GenericWebhookParser) Parse(headers map[string]string, body []byte) (*NormalizedWebhookEvent, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: generic json decode: %v", ErrInvalidWebhookData, err)
	}

	eventID, _ := payload["id"].(string)
	if eventID == "" {
		eventID = fmt.Sprintf("evt_%s_%d", p.provider, time.Now().UnixNano())
	}

	eventType, _ := payload["type"].(string)
	if eventType == "" {
		eventType = fmt.Sprintf("%s.event", strings.ToLower(string(p.provider)))
	}

	paymentID, _ := payload["payment_id"].(string)
	providerTxID, _ := payload["transaction_id"].(string)
	if providerTxID == "" {
		providerTxID, _ = payload["provider_tx_id"].(string)
	}

	statusStr, _ := payload["status"].(string)
	status := payment.PaymentStatusSucceeded
	switch strings.ToUpper(statusStr) {
	case "FAILED":
		status = payment.PaymentStatusFailed
	case "CANCELLED":
		status = payment.PaymentStatusCancelled
	case "REFUNDED":
		status = payment.PaymentStatusRefunded
	}

	var amount float64
	if amt, ok := payload["amount"].(float64); ok {
		amount = amt
	}
	currency, _ := payload["currency"].(string)

	return &NormalizedWebhookEvent{
		Provider:     p.provider,
		EventID:      eventID,
		EventType:    eventType,
		PaymentID:    paymentID,
		ProviderTxID: providerTxID,
		Status:       status,
		Amount:       amount,
		Currency:     strings.ToUpper(currency),
		Metadata:     payload,
		RawPayload:   payload,
	}, nil
}
