package webhook

import (
	"context"
	"testing"
	"time"

	"ledgerflow-payment-service/internal/modules/payment"

	"go.uber.org/zap"
)

type mockWebhookRepo struct {
	events map[string]bool
	list   []WebhookEvent
}

func newMockWebhookRepo() *mockWebhookRepo {
	return &mockWebhookRepo{
		events: make(map[string]bool),
		list:   []WebhookEvent{},
	}
}

func (m *mockWebhookRepo) RecordEvent(ctx context.Context, provider string, eventID string, eventType string, payload string) (bool, error) {
	key := provider + ":" + eventID
	if m.events[key] {
		return false, nil
	}
	m.events[key] = true
	m.list = append(m.list, WebhookEvent{
		ID:          "ev_" + eventID,
		Provider:    provider,
		EventID:     eventID,
		EventType:   eventType,
		Payload:     payload,
		Status:      EventStatusProcessed,
		ProcessedAt: time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	})
	return true, nil
}

func (m *mockWebhookRepo) GetEventByProviderAndID(ctx context.Context, provider string, eventID string) (*WebhookEvent, error) {
	for _, e := range m.list {
		if e.Provider == provider && e.EventID == eventID {
			return &e, nil
		}
	}
	return nil, ErrEventNotFound
}

func (m *mockWebhookRepo) ListEvents(ctx context.Context, provider string, page int, limit int) ([]WebhookEvent, int, error) {
	var result []WebhookEvent
	for _, e := range m.list {
		if provider != "" && e.Provider != provider {
			continue
		}
		result = append(result, e)
	}
	return result, len(result), nil
}

type mockPaymentService struct {
	payment.IService
	events []payment.WebhookPaymentEvent
}

func (m *mockPaymentService) ProcessWebhookEvent(ctx context.Context, event payment.WebhookPaymentEvent) (*payment.PaymentResponse, error) {
	m.events = append(m.events, event)
	return &payment.PaymentResponse{
		ID:        event.PaymentID,
		Status:    event.Status,
		Amount:    event.Amount,
		Currency:  event.Currency,
		Provider:  event.Provider,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func TestProcessWebhook_Stripe_Success(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	stripePayload := []byte(`{
		"id": "evt_stripe_999",
		"object": "event",
		"type": "payment_intent.succeeded",
		"data": {
			"object": {
				"id": "pi_3Mtw",
				"amount": 25000,
				"currency": "usd",
				"status": "succeeded",
				"metadata": {
					"payment_id": "pay_stripe_101"
				}
			}
		}
	}`)

	res, err := svc.ProcessWebhook(ctx, "STRIPE", nil, stripePayload)
	if err != nil {
		t.Fatalf("unexpected error processing stripe webhook: %v", err)
	}

	if res.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res.Status)
	}
	if len(paySvc.events) != 1 {
		t.Fatalf("expected 1 event forwarded to payment, got %d", len(paySvc.events))
	}
	ev := paySvc.events[0]
	if ev.PaymentID != "pay_stripe_101" {
		t.Errorf("expected paymentID pay_stripe_101, got %s", ev.PaymentID)
	}
	if ev.Amount != 250.0 {
		t.Errorf("expected amount 250.00, got %.2f", ev.Amount)
	}
	if ev.Status != payment.PaymentStatusSucceeded {
		t.Errorf("expected SUCCEEDED status, got %v", ev.Status)
	}
}

func TestProcessWebhook_Stripe_Failed(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	payload := []byte(`{
		"id": "evt_stripe_fail_1",
		"type": "payment_intent.payment_failed",
		"data": {
			"object": {
				"id": "pi_fail",
				"amount": 1000,
				"currency": "usd",
				"metadata": { "payment_id": "pay_fail_1" },
				"last_payment_error": { "code": "card_declined", "message": "Insufficient funds" }
			}
		}
	}`)

	res, err := svc.ProcessWebhook(ctx, "STRIPE", nil, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res.Status)
	}
	if paySvc.events[0].Status != payment.PaymentStatusFailed {
		t.Errorf("expected status FAILED, got %v", paySvc.events[0].Status)
	}
	if paySvc.events[0].ErrorMessage == nil || *paySvc.events[0].ErrorMessage != "Insufficient funds" {
		t.Errorf("expected message 'Insufficient funds', got %v", paySvc.events[0].ErrorMessage)
	}
}

