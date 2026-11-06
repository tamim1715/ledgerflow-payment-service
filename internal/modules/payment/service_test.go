package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// In-memory mock repository for unit testing
type mockRepository struct {
	payments     map[string]*Payment
	idempotency  map[string]*Payment
	refunds      map[string]*Refund
	transactions map[string][]Transaction
	customers    map[string]*Customer
	customerMail map[string]*Customer
	bills        map[string]*BillSummary
	webhooks     map[string]bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		payments:     make(map[string]*Payment),
		idempotency:  make(map[string]*Payment),
		refunds:      make(map[string]*Refund),
		transactions: make(map[string][]Transaction),
		customers:    make(map[string]*Customer),
		customerMail: make(map[string]*Customer),
		bills:        make(map[string]*BillSummary),
		webhooks:     make(map[string]bool),
	}
}

func (m *mockRepository) CreatePayment(ctx context.Context, payment *Payment, tx *Transaction) error {
	if _, exists := m.idempotency[payment.IdempotencyKey]; exists {
		return ErrDuplicateIdempotencyKey
	}
	m.payments[payment.ID] = payment
	m.idempotency[payment.IdempotencyKey] = payment
	if tx != nil {
		m.transactions[payment.ID] = append(m.transactions[payment.ID], *tx)
	}
	return nil
}

func (m *mockRepository) GetPaymentByID(ctx context.Context, id string) (*Payment, error) {
	p, ok := m.payments[id]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	copyP := *p
	return &copyP, nil
}

func (m *mockRepository) GetPaymentByIdempotencyKey(ctx context.Context, key string) (*Payment, error) {
	p, ok := m.idempotency[key]
	if !ok {
		return nil, ErrPaymentNotFound
	}
	copyP := *p
	return &copyP, nil
}

func (m *mockRepository) ListPayments(ctx context.Context, customerID string, status string, provider string, page int, limit int) ([]Payment, int, error) {
	var result []Payment
	for _, p := range m.payments {
		if customerID != "" && p.CustomerID != customerID {
			continue
		}
		if status != "" && string(p.Status) != status {
			continue
		}
		if provider != "" && string(p.Provider) != provider {
			continue
		}
		result = append(result, *p)
	}
	return result, len(result), nil
}

func (m *mockRepository) UpdatePayment(ctx context.Context, payment *Payment) error {
	if _, ok := m.payments[payment.ID]; !ok {
		return ErrPaymentNotFound
	}
	m.payments[payment.ID] = payment
	return nil
}

func (m *mockRepository) UpdatePaymentWithTransaction(ctx context.Context, payment *Payment, tx *Transaction) error {
	if _, ok := m.payments[payment.ID]; !ok {
		return ErrPaymentNotFound
	}
	m.payments[payment.ID] = payment
	if tx != nil {
		m.transactions[payment.ID] = append(m.transactions[payment.ID], *tx)
	}
	return nil
}

func (m *mockRepository) CreateRefund(ctx context.Context, refund *Refund, payment *Payment, tx *Transaction) error {
	m.refunds[refund.ID] = refund
	m.payments[payment.ID] = payment
	if tx != nil {
		m.transactions[payment.ID] = append(m.transactions[payment.ID], *tx)
	}
	return nil
}

func (m *mockRepository) GetRefundByID(ctx context.Context, id string) (*Refund, error) {
	r, ok := m.refunds[id]
	if !ok {
		return nil, ErrRefundNotFound
	}
	copyR := *r
	return &copyR, nil
}

func (m *mockRepository) ListRefunds(ctx context.Context, paymentID string, page int, limit int) ([]Refund, int, error) {
	var result []Refund
	for _, r := range m.refunds {
		if paymentID != "" && r.PaymentID != paymentID {
			continue
		}
		result = append(result, *r)
	}
	return result, len(result), nil
}

func (m *mockRepository) GetTotalRefundedAmount(ctx context.Context, paymentID string) (float64, error) {
	var total float64
	for _, r := range m.refunds {
		if r.PaymentID == paymentID && r.Status != RefundStatusFailed {
			total += r.Amount
		}
	}
	return total, nil
}

func (m *mockRepository) CreateTransaction(ctx context.Context, tx *Transaction) error {
	m.transactions[tx.PaymentID] = append(m.transactions[tx.PaymentID], *tx)
	return nil
}

func (m *mockRepository) ListTransactionsByPaymentID(ctx context.Context, paymentID string) ([]Transaction, error) {
	return m.transactions[paymentID], nil
}

