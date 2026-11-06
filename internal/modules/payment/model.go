package payment

import (
	"time"
)

type Customer struct {
	ID               string    `db:"id" json:"id"`
	Email            string    `db:"email" json:"email"`
	Name             *string   `db:"name" json:"name,omitempty"`
	Phone            *string   `db:"phone" json:"phone,omitempty"`
	StripeCustomerID *string   `db:"stripe_customer_id" json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "PENDING"
	PaymentStatusProcessing        PaymentStatus = "PROCESSING"
	PaymentStatusSucceeded         PaymentStatus = "SUCCEEDED"
	PaymentStatusFailed            PaymentStatus = "FAILED"
	PaymentStatusCancelled         PaymentStatus = "CANCELLED"
	PaymentStatusRefunded          PaymentStatus = "REFUNDED"
	PaymentStatusPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
)

type PaymentProvider string

const (
	ProviderStripe     PaymentProvider = "STRIPE"
	ProviderSSLCommerz PaymentProvider = "SSLCOMMERZ"
	ProviderPayPal     PaymentProvider = "PAYPAL"
	ProviderBKash      PaymentProvider = "BKASH"
)

type Payment struct {
	ID             string          `db:"id" json:"id"`
	IdempotencyKey string          `db:"idempotency_key" json:"idempotency_key"`
	BillID         *string         `db:"bill_id" json:"bill_id,omitempty"`
	CustomerID     string          `db:"customer_id" json:"customer_id"`
	Amount         float64         `db:"amount" json:"amount"`
	Currency       string          `db:"currency" json:"currency"`
	Provider       PaymentProvider `db:"provider" json:"provider"`
	Status         PaymentStatus   `db:"status" json:"status"`
	Description    *string         `db:"description" json:"description,omitempty"`
	Metadata       *string         `db:"metadata" json:"metadata,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

type TransactionType string

const (
	TxTypeCharge   TransactionType = "CHARGE"
	TxTypeRefund   TransactionType = "REFUND"
	TxTypeReversal TransactionType = "REVERSAL"
)

type TransactionStatus string

const (
	TxStatusSuccess TransactionStatus = "SUCCESS"
	TxStatusFailed  TransactionStatus = "FAILED"
	TxStatusPending TransactionStatus = "PENDING"
)

type Transaction struct {
	ID                    string            `db:"id" json:"id"`
	PaymentID             string            `db:"payment_id" json:"payment_id"`
	ProviderTransactionID *string           `db:"provider_transaction_id" json:"provider_transaction_id,omitempty"`
	Type                  TransactionType   `db:"type" json:"type"`
	Amount                float64           `db:"amount" json:"amount"`
	Currency              string            `db:"currency" json:"currency"`
	Status                TransactionStatus `db:"status" json:"status"`
	RawResponse           *string           `db:"raw_response" json:"raw_response,omitempty"`
	ErrorCode             *string           `db:"error_code" json:"error_code,omitempty"`
	ErrorMessage          *string           `db:"error_message" json:"error_message,omitempty"`
	CreatedAt             time.Time         `db:"created_at" json:"created_at"`
}

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusSucceeded RefundStatus = "SUCCEEDED"
	RefundStatusFailed    RefundStatus = "FAILED"
)

type Refund struct {
	ID               string       `db:"id" json:"id"`
	PaymentID        string       `db:"payment_id" json:"payment_id"`
	ProviderRefundID *string      `db:"provider_refund_id" json:"provider_refund_id,omitempty"`
	Amount           float64      `db:"amount" json:"amount"`
	Currency         string       `db:"currency" json:"currency"`
	Reason           *string      `db:"reason" json:"reason,omitempty"`
	Status           RefundStatus `db:"status" json:"status"`
	CreatedAt        time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time    `db:"updated_at" json:"updated_at"`
}