func TestProcessWebhook_SSLCommerz_JSON(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	payload := []byte(`{
		"tran_id": "pay_ssl_201",
		"val_id": "20081022375510",
		"amount": "1500.00",
		"currency": "BDT",
		"bank_tran_id": "SSL_BANK_777",
		"status": "VALID"
	}`)

	res, err := svc.ProcessWebhook(ctx, "SSLCOMMERZ", nil, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res.Status)
	}
	ev := paySvc.events[0]
	if ev.PaymentID != "pay_ssl_201" {
		t.Errorf("expected paymentID pay_ssl_201, got %s", ev.PaymentID)
	}
	if ev.Status != payment.PaymentStatusSucceeded {
		t.Errorf("expected status SUCCEEDED, got %v", ev.Status)
	}
	if ev.ProviderTxID != "SSL_BANK_777" {
		t.Errorf("expected ProviderTxID SSL_BANK_777, got %s", ev.ProviderTxID)
	}
}

func TestProcessWebhook_SSLCommerz_FormURLEncoded(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	formPayload := []byte("tran_id=pay_ssl_form_1&val_id=VAL999&amount=500.00&currency=BDT&status=VALID&bank_tran_id=BTRX123")

	res, err := svc.ProcessWebhook(ctx, "SSLCOMMERZ", nil, formPayload)
	if err != nil {
		t.Fatalf("unexpected error parsing url-encoded form: %v", err)
	}

	if res.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res.Status)
	}
	ev := paySvc.events[0]
	if ev.PaymentID != "pay_ssl_form_1" {
		t.Errorf("expected paymentID pay_ssl_form_1, got %s", ev.PaymentID)
	}
	if ev.Amount != 500.00 {
		t.Errorf("expected amount 500.00, got %.2f", ev.Amount)
	}
}

func TestProcessWebhook_BKash_Completed(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	bkashPayload := []byte(`{
		"paymentID": "TR0011B101",
		"trxID": "8A541Z89",
		"transactionStatus": "Completed",
		"amount": "250.00",
		"currency": "BDT",
		"merchantInvoiceNumber": "pay_bkash_301"
	}`)

	res, err := svc.ProcessWebhook(ctx, "BKASH", nil, bkashPayload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res.Status)
	}
	ev := paySvc.events[0]
	if ev.PaymentID != "pay_bkash_301" {
		t.Errorf("expected paymentID pay_bkash_301, got %s", ev.PaymentID)
	}
	if ev.ProviderTxID != "8A541Z89" {
		t.Errorf("expected ProviderTxID 8A541Z89, got %s", ev.ProviderTxID)
	}
	if ev.Status != payment.PaymentStatusSucceeded {
		t.Errorf("expected SUCCEEDED status, got %v", ev.Status)
	}
}

func TestProcessWebhook_PayPal_CaptureCompleted(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	paypalPayload := []byte(`{
		"id": "WH-909876",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
		"resource": {
			"id": "2GG12345",
			"status": "COMPLETED",
			"amount": {
				"currency_code": "USD",
				"value": "120.00"
			},
			"custom_id": "pay_paypal_401"
		}
	}`)

	res, err := svc.ProcessWebhook(ctx, "PAYPAL", nil, paypalPayload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res.Status)
	}
	ev := paySvc.events[0]
	if ev.PaymentID != "pay_paypal_401" {
		t.Errorf("expected paymentID pay_paypal_401, got %s", ev.PaymentID)
	}
	if ev.Amount != 120.00 {
		t.Errorf("expected amount 120.00, got %.2f", ev.Amount)
	}
	if ev.Status != payment.PaymentStatusSucceeded {
		t.Errorf("expected SUCCEEDED status, got %v", ev.Status)
	}
}

func TestProcessWebhook_Deduplication(t *testing.T) {
	repo := newMockWebhookRepo()
	paySvc := &mockPaymentService{}
	svc := NewService(repo, paySvc, zap.NewNop())
	ctx := context.Background()

	payload := []byte(`{
		"id": "evt_duplicate_test",
		"type": "payment_intent.succeeded",
		"data": {
			"object": {
				"id": "pi_dup",
				"amount": 5000,
				"currency": "usd",
				"metadata": { "payment_id": "pay_dup" }
			}
		}
	}`)

	// 1st request -> PROCESSED
	res1, err := svc.ProcessWebhook(ctx, "STRIPE", nil, payload)
	if err != nil {
		t.Fatalf("unexpected error on 1st request: %v", err)
	}
	if res1.Status != "PROCESSED" {
		t.Errorf("expected PROCESSED, got %s", res1.Status)
	}

	// 2nd identical request -> IGNORED_DUPLICATE
	res2, err := svc.ProcessWebhook(ctx, "STRIPE", nil, payload)
	if err != nil {
		t.Fatalf("unexpected error on 2nd request: %v", err)
	}
	if res2.Status != "IGNORED_DUPLICATE" {
		t.Errorf("expected IGNORED_DUPLICATE, got %s", res2.Status)
	}

	// Payment service should only have been called ONCE!
	if len(paySvc.events) != 1 {
		t.Errorf("expected exactly 1 call to payment service, got %d", len(paySvc.events))
	}
}
