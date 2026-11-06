package billing

import (
	"context"
	"testing"
	"time"

	"ledgerflow-payment-service/internal/modules/payment"

	"go.uber.org/zap"
)

type mockBillingRepo struct {
	bills map[string]*Bill
}

func newMockBillingRepo() *mockBillingRepo {
	return &mockBillingRepo{
		bills: make(map[string]*Bill),
	}
}

func (m *mockBillingRepo) Create(ctx context.Context, bill *Bill, items []BillItem) error {
	bill.Items = items
	m.bills[bill.ID] = bill
	return nil
}

func (m *mockBillingRepo) GetByID(ctx context.Context, id string) (*Bill, error) {
	b, ok := m.bills[id]
	if !ok {
		return nil, ErrBillNotFound
	}
	copyB := *b
	return &copyB, nil
}

func (m *mockBillingRepo) List(ctx context.Context, customerID string, status string, page int, limit int) ([]Bill, int, error) {
	var result []Bill
	for _, b := range m.bills {
		if customerID != "" && b.CustomerID != customerID {
			continue
		}
		if status != "" && string(b.Status) != status {
			continue
		}
		result = append(result, *b)
	}
	return result, len(result), nil
}

func (m *mockBillingRepo) Update(ctx context.Context, bill *Bill, items []BillItem) error {
	if _, ok := m.bills[bill.ID]; !ok {
		return ErrBillNotFound
	}
	if items != nil {
		bill.Items = items
	}
	m.bills[bill.ID] = bill
	return nil
}

func (m *mockBillingRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.bills[id]; !ok {
		return ErrBillNotFound
	}
	delete(m.bills, id)
	return nil
}

type mockPaymentService struct {
	payment.IService
	createdPayments []payment.CreatePaymentRequest
	paymentsByBill  map[string][]payment.PaymentResponse
}

func newMockPaymentService() *mockPaymentService {
	return &mockPaymentService{
		paymentsByBill: make(map[string][]payment.PaymentResponse),
	}
}

