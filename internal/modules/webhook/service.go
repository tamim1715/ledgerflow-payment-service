package webhook

import (
	"context"
	"fmt"
	"strings"

	"ledgerflow-payment-service/internal/modules/payment"

	"go.uber.org/zap"
)

type IService interface {
	ProcessWebhook(ctx context.Context, providerStr string, headers map[string]string, body []byte) (*WebhookProcessResult, error)
	ListEvents(ctx context.Context, provider string, page int, limit int) ([]WebhookEvent, int, error)
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

func (s *service) ProcessWebhook(ctx context.Context, providerStr string, headers map[string]string, body []byte) (*WebhookProcessResult, error) {
	provider := payment.PaymentProvider(strings.ToUpper(providerStr))

	// 1. Resolve parser for provider
	parser, err := NewWebhookParser(provider)
	if err != nil {
		s.logger.Error("Unsupported webhook provider", zap.String("provider", providerStr), zap.Error(err))
		return nil, fmt.Errorf("resolve parser: %w", err)
	}

	// 2. Parse provider payload into NormalizedWebhookEvent
	normalized, err := parser.Parse(headers, body)
	if err != nil {
		s.logger.Warn("Failed to parse webhook payload",
			zap.String("provider", providerStr),
			zap.Error(err),
		)
		return nil, fmt.Errorf("parse webhook payload: %w", err)
	}

	// 3. Deduplication: Idempotency check via webhook_events table
	isNew, err := s.repo.RecordEvent(ctx, string(provider), normalized.EventID, normalized.EventType, string(body))
	if err != nil {
		s.logger.Error("Failed to record webhook event in repository", zap.Error(err))
		return nil, fmt.Errorf("record event: %w", err)
	}

	if !isNew {
		s.logger.Info("Duplicate webhook event ignored (idempotent)",
			zap.String("provider", string(provider)),
			zap.String("event_id", normalized.EventID),
			zap.String("event_type", normalized.EventType),
		)
		return &WebhookProcessResult{
			EventID:   normalized.EventID,
			Provider:  string(provider),
			EventType: normalized.EventType,
			Status:    "IGNORED_DUPLICATE",
			Message:   "Event previously processed",
		}, nil
	}

	// 4. If no target payment ID identified, just acknowledge reception
	if normalized.PaymentID == "" {
		s.logger.Info("Webhook event received and recorded with no target payment",
			zap.String("provider", string(provider)),
			zap.String("event_type", normalized.EventType),
		)
		return &WebhookProcessResult{
			EventID:   normalized.EventID,
			Provider:  string(provider),
			EventType: normalized.EventType,
			Status:    "LOGGED",
			Message:   "Event logged without target payment",
		}, nil
	}

	// 5. Forward normalized event to Payment module
	paymentEvent := payment.WebhookPaymentEvent{
		PaymentID:    normalized.PaymentID,
		Provider:     normalized.Provider,
		ProviderTxID: normalized.ProviderTxID,
		Status:       normalized.Status,
		Amount:       normalized.Amount,
		Currency:     normalized.Currency,
		ErrorCode:    normalized.ErrorCode,
		ErrorMessage: normalized.ErrorMessage,
		RawResponse:  normalized.RawPayload,
	}

	paymentRes, err := s.paymentSvc.ProcessWebhookEvent(ctx, paymentEvent)
	if err != nil {
		s.logger.Error("Failed to apply webhook event to payment module",
			zap.String("payment_id", normalized.PaymentID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("apply webhook event: %w", err)
	}

	s.logger.Info("Webhook processed successfully and payment updated",
		zap.String("provider", string(provider)),
		zap.String("event_id", normalized.EventID),
		zap.String("payment_id", normalized.PaymentID),
		zap.String("payment_status", string(paymentRes.Status)),
	)

	return &WebhookProcessResult{
		EventID:   normalized.EventID,
		Provider:  string(provider),
		EventType: normalized.EventType,
		Status:    "PROCESSED",
		Payment:   paymentRes,
		Message:   "Payment updated successfully from webhook",
	}, nil
}

func (s *service) ListEvents(ctx context.Context, provider string, page int, limit int) ([]WebhookEvent, int, error) {
	return s.repo.ListEvents(ctx, provider, page, limit)
}
