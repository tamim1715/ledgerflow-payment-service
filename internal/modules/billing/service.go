package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"ledgerflow-payment-service/internal/modules/payment"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IService interface {
	CreateBill(ctx context.Context, req CreateBillRequest) (*BillResponse, error)
	GetBillByID(ctx context.Context, id string) (*BillResponse, error)
	ListBills(ctx context.Context, customerID string, status string, page int, limit int) (*BillListResponse, error)
	UpdateBill(ctx context.Context, id string, req UpdateBillRequest) (*BillResponse, error)
	DeleteBill(ctx context.Context, id string) error
	PayBill(ctx context.Context, id string, req PayBillRequest) (*payment.PaymentResponse, error)
	GetBillPayments(ctx context.Context, id string) (*BillPaymentDetailResponse, error)
}

type service struct {
	repo       IRepository
	paymentSvc payment.IService
	logger     *zap.Logger
}

func NewService(repo IRepository, paymentSvc payment.IService, logger *zap.Logger) IService {
	return &service{
		repo:       repo,
		paymentSvc: paymentSvc,
		logger:     logger,
	}
}

func (s *service) CreateBill(ctx context.Context, req CreateBillRequest) (*BillResponse, error) {
	now := time.Now().UTC()
	billID := uuid.New().String()
	billNumber := fmt.Sprintf("INV-%s-%04d", now.Format("20060102"), rand.Intn(10000))

	var subtotal float64
	items := make([]BillItem, 0, len(req.Items))

	for _, itemReq := range req.Items {
		itemAmount := float64(itemReq.Quantity) * itemReq.UnitPrice
		subtotal += itemAmount

		items = append(items, BillItem{
			ID:          uuid.New().String(),
			BillID:      billID,
			Description: itemReq.Description,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
			Amount:      itemAmount,
			CreatedAt:   now,
		})
	}

	totalAmount := subtotal + req.Tax - req.Discount
	if totalAmount < 0 {
		totalAmount = 0
	}

	var metadataStr *string
	if len(req.Metadata) > 0 {
		bytes, err := json.Marshal(req.Metadata)
		if err == nil {
			str := string(bytes)
			metadataStr = &str
		}
	}

	var notesPtr *string
	if req.Notes != "" {
		notesPtr = &req.Notes
	}

	bill := &Bill{
		ID:          billID,
		BillNumber:  billNumber,
		CustomerID:  req.CustomerID,
		Subtotal:    subtotal,
		Tax:         req.Tax,
		Discount:    req.Discount,
		TotalAmount: totalAmount,
		Currency:    req.Currency,
		Status:      BillStatusDraft,
		IssueDate:   now,
		DueDate:     req.DueDate,
		Notes:       notesPtr,
		Metadata:    metadataStr,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, bill, items); err != nil {
		s.logger.Error("failed to create bill", zap.Error(err))
		return nil, fmt.Errorf("create bill: %w", err)
	}

	bill.Items = items
	return toBillResponse(bill), nil
}

func (s *service) GetBillByID(ctx context.Context, id string) (*BillResponse, error) {
	bill, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toBillResponse(bill), nil
}

