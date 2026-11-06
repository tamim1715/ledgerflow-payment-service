package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInvalidPaymentStatus = errors.New("invalid payment status for requested operation")
	ErrRefundAmountExceeded = errors.New("refund amount exceeds refundable balance")
)

type IService interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentResponse, error)
	GetPaymentByID(ctx context.Context, id string) (*PaymentDetailResponse, error)
	ListPayments(ctx context.Context, customerID string, status string, provider string, page int, limit int) (*PaymentListResponse, error)
	CapturePayment(ctx context.Context, id string) (*PaymentResponse, error)
	CancelPayment(ctx context.Context, id string, req CancelPaymentRequest) (*PaymentResponse, error)

	ProcessRefund(ctx context.Context, req CreateRefundRequest) (*RefundResponse, error)
	GetRefundByID(ctx context.Context, id string) (*RefundResponse, error)
	ListRefunds(ctx context.Context, paymentID string, page int, limit int) (*RefundListResponse, error)

	ListTransactions(ctx context.Context, paymentID string) ([]TransactionResponse, error)
	ListPaymentsByBillID(ctx context.Context, billID string) ([]PaymentResponse, error)

	HandleWebhook(ctx context.Context, provider PaymentProvider, payload []byte, signature string) (*PaymentResponse, error)

	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error)
	GetCustomerByID(ctx context.Context, id string) (*CustomerResponse, error)
	ListCustomers(ctx context.Context, page int, limit int) (*CustomerListResponse, error)
}

type service struct {
	repo   IRepository
	logger *zap.Logger
}