func (m *mockRepository) CreateCustomer(ctx context.Context, customer *Customer) error {
	if _, exists := m.customerMail[customer.Email]; exists {
		return ErrCustomerAlreadyExists
	}
	m.customers[customer.ID] = customer
	m.customerMail[customer.Email] = customer
	return nil
}

func (m *mockRepository) GetCustomerByID(ctx context.Context, id string) (*Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	copyC := *c
	return &copyC, nil
}

func (m *mockRepository) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	c, ok := m.customerMail[email]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	copyC := *c
	return &copyC, nil
}

func (m *mockRepository) ListCustomers(ctx context.Context, page int, limit int) ([]Customer, int, error) {
	var list []Customer
	for _, c := range m.customers {
		list = append(list, *c)
	}
	return list, len(list), nil
}

func (m *mockRepository) ListPaymentsByBillID(ctx context.Context, billID string) ([]Payment, error) {
	var list []Payment
	for _, p := range m.payments {
		if p.BillID != nil && *p.BillID == billID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockRepository) GetBillSummary(ctx context.Context, billID string) (*BillSummary, error) {
	b, ok := m.bills[billID]
	if !ok {
		return nil, ErrBillNotFound
	}
	copyB := *b
	return &copyB, nil
}

func (m *mockRepository) RecordWebhookEvent(ctx context.Context, provider string, eventID string, eventType string, payload string) (bool, error) {
	key := provider + ":" + eventID
	if m.webhooks[key] {
		return false, nil
	}
	m.webhooks[key] = true
	return true, nil
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

func TestCreatePayment_Success(t *testing.T) {
	repo := newMockRepository()
	logger := zap.NewNop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// Seed customer
	_ = repo.CreateCustomer(ctx, &Customer{
		ID:    "cust_12345",
		Email: "jane@example.com",
	})

	req := CreatePaymentRequest{
		IdempotencyKey: "ik_unique_1001",
		CustomerID:     "cust_12345",
		Amount:         250.00,
		Currency:       "USD",
		Provider:       ProviderStripe,
		Description:    "Monthly Cloud Subscription",
	}

	res, err := svc.CreatePayment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error creating payment: %v", err)
	}

	if res.ID == "" {
		t.Errorf("expected non-empty payment ID")
	}
	if res.Status != PaymentStatusSucceeded {
		t.Errorf("expected status %v, got %v", PaymentStatusSucceeded, res.Status)
	}
	if res.Amount != 250.00 {
		t.Errorf("expected amount 250.00, got %v", res.Amount)
	}
	if res.PaymentURL == nil {
		t.Errorf("expected payment URL for Stripe")
	}

	// Verify idempotency: calling again with same key returns existing payment
	res2, err2 := svc.CreatePayment(ctx, req)
	if err2 != nil {
		t.Fatalf("unexpected error on idempotent call: %v", err2)
	}
	if res2.ID != res.ID {
		t.Errorf("expected idempotent ID %s, got %s", res.ID, res2.ID)
	}
}

func TestCreatePayment_CustomerNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	req := CreatePaymentRequest{
		IdempotencyKey: "ik_fail_cust",
		CustomerID:     "non_existent_cust",
		Amount:         100.00,
		Currency:       "USD",
		Provider:       ProviderStripe,
	}

	_, err := svc.CreatePayment(ctx, req)
	if !errors.Is(err, ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestGetPaymentByID(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	_ = repo.CreateCustomer(ctx, &Customer{ID: "cust_1", Email: "c1@test.com"})

	res, err := svc.CreatePayment(ctx, CreatePaymentRequest{
		IdempotencyKey: "ik_get_1",
		CustomerID:     "cust_1",
		Amount:         120.00,
		Currency:       "USD",
		Provider:       ProviderPayPal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	detail, err := svc.GetPaymentByID(ctx, res.ID)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if detail.ID != res.ID {
		t.Errorf("expected payment ID %s, got %s", res.ID, detail.ID)
	}
	if len(detail.Transactions) == 0 {
		t.Errorf("expected at least 1 transaction record")
	}

	// Non-existent payment
	_, err = svc.GetPaymentByID(ctx, "invalid_id")
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestListPayments(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	_ = repo.CreateCustomer(ctx, &Customer{ID: "cust_list", Email: "list@test.com"})

	for i := 1; i <= 3; i++ {
		_, _ = svc.CreatePayment(ctx, CreatePaymentRequest{
			IdempotencyKey: time.Now().String() + string(rune(i)),
			CustomerID:     "cust_list",
			Amount:         float64(i * 50),
			Currency:       "USD",
			Provider:       ProviderSSLCommerz,
		})
	}

	list, err := svc.ListPayments(ctx, "cust_list", "", "", 1, 10)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if list.Total != 3 {
		t.Errorf("expected total 3, got %d", list.Total)
	}
}

func TestProcessRefund_FullAndPartial(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	_ = repo.CreateCustomer(ctx, &Customer{ID: "cust_ref", Email: "ref@test.com"})

	// 1. Create a payment of 100.00
	payRes, err := svc.CreatePayment(ctx, CreatePaymentRequest{
		IdempotencyKey: "ik_refund_test",
		CustomerID:     "cust_ref",
		Amount:         100.00,
		Currency:       "USD",
		Provider:       ProviderStripe,
	})
	if err != nil {
		t.Fatalf("failed to create payment: %v", err)
	}

	// 2. Partial refund of 40.00
	refRes1, err := svc.ProcessRefund(ctx, CreateRefundRequest{
		PaymentID: payRes.ID,
		Amount:    40.00,
		Reason:    "Partial customer return",
	})
	if err != nil {
		t.Fatalf("failed to process partial refund: %v", err)
	}
	if refRes1.Status != RefundStatusSucceeded {
		t.Errorf("expected refund status SUCCEEDED, got %v", refRes1.Status)
	}

	// Verify payment transitioned to PARTIALLY_REFUNDED
	payDetail, err := svc.GetPaymentByID(ctx, payRes.ID)
	if err != nil {
		t.Fatalf("failed to get payment: %v", err)
	}
	if payDetail.Status != PaymentStatusPartiallyRefunded {
		t.Errorf("expected status PARTIALLY_REFUNDED, got %v", payDetail.Status)
	}

	// 3. Attempt refund exceeding remaining balance (remaining: 60.00, trying: 70.00)
	_, err = svc.ProcessRefund(ctx, CreateRefundRequest{
		PaymentID: payRes.ID,
		Amount:    70.00,
	})
	if !errors.Is(err, ErrRefundAmountExceeded) {
		t.Fatalf("expected ErrRefundAmountExceeded, got %v", err)
	}

	// 4. Remaining full refund of 60.00
	refRes2, err := svc.ProcessRefund(ctx, CreateRefundRequest{
		PaymentID: payRes.ID,
		Amount:    60.00,
		Reason:    "Remaining balance refund",
	})
	if err != nil {
		t.Fatalf("failed to process final refund: %v", err)
	}
	if refRes2.Amount != 60.00 {
		t.Errorf("expected 60.00 refund amount, got %v", refRes2.Amount)
	}

	// Verify payment transitioned to REFUNDED
	payDetailFinal, _ := svc.GetPaymentByID(ctx, payRes.ID)
	if payDetailFinal.Status != PaymentStatusRefunded {
		t.Errorf("expected status REFUNDED, got %v", payDetailFinal.Status)
	}
	if len(payDetailFinal.Refunds) != 2 {
		t.Errorf("expected 2 refund records, got %d", len(payDetailFinal.Refunds))
	}
}

func TestCancelPayment(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	_ = repo.CreateCustomer(ctx, &Customer{ID: "cust_c", Email: "c@test.com"})

	// Manually insert a PENDING payment
	pendingPayment := &Payment{
		ID:             "pay_pending_1",
		IdempotencyKey: "ik_pending_1",
		CustomerID:     "cust_c",
		Amount:         80.00,
		Currency:       "USD",
		Provider:       ProviderBKash,
		Status:         PaymentStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = repo.CreatePayment(ctx, pendingPayment, nil)

	cancelRes, err := svc.CancelPayment(ctx, pendingPayment.ID, CancelPaymentRequest{
		Reason: "User changed mind",
	})
	if err != nil {
		t.Fatalf("failed to cancel payment: %v", err)
	}
	if cancelRes.Status != PaymentStatusCancelled {
		t.Errorf("expected CANCELLED status, got %v", cancelRes.Status)
	}

	// Cannot cancel already cancelled payment
	_, err = svc.CancelPayment(ctx, pendingPayment.ID, CancelPaymentRequest{})
	if !errors.Is(err, ErrInvalidPaymentStatus) {
		t.Fatalf("expected ErrInvalidPaymentStatus on second cancel, got %v", err)
	}
}

func TestCustomerManagement(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	// 1. Create customer
	custRes, err := svc.CreateCustomer(ctx, CreateCustomerRequest{
		Email: "alex@enterprise.com",
		Name:  "Alex Morgan",
		Phone: "+1555123456",
	})
	if err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}
	if custRes.Email != "alex@enterprise.com" {
		t.Errorf("expected email alex@enterprise.com, got %s", custRes.Email)
	}

	// 2. Duplicate email should conflict
	_, err = svc.CreateCustomer(ctx, CreateCustomerRequest{
		Email: "alex@enterprise.com",
	})
	if !errors.Is(err, ErrCustomerAlreadyExists) {
		t.Fatalf("expected ErrCustomerAlreadyExists, got %v", err)
	}

	// 3. Get customer by ID
	fetched, err := svc.GetCustomerByID(ctx, custRes.ID)
	if err != nil {
		t.Fatalf("failed to get customer: %v", err)
	}
	if fetched.ID != custRes.ID {
		t.Errorf("expected customer ID %s, got %s", custRes.ID, fetched.ID)
	}

	// 4. List customers
	custList, err := svc.ListCustomers(ctx, 1, 10)
	if err != nil {
		t.Fatalf("failed to list customers: %v", err)
	}
	if custList.Total != 1 {
		t.Errorf("expected 1 customer, got %d", custList.Total)
	}
}

func TestPaymentWithLinkedBill(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	// 1. Seed customer and bill
	cust := &Customer{ID: "cust_bridge_1", Email: "bridge@test.com"}
	_ = repo.CreateCustomer(ctx, cust)

	repo.bills["bill_bridge_1"] = &BillSummary{
		ID:          "bill_bridge_1",
		CustomerID:  "cust_bridge_1",
		TotalAmount: 200.0,
		PaidAmount:  0.0,
		Currency:    "USD",
		Status:      "PENDING",
	}

	billID := "bill_bridge_1"

	// 2. Customer mismatch should fail
	_, err := svc.CreatePayment(ctx, CreatePaymentRequest{
		IdempotencyKey: "ik_mismatch",
		BillID:         &billID,
		CustomerID:     "different_customer",
		Amount:         100.0,
		Currency:       "USD",
		Provider:       ProviderStripe,
	})
	if err == nil {
		t.Fatal("expected error on customer mismatch, got nil")
	}

	// 3. Amount exceeding remaining should fail
	_, err = svc.CreatePayment(ctx, CreatePaymentRequest{
		IdempotencyKey: "ik_exceed",
		BillID:         &billID,
		CustomerID:     "cust_bridge_1",
		Amount:         250.0,
		Currency:       "USD",
		Provider:       ProviderStripe,
	})
	if err == nil {
		t.Fatal("expected error on amount exceeding remaining balance, got nil")
	}

	// 4. Successful payment linked to bill
	payRes, err := svc.CreatePayment(ctx, CreatePaymentRequest{
		IdempotencyKey: "ik_success_bill_pay",
		BillID:         &billID,
		CustomerID:     "cust_bridge_1",
		Amount:         150.0,
		Currency:       "USD",
		Provider:       ProviderStripe,
	})
	if err != nil {
		t.Fatalf("unexpected error creating linked payment: %v", err)
	}
	if payRes.BillID == nil || *payRes.BillID != "bill_bridge_1" {
		t.Errorf("expected BillID bill_bridge_1, got %v", payRes.BillID)
	}

	// 5. Query payments by bill ID
	billPayments, err := svc.ListPaymentsByBillID(ctx, "bill_bridge_1")
	if err != nil {
		t.Fatalf("unexpected error listing payments by bill ID: %v", err)
	}
	if len(billPayments) != 1 {
		t.Fatalf("expected 1 payment for bill, got %d", len(billPayments))
	}
}

func TestHandleWebhook_SucceedsPayment(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	// 1. Seed pending payment
	payment := &Payment{
		ID:             "pay_webhook_test",
		IdempotencyKey: "ik_webhook_test",
		CustomerID:     "cust_1",
		Amount:         100.0,
		Currency:       "USD",
		Provider:       ProviderStripe,
		Status:         PaymentStatusPending,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.payments[payment.ID] = payment

	// 2. Incoming webhook payload
	payload := []byte(`{"id": "evt_stripe_123", "type": "payment_intent.succeeded", "payment_id": "pay_webhook_test"}`)

	res, err := svc.HandleWebhook(ctx, ProviderStripe, payload, "dummy_sig")
	if err != nil {
		t.Fatalf("unexpected webhook error: %v", err)
	}
	if res == nil || res.Status != PaymentStatusSucceeded {
		t.Fatalf("expected payment status to transition to SUCCEEDED, got %v", res)
	}

	// 3. Second identical webhook should be deduplicated (idempotent ignore)
	res2, err := svc.HandleWebhook(ctx, ProviderStripe, payload, "dummy_sig")
	if err != nil {
		t.Fatalf("unexpected error on deduplicated webhook: %v", err)
	}
	if res2 != nil {
		t.Errorf("expected nil on duplicate webhook event, got %v", res2)
	}
}