func (s *service) ListBills(ctx context.Context, customerID string, status string, page int, limit int) (*BillListResponse, error) {
	bills, total, err := s.repo.List(ctx, customerID, status, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list bills: %w", err)
	}

	responses := make([]BillResponse, len(bills))
	for i, b := range bills {
		responses[i] = *toBillResponse(&b)
	}

	return &BillListResponse{
		Bills: responses,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *service) UpdateBill(ctx context.Context, id string, req UpdateBillRequest) (*BillResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	existing.UpdatedAt = now

	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.Tax != nil {
		existing.Tax = *req.Tax
	}
	if req.Discount != nil {
		existing.Discount = *req.Discount
	}
	if req.DueDate != nil {
		existing.DueDate = *req.DueDate
	}
	if req.Notes != nil {
		existing.Notes = req.Notes
	}

	if len(req.Metadata) > 0 {
		bytes, err := json.Marshal(req.Metadata)
		if err == nil {
			str := string(bytes)
			existing.Metadata = &str
		}
	}

	var items []BillItem
	if req.Items != nil {
		var subtotal float64
		items = make([]BillItem, 0, len(req.Items))

		for _, itemReq := range req.Items {
			itemAmount := float64(itemReq.Quantity) * itemReq.UnitPrice
			subtotal += itemAmount

			items = append(items, BillItem{
				ID:          uuid.New().String(),
				BillID:      existing.ID,
				Description: itemReq.Description,
				Quantity:    itemReq.Quantity,
				UnitPrice:   itemReq.UnitPrice,
				Amount:      itemAmount,
				CreatedAt:   now,
			})
		}
		existing.Subtotal = subtotal
	}

	existing.TotalAmount = existing.Subtotal + existing.Tax - existing.Discount
	if existing.TotalAmount < 0 {
		existing.TotalAmount = 0
	}

	if err := s.repo.Update(ctx, existing, items); err != nil {
		s.logger.Error("failed to update bill", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("update bill: %w", err)
	}

	if items != nil {
		existing.Items = items
	}

	return toBillResponse(existing), nil
}

func (s *service) DeleteBill(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (s *service) PayBill(ctx context.Context, id string, req PayBillRequest) (*payment.PaymentResponse, error) {
	bill, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if bill.Status == BillStatusPaid {
		return nil, fmt.Errorf("invoice %s is already fully paid", bill.BillNumber)
	}
	if bill.Status == BillStatusCancelled {
		return nil, fmt.Errorf("cannot pay cancelled invoice %s", bill.BillNumber)
	}

	remainingAmount := bill.TotalAmount - bill.PaidAmount
	chargeAmount := remainingAmount
	if req.Amount != nil && *req.Amount > 0 {
		chargeAmount = *req.Amount
	}

	if chargeAmount > remainingAmount {
		return nil, fmt.Errorf("payment amount %.2f exceeds remaining balance %.2f", chargeAmount, remainingAmount)
	}

	desc := fmt.Sprintf("Payment for invoice %s", bill.BillNumber)
	if req.Description != "" {
		desc = req.Description
	}

	createPaymentReq := payment.CreatePaymentRequest{
		IdempotencyKey: req.IdempotencyKey,
		BillID:         &bill.ID,
		CustomerID:     bill.CustomerID,
		Amount:         chargeAmount,
		Currency:       bill.Currency,
		Provider:       req.Provider,
		Description:    desc,
		Metadata:       req.Metadata,
	}

	paymentRes, err := s.paymentSvc.CreatePayment(ctx, createPaymentReq)
	if err != nil {
		s.logger.Error("Failed to initiate payment for invoice", zap.String("bill_id", bill.ID), zap.Error(err))
		return nil, fmt.Errorf("pay invoice: %w", err)
	}

	s.logger.Info("Payment initiated for invoice",
		zap.String("bill_id", bill.ID),
		zap.String("payment_id", paymentRes.ID),
		zap.Float64("amount", chargeAmount),
		zap.String("provider", string(req.Provider)),
	)

	return paymentRes, nil
}

func (s *service) GetBillPayments(ctx context.Context, id string) (*BillPaymentDetailResponse, error) {
	bill, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	payments, err := s.paymentSvc.ListPaymentsByBillID(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to list payments for bill", zap.String("bill_id", id), zap.Error(err))
		payments = []payment.PaymentResponse{}
	}

	return &BillPaymentDetailResponse{
		BillResponse: *toBillResponse(bill),
		Payments:     payments,
	}, nil
}

func toBillResponse(b *Bill) *BillResponse {
	var metadataMap map[string]any
	if b.Metadata != nil && *b.Metadata != "" {
		_ = json.Unmarshal([]byte(*b.Metadata), &metadataMap)
	}

	items := b.Items
	if items == nil {
		items = []BillItem{}
	}

	return &BillResponse{
		ID:          b.ID,
		BillNumber:  b.BillNumber,
		CustomerID:  b.CustomerID,
		Subtotal:    b.Subtotal,
		Tax:         b.Tax,
		Discount:    b.Discount,
		TotalAmount: b.TotalAmount,
		PaidAmount:  b.PaidAmount,
		Currency:    b.Currency,
		Status:      b.Status,
		IssueDate:   b.IssueDate.Format(time.RFC3339),
		DueDate:     b.DueDate.Format(time.RFC3339),
		Notes:       b.Notes,
		Metadata:    metadataMap,
		Items:       items,
		CreatedAt:   b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   b.UpdatedAt.Format(time.RFC3339),
	}
}