func (m *mockPaymentService) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.PaymentResponse, error) {
	m.createdPayments = append(m.createdPayments, req)
	res := &payment.PaymentResponse{
		ID:             "pay_test_123",
		IdempotencyKey: req.IdempotencyKey,
		BillID:         req.BillID,
		CustomerID:     req.CustomerID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Provider:       req.Provider,
		Status:         payment.PaymentStatusSucceeded,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	if req.BillID != nil {
		m.paymentsByBill[*req.BillID] = append(m.paymentsByBill[*req.BillID], *res)
	}
	return res, nil
}

func (m *mockPaymentService) ListPaymentsByBillID(ctx context.Context, billID string) ([]payment.PaymentResponse, error) {
	return m.paymentsByBill[billID], nil
}

func TestPayBill_Success(t *testing.T) {
	repo := newMockBillingRepo()
	paySvc := newMockPaymentService()
	logger := zap.NewNop()
	svc := NewService(repo, paySvc, logger)
	ctx := context.Background()

	bill := &Bill{
		ID:          "bill_100",
		BillNumber:  "INV-2026-0001",
		CustomerID:  "cust_1",
		TotalAmount: 200.0,
		PaidAmount:  0.0,
		Currency:    "USD",
		Status:      BillStatusPending,
		DueDate:     time.Now().AddDate(0, 0, 14),
	}
	_ = repo.Create(ctx, bill, nil)

	req := PayBillRequest{
		IdempotencyKey: "ik_pay_bill_100",
		Provider:       payment.ProviderStripe,
	}

	payRes, err := svc.PayBill(ctx, "bill_100", req)
	if err != nil {
		t.Fatalf("unexpected error paying bill: %v", err)
	}

	if payRes.Amount != 200.0 {
		t.Errorf("expected full remaining balance 200.0 to be charged, got %.2f", payRes.Amount)
	}
	if payRes.BillID == nil || *payRes.BillID != "bill_100" {
		t.Errorf("expected payment to link to bill_100, got %v", payRes.BillID)
	}
	if len(paySvc.createdPayments) != 1 {
		t.Fatalf("expected 1 payment call, got %d", len(paySvc.createdPayments))
	}
}

func TestPayBill_PartialPayment(t *testing.T) {
	repo := newMockBillingRepo()
	paySvc := newMockPaymentService()
	logger := zap.NewNop()
	svc := NewService(repo, paySvc, logger)
	ctx := context.Background()

	bill := &Bill{
		ID:          "bill_101",
		BillNumber:  "INV-2026-0002",
		CustomerID:  "cust_1",
		TotalAmount: 300.0,
		PaidAmount:  100.0,
		Currency:    "USD",
		Status:      BillStatusPartiallyPaid,
		DueDate:     time.Now().AddDate(0, 0, 14),
	}
	_ = repo.Create(ctx, bill, nil)

	partialAmount := 50.0
	req := PayBillRequest{
		IdempotencyKey: "ik_partial_pay",
		Amount:         &partialAmount,
		Provider:       payment.ProviderStripe,
	}

	payRes, err := svc.PayBill(ctx, "bill_101", req)
	if err != nil {
		t.Fatalf("unexpected error paying bill partially: %v", err)
	}

	if payRes.Amount != 50.0 {
		t.Errorf("expected 50.0 to be charged, got %.2f", payRes.Amount)
	}
}

func TestPayBill_ExceedsBalance(t *testing.T) {
	repo := newMockBillingRepo()
	paySvc := newMockPaymentService()
	logger := zap.NewNop()
	svc := NewService(repo, paySvc, logger)
	ctx := context.Background()

	bill := &Bill{
		ID:          "bill_102",
		BillNumber:  "INV-2026-0003",
		CustomerID:  "cust_1",
		TotalAmount: 100.0,
		PaidAmount:  80.0,
		Currency:    "USD",
		Status:      BillStatusPartiallyPaid,
		DueDate:     time.Now().AddDate(0, 0, 14),
	}
	_ = repo.Create(ctx, bill, nil)

	exceedAmount := 50.0 // remaining is only 20
	req := PayBillRequest{
		IdempotencyKey: "ik_exceed_pay",
		Amount:         &exceedAmount,
		Provider:       payment.ProviderStripe,
	}

	_, err := svc.PayBill(ctx, "bill_102", req)
	if err == nil {
		t.Fatal("expected error when amount exceeds remaining balance, got nil")
	}
}

func TestPayBill_AlreadyPaid(t *testing.T) {
	repo := newMockBillingRepo()
	paySvc := newMockPaymentService()
	logger := zap.NewNop()
	svc := NewService(repo, paySvc, logger)
	ctx := context.Background()

	bill := &Bill{
		ID:          "bill_103",
		BillNumber:  "INV-2026-0004",
		CustomerID:  "cust_1",
		TotalAmount: 100.0,
		PaidAmount:  100.0,
		Currency:    "USD",
		Status:      BillStatusPaid,
		DueDate:     time.Now().AddDate(0, 0, 14),
	}
	_ = repo.Create(ctx, bill, nil)

	req := PayBillRequest{
		IdempotencyKey: "ik_already_paid",
		Provider:       payment.ProviderStripe,
	}

	_, err := svc.PayBill(ctx, "bill_103", req)
	if err == nil {
		t.Fatal("expected error when paying already paid bill, got nil")
	}
}

func TestGetBillPayments(t *testing.T) {
	repo := newMockBillingRepo()
	paySvc := newMockPaymentService()
	logger := zap.NewNop()
	svc := NewService(repo, paySvc, logger)
	ctx := context.Background()

	bill := &Bill{
		ID:          "bill_104",
		BillNumber:  "INV-2026-0005",
		CustomerID:  "cust_1",
		TotalAmount: 150.0,
		PaidAmount:  150.0,
		Currency:    "USD",
		Status:      BillStatusPaid,
		DueDate:     time.Now().AddDate(0, 0, 14),
	}
	_ = repo.Create(ctx, bill, nil)

	// Add dummy payment to mock payment service
	paySvc.paymentsByBill["bill_104"] = []payment.PaymentResponse{
		{
			ID:         "pay_104_1",
			BillID:     &bill.ID,
			CustomerID: bill.CustomerID,
			Amount:     150.0,
			Currency:   bill.Currency,
			Provider:   payment.ProviderStripe,
			Status:     payment.PaymentStatusSucceeded,
		},
	}

	res, err := svc.GetBillPayments(ctx, "bill_104")
	if err != nil {
		t.Fatalf("unexpected error getting bill payments: %v", err)
	}

	if len(res.Payments) != 1 {
		t.Fatalf("expected 1 payment in response, got %d", len(res.Payments))
	}
	if res.Payments[0].ID != "pay_104_1" {
		t.Errorf("expected payment ID pay_104_1, got %s", res.Payments[0].ID)
	}
}