func NewService(repo IRepository, logger *zap.Logger) IService {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

// ---------------------------------------------------------------------------
// Payment Operations
// ---------------------------------------------------------------------------

func (s *service) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentResponse, error) {
	// 1. Check idempotency: Return existing payment if idempotency key was previously processed
	existingPayment, err := s.repo.GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existingPayment != nil {
		s.logger.Info("Returning idempotent payment response",
			zap.String("idempotency_key", req.IdempotencyKey),
			zap.String("payment_id", existingPayment.ID),
		)
		return toPaymentResponse(existingPayment, nil), nil
	}

	// 2. Validate Customer exists
	customer, err := s.repo.GetCustomerByID(ctx, req.CustomerID)
	if err != nil {
		s.logger.Warn("Payment customer validation failed",
			zap.String("customer_id", req.CustomerID),
			zap.Error(err),
		)
		return nil, ErrCustomerNotFound
	}

	// 2.1 Validate linked Bill exists and is payable, if bill_id is provided
	if req.BillID != nil && *req.BillID != "" {
		bill, err := s.repo.GetBillSummary(ctx, *req.BillID)
		if err != nil {
			s.logger.Warn("Payment bill validation failed", zap.String("bill_id", *req.BillID), zap.Error(err))
			return nil, ErrBillNotFound
		}
		if bill.CustomerID != req.CustomerID {
			return nil, fmt.Errorf("%w: invoice %s belongs to a different customer", ErrInvalidPaymentRequest, *req.BillID)
		}
		if bill.Currency != req.Currency {
			return nil, fmt.Errorf("%w: payment currency %s does not match invoice currency %s", ErrInvalidPaymentRequest, req.Currency, bill.Currency)
		}
		if bill.Status == "PAID" {
			return nil, fmt.Errorf("%w: invoice %s is already fully paid", ErrInvalidPaymentRequest, *req.BillID)
		}
		if bill.Status == "CANCELLED" {
			return nil, fmt.Errorf("%w: cannot process payment for cancelled invoice %s", ErrInvalidPaymentRequest, *req.BillID)
		}
		remainingBalance := bill.TotalAmount - bill.PaidAmount
		if req.Amount > remainingBalance {
			return nil, fmt.Errorf("%w: payment amount %.2f exceeds remaining balance %.2f", ErrInvalidPaymentRequest, req.Amount, remainingBalance)
		}
	}

	now := time.Now().UTC()
	paymentID := fmt.Sprintf("pay_%s", uuid.New().String()[:18])

	var metadataStr *string
	if len(req.Metadata) > 0 {
		bytes, err := json.Marshal(req.Metadata)
		if err == nil {
			str := string(bytes)
			metadataStr = &str
		}
	}

	var descPtr *string
	if req.Description != "" {
		descPtr = &req.Description
	}

	payment := &Payment{
		ID:             paymentID,
		IdempotencyKey: req.IdempotencyKey,
		BillID:         req.BillID,
		CustomerID:     req.CustomerID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Provider:       req.Provider,
		Status:         PaymentStatusProcessing,
		Description:    descPtr,
		Metadata:       metadataStr,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 3. Initiate Payment via Gateway Provider
	gateway := NewGateway(req.Provider)
	gwResult, err := gateway.InitiatePayment(ctx, payment, customer)
	if err != nil {
		s.logger.Error("Gateway payment initiation error", zap.Error(err))
		return nil, fmt.Errorf("gateway initiate payment: %w", err)
	}

	// Update payment status from gateway
	payment.Status = gwResult.PaymentStatus

	var rawResponseStr *string
	if gwResult.RawResponse != nil {
		bytes, err := json.Marshal(gwResult.RawResponse)
		if err == nil {
			str := string(bytes)
			rawResponseStr = &str
		}
	}

	tx := &Transaction{
		ID:                    fmt.Sprintf("tx_%s", uuid.New().String()[:18]),
		PaymentID:             paymentID,
		ProviderTransactionID: &gwResult.ProviderTxID,
		Type:                  TxTypeCharge,
		Amount:                req.Amount,
		Currency:              req.Currency,
		Status:                gwResult.Status,
		RawResponse:           rawResponseStr,
		ErrorCode:             gwResult.ErrorCode,
		ErrorMessage:          gwResult.ErrorMessage,
		CreatedAt:             now,
	}

	// 4. Save Payment and Ledger Transaction
	if err := s.repo.CreatePayment(ctx, payment, tx); err != nil {
		s.logger.Error("Failed to persist payment and transaction", zap.Error(err))
		return nil, fmt.Errorf("create payment: %w", err)
	}

	s.logger.Info("Payment created successfully",
		zap.String("id", payment.ID),
		zap.String("provider", string(payment.Provider)),
		zap.Float64("amount", payment.Amount),
		zap.String("status", string(payment.Status)),
	)

	return toPaymentResponse(payment, gwResult.PaymentURL), nil
}

func (s *service) GetPaymentByID(ctx context.Context, id string) (*PaymentDetailResponse, error) {
	payment, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	txs, err := s.repo.ListTransactionsByPaymentID(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to load transactions for payment", zap.String("id", id), zap.Error(err))
		txs = []Transaction{}
	}

	refunds, _, err := s.repo.ListRefunds(ctx, id, 1, 100)
	if err != nil {
		s.logger.Warn("Failed to load refunds for payment", zap.String("id", id), zap.Error(err))
		refunds = []Refund{}
	}

	txResponses := make([]TransactionResponse, len(txs))
	for i, t := range txs {
		txResponses[i] = *toTransactionResponse(&t)
	}

	refundResponses := make([]RefundResponse, len(refunds))
	for i, r := range refunds {
		refundResponses[i] = *toRefundResponse(&r)
	}

	baseRes := toPaymentResponse(payment, nil)

	return &PaymentDetailResponse{
		PaymentResponse: *baseRes,
		Transactions:    txResponses,
		Refunds:         refundResponses,
	}, nil
}

func (s *service) ListPayments(ctx context.Context, customerID string, status string, provider string, page int, limit int) (*PaymentListResponse, error) {
	payments, total, err := s.repo.ListPayments(ctx, customerID, status, provider, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}

	responses := make([]PaymentResponse, len(payments))
	for i, p := range payments {
		responses[i] = *toPaymentResponse(&p, nil)
	}

	return &PaymentListResponse{
		Payments: responses,
		Total:    total,
		Page:     page,
		Limit:    limit,
	}, nil
}

func (s *service) CapturePayment(ctx context.Context, id string) (*PaymentResponse, error) {
	payment, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payment.Status != PaymentStatusPending && payment.Status != PaymentStatusProcessing {
		return nil, fmt.Errorf("%w: cannot capture payment with status %s", ErrInvalidPaymentStatus, payment.Status)
	}

	now := time.Now().UTC()
	payment.Status = PaymentStatusSucceeded
	payment.UpdatedAt = now

	rawResponse := map[string]any{
		"action":      "capture",
		"payment_id":  payment.ID,
		"captured_at": now.Format(time.RFC3339),
		"status":      "succeeded",
	}
	bytes, _ := json.Marshal(rawResponse)
	rawStr := string(bytes)
	txID := fmt.Sprintf("ch_cap_%s", uuid.New().String()[:14])

	tx := &Transaction{
		ID:                    fmt.Sprintf("tx_%s", uuid.New().String()[:18]),
		PaymentID:             payment.ID,
		ProviderTransactionID: &txID,
		Type:                  TxTypeCharge,
		Amount:                payment.Amount,
		Currency:              payment.Currency,
		Status:                TxStatusSuccess,
		RawResponse:           &rawStr,
		CreatedAt:             now,
	}

	if err := s.repo.UpdatePaymentWithTransaction(ctx, payment, tx); err != nil {
		return nil, fmt.Errorf("capture payment: %w", err)
	}

	s.logger.Info("Payment captured successfully", zap.String("id", payment.ID))
	return toPaymentResponse(payment, nil), nil
}

func (s *service) CancelPayment(ctx context.Context, id string, req CancelPaymentRequest) (*PaymentResponse, error) {
	payment, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payment.Status == PaymentStatusSucceeded || payment.Status == PaymentStatusRefunded || payment.Status == PaymentStatusCancelled {
		return nil, fmt.Errorf("%w: cannot cancel payment with status %s", ErrInvalidPaymentStatus, payment.Status)
	}

	now := time.Now().UTC()
	payment.Status = PaymentStatusCancelled
	payment.UpdatedAt = now

	reason := "Payment cancelled"
	if req.Reason != "" {
		reason = req.Reason
	}

	rawResponse := map[string]any{
		"action":       "cancel",
		"payment_id":   payment.ID,
		"cancelled_at": now.Format(time.RFC3339),
		"reason":       reason,
	}
	bytes, _ := json.Marshal(rawResponse)
	rawStr := string(bytes)

	tx := &Transaction{
		ID:           fmt.Sprintf("tx_%s", uuid.New().String()[:18]),
		PaymentID:    payment.ID,
		Type:         TxTypeReversal,
		Amount:       payment.Amount,
		Currency:     payment.Currency,
		Status:       TxStatusSuccess,
		RawResponse:  &rawStr,
		ErrorMessage: &reason,
		CreatedAt:    now,
	}

	if err := s.repo.UpdatePaymentWithTransaction(ctx, payment, tx); err != nil {
		return nil, fmt.Errorf("cancel payment: %w", err)
	}

	s.logger.Info("Payment cancelled successfully", zap.String("id", payment.ID), zap.String("reason", reason))
	return toPaymentResponse(payment, nil), nil
}

// ---------------------------------------------------------------------------
// Refund Operations
// ---------------------------------------------------------------------------

func (s *service) ProcessRefund(ctx context.Context, req CreateRefundRequest) (*RefundResponse, error) {
	payment, err := s.repo.GetPaymentByID(ctx, req.PaymentID)
	if err != nil {
		return nil, err
	}

	// Only SUCCEEDED and PARTIALLY_REFUNDED payments can be refunded
	if payment.Status != PaymentStatusSucceeded && payment.Status != PaymentStatusPartiallyRefunded {
		return nil, fmt.Errorf("%w: payment with status %s cannot be refunded", ErrInvalidPaymentStatus, payment.Status)
	}

	// Verify refund amount against total already refunded
	totalRefunded, err := s.repo.GetTotalRefundedAmount(ctx, payment.ID)
	if err != nil {
		return nil, fmt.Errorf("calculate refunded amount: %w", err)
	}

	refundableBalance := payment.Amount - totalRefunded
	if req.Amount > refundableBalance {
		return nil, fmt.Errorf("%w: requested amount %.2f exceeds refundable balance %.2f (total paid: %.2f, already refunded: %.2f)",
			ErrRefundAmountExceeded, req.Amount, refundableBalance, payment.Amount, totalRefunded)
	}

	now := time.Now().UTC()
	refundID := fmt.Sprintf("ref_%s", uuid.New().String()[:18])

	var reasonPtr *string
	if req.Reason != "" {
		reasonPtr = &req.Reason
	}

	refund := &Refund{
		ID:        refundID,
		PaymentID: payment.ID,
		Amount:    req.Amount,
		Currency:  payment.Currency,
		Reason:    reasonPtr,
		Status:    RefundStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Process refund with gateway provider
	gateway := NewGateway(payment.Provider)
	gwRefundResult, err := gateway.ProcessRefund(ctx, payment, refund)
	if err != nil {
		s.logger.Error("Gateway refund processing failed", zap.Error(err))
		return nil, fmt.Errorf("gateway process refund: %w", err)
	}

	refund.ProviderRefundID = &gwRefundResult.ProviderRefundID
	refund.Status = gwRefundResult.Status

	// Determine new payment status
	newTotalRefunded := totalRefunded + req.Amount
	if newTotalRefunded >= payment.Amount {
		payment.Status = PaymentStatusRefunded
	} else {
		payment.Status = PaymentStatusPartiallyRefunded
	}
	payment.UpdatedAt = now

	var rawResponseStr *string
	if gwRefundResult.RawResponse != nil {
		bytes, err := json.Marshal(gwRefundResult.RawResponse)
		if err == nil {
			str := string(bytes)
			rawResponseStr = &str
		}
	}

	tx := &Transaction{
		ID:                    fmt.Sprintf("tx_%s", uuid.New().String()[:18]),
		PaymentID:             payment.ID,
		ProviderTransactionID: &gwRefundResult.ProviderRefundID,
		Type:                  TxTypeRefund,
		Amount:                req.Amount,
		Currency:              payment.Currency,
		Status:                TxStatusSuccess,
		RawResponse:           rawResponseStr,
		ErrorCode:             gwRefundResult.ErrorCode,
		ErrorMessage:          gwRefundResult.ErrorMessage,
		CreatedAt:             now,
	}

	if err := s.repo.CreateRefund(ctx, refund, payment, tx); err != nil {
		s.logger.Error("Failed to persist refund and transaction", zap.Error(err))
		return nil, fmt.Errorf("create refund: %w", err)
	}

	s.logger.Info("Refund processed successfully",
		zap.String("refund_id", refund.ID),
		zap.String("payment_id", payment.ID),
		zap.Float64("amount", refund.Amount),
		zap.String("new_payment_status", string(payment.Status)),
	)

	return toRefundResponse(refund), nil
}

func (s *service) GetRefundByID(ctx context.Context, id string) (*RefundResponse, error) {
	refund, err := s.repo.GetRefundByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toRefundResponse(refund), nil
}

func (s *service) ListRefunds(ctx context.Context, paymentID string, page int, limit int) (*RefundListResponse, error) {
	refunds, total, err := s.repo.ListRefunds(ctx, paymentID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list refunds: %w", err)
	}

	responses := make([]RefundResponse, len(refunds))
	for i, r := range refunds {
		responses[i] = *toRefundResponse(&r)
	}

	return &RefundListResponse{
		Refunds: responses,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

func (s *service) ListTransactions(ctx context.Context, paymentID string) ([]TransactionResponse, error) {
	txs, err := s.repo.ListTransactionsByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	responses := make([]TransactionResponse, len(txs))
	for i, t := range txs {
		responses[i] = *toTransactionResponse(&t)
	}

	return responses, nil
}

func (s *service) ListPaymentsByBillID(ctx context.Context, billID string) ([]PaymentResponse, error) {
	payments, err := s.repo.ListPaymentsByBillID(ctx, billID)
	if err != nil {
		return nil, fmt.Errorf("list payments by bill id: %w", err)
	}

	responses := make([]PaymentResponse, len(payments))
	for i, p := range payments {
		responses[i] = *toPaymentResponse(&p, nil)
	}
	return responses, nil
}

func (s *service) HandleWebhook(ctx context.Context, provider PaymentProvider, payload []byte, signature string) (*PaymentResponse, error) {
	var bodyMap map[string]any
	if err := json.Unmarshal(payload, &bodyMap); err != nil {
		return nil, fmt.Errorf("parse webhook json: %w", err)
	}

	// Extract or generate event identifier for deduplication
	eventID := fmt.Sprintf("evt_%s_%d", provider, time.Now().UnixNano())
	if id, ok := bodyMap["id"].(string); ok && id != "" {
		eventID = id
	}

	eventType := "payment.succeeded"
	if t, ok := bodyMap["type"].(string); ok && t != "" {
		eventType = t
	} else if ev, ok := bodyMap["event"].(string); ok && ev != "" {
		eventType = ev
	}

	// Idempotency check on webhook event
	isNew, err := s.repo.RecordWebhookEvent(ctx, string(provider), eventID, eventType, string(payload))
	if err != nil {
		s.logger.Error("Failed to record webhook event", zap.Error(err))
		return nil, fmt.Errorf("record webhook event: %w", err)
	}
	if !isNew {
		s.logger.Info("Webhook event already processed (idempotent ignore)",
			zap.String("provider", string(provider)),
			zap.String("event_id", eventID),
		)
		return nil, nil
	}

	// Extract target payment ID
	var paymentID string
	if pID, ok := bodyMap["payment_id"].(string); ok {
		paymentID = pID
	} else if data, ok := bodyMap["data"].(map[string]any); ok {
		if obj, ok := data["object"].(map[string]any); ok {
			if meta, ok := obj["metadata"].(map[string]any); ok {
				if mPID, ok := meta["payment_id"].(string); ok {
					paymentID = mPID
				}
			}
		}
	} else if tranID, ok := bodyMap["tran_id"].(string); ok {
		paymentID = tranID
	}

	if paymentID == "" {
		s.logger.Info("Webhook received and logged with no direct payment_id to transition",
			zap.String("provider", string(provider)),
			zap.String("event_type", eventType),
		)
		return nil, nil
	}

	// Lookup payment
	payment, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		s.logger.Warn("Webhook referenced payment not found", zap.String("payment_id", paymentID), zap.Error(err))
		return nil, err
	}

	if payment.Status == PaymentStatusSucceeded {
		return toPaymentResponse(payment, nil), nil
	}

	// Transition payment to SUCCEEDED (which automatically settles linked invoice via repository)
	now := time.Now().UTC()
	payment.Status = PaymentStatusSucceeded
	payment.UpdatedAt = now

	rawStr := string(payload)
	txID := fmt.Sprintf("tx_wh_%s", uuid.New().String()[:14])
	tx := &Transaction{
		ID:                    fmt.Sprintf("tx_%s", uuid.New().String()[:18]),
		PaymentID:             payment.ID,
		ProviderTransactionID: &txID,
		Type:                  TxTypeCharge,
		Amount:                payment.Amount,
		Currency:              payment.Currency,
		Status:                TxStatusSuccess,
		RawResponse:           &rawStr,
		CreatedAt:             now,
	}

	if err := s.repo.UpdatePaymentWithTransaction(ctx, payment, tx); err != nil {
		s.logger.Error("Failed to update payment on webhook", zap.Error(err))
		return nil, fmt.Errorf("update payment on webhook: %w", err)
	}

	s.logger.Info("Payment transitioned to SUCCEEDED via webhook (and linked invoice settled)",
		zap.String("payment_id", payment.ID),
		zap.String("provider", string(provider)),
	)

	return toPaymentResponse(payment, nil), nil
}

// ---------------------------------------------------------------------------
// Customer Operations
// ---------------------------------------------------------------------------

func (s *service) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error) {
	existing, err := s.repo.GetCustomerByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrCustomerAlreadyExists
	}

	now := time.Now().UTC()
	custID := fmt.Sprintf("cust_%s", uuid.New().String()[:12])

	var namePtr *string
	if req.Name != "" {
		namePtr = &req.Name
	}
	var phonePtr *string
	if req.Phone != "" {
		phonePtr = &req.Phone
	}
	var stripeIDPtr *string
	if req.StripeCustomerID != "" {
		stripeIDPtr = &req.StripeCustomerID
	}

	customer := &Customer{
		ID:               custID,
		Email:            req.Email,
		Name:             namePtr,
		Phone:            phonePtr,
		StripeCustomerID: stripeIDPtr,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateCustomer(ctx, customer); err != nil {
		s.logger.Error("Failed to create customer", zap.Error(err))
		return nil, fmt.Errorf("create customer: %w", err)
	}

	s.logger.Info("Customer created successfully", zap.String("id", customer.ID), zap.String("email", customer.Email))
	return toCustomerResponse(customer), nil
}

func (s *service) GetCustomerByID(ctx context.Context, id string) (*CustomerResponse, error) {
	customer, err := s.repo.GetCustomerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCustomerResponse(customer), nil
}

func (s *service) ListCustomers(ctx context.Context, page int, limit int) (*CustomerListResponse, error) {
	customers, total, err := s.repo.ListCustomers(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	responses := make([]CustomerResponse, len(customers))
	for i, c := range customers {
		responses[i] = *toCustomerResponse(&c)
	}

	return &CustomerListResponse{
		Customers: responses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

// ---------------------------------------------------------------------------
// Response Mappers
// ---------------------------------------------------------------------------

func toPaymentResponse(p *Payment, paymentURL *string) *PaymentResponse {
	var metadataMap map[string]any
	if p.Metadata != nil && *p.Metadata != "" {
		_ = json.Unmarshal([]byte(*p.Metadata), &metadataMap)
	}

	return &PaymentResponse{
		ID:             p.ID,
		IdempotencyKey: p.IdempotencyKey,
		BillID:         p.BillID,
		CustomerID:     p.CustomerID,
		Amount:         p.Amount,
		Currency:       p.Currency,
		Provider:       p.Provider,
		Status:         p.Status,
		Description:    p.Description,
		Metadata:       metadataMap,
		PaymentURL:     paymentURL,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}
}

func toRefundResponse(r *Refund) *RefundResponse {
	return &RefundResponse{
		ID:               r.ID,
		PaymentID:        r.PaymentID,
		ProviderRefundID: r.ProviderRefundID,
		Amount:           r.Amount,
		Currency:         r.Currency,
		Reason:           r.Reason,
		Status:           r.Status,
		CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
	}
}

func toTransactionResponse(t *Transaction) *TransactionResponse {
	var rawMap map[string]any
	if t.RawResponse != nil && *t.RawResponse != "" {
		_ = json.Unmarshal([]byte(*t.RawResponse), &rawMap)
	}

	return &TransactionResponse{
		ID:                    t.ID,
		PaymentID:             t.PaymentID,
		ProviderTransactionID: t.ProviderTransactionID,
		Type:                  t.Type,
		Amount:                t.Amount,
		Currency:              t.Currency,
		Status:                t.Status,
		RawResponse:           rawMap,
		ErrorCode:             t.ErrorCode,
		ErrorMessage:          t.ErrorMessage,
		CreatedAt:             t.CreatedAt.Format(time.RFC3339),
	}
}

func toCustomerResponse(c *Customer) *CustomerResponse {
	return &CustomerResponse{
		ID:               c.ID,
		Email:            c.Email,
		Name:             c.Name,
		Phone:            c.Phone,
		StripeCustomerID: c.StripeCustomerID,
		CreatedAt:        c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        c.UpdatedAt.Format(time.RFC3339),
	}
}
