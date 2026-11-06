package payment

type CreatePaymentRequest struct {
	IdempotencyKey string          `json:"idempotency_key" binding:"required" example:"ik_987654321"`
	BillID         *string         `json:"bill_id,omitempty" example:"bill_123abc456def"`
	CustomerID     string          `json:"customer_id" binding:"required" example:"cust_987654"`
	Amount         float64         `json:"amount" binding:"required,gt=0" example:"99.99"`
	Currency       string          `json:"currency" binding:"required,len=3" example:"USD"`
	Provider       PaymentProvider `json:"provider" binding:"required" example:"STRIPE"`
	Description    string          `json:"description,omitempty" example:"Payment for Invoice #1234"`
	Metadata       map[string]any  `json:"metadata,omitempty" swaggertype:"object"`
}

type CancelPaymentRequest struct {
	Reason string `json:"reason,omitempty" example:"Customer requested cancellation"`
}

type PaymentResponse struct {
	ID             string          `json:"id" example:"pay_123abc456def"`
	IdempotencyKey string          `json:"idempotency_key" example:"ik_987654321"`
	BillID         *string         `json:"bill_id,omitempty" example:"bill_123abc456def"`
	CustomerID     string          `json:"customer_id" example:"cust_987654"`
	Amount         float64         `json:"amount" example:"99.99"`
	Currency       string          `json:"currency" example:"USD"`
	Provider       PaymentProvider `json:"provider" example:"STRIPE"`
	Status         PaymentStatus   `json:"status" example:"SUCCEEDED"`
	Description    *string         `json:"description,omitempty" example:"Payment for Invoice #1234"`
	Metadata       map[string]any  `json:"metadata,omitempty" swaggertype:"object"`
	PaymentURL     *string         `json:"payment_url,omitempty" example:"https://checkout.stripe.com/pay/cs_test_123"`
	CreatedAt      string          `json:"created_at" example:"2026-08-08T12:00:00Z"`
	UpdatedAt      string          `json:"updated_at" example:"2026-08-08T12:00:00Z"`
}

type PaymentDetailResponse struct {
	PaymentResponse
	Transactions []TransactionResponse `json:"transactions"`
	Refunds      []RefundResponse      `json:"refunds"`
}

type PaymentListResponse struct {
	Payments []PaymentResponse `json:"payments"`
	Total    int               `json:"total" example:"10"`
	Page     int               `json:"page" example:"1"`
	Limit    int               `json:"limit" example:"10"`
}

type CreateRefundRequest struct {
	PaymentID string  `json:"payment_id" binding:"required" example:"pay_123abc456def"`
	Amount    float64 `json:"amount" binding:"required,gt=0" example:"25.00"`
	Reason    string  `json:"reason,omitempty" example:"Customer requested cancellation"`
}

type RefundResponse struct {
	ID               string       `json:"id" example:"ref_123abc456def"`
	PaymentID        string       `json:"payment_id" example:"pay_123abc456def"`
	ProviderRefundID *string      `json:"provider_refund_id,omitempty" example:"re_stripe_123"`
	Amount           float64      `json:"amount" example:"25.00"`
	Currency         string       `json:"currency" example:"USD"`
	Reason           *string      `json:"reason,omitempty" example:"Customer requested cancellation"`
	Status           RefundStatus `json:"status" example:"SUCCEEDED"`
	CreatedAt        string       `json:"created_at" example:"2026-08-08T12:00:00Z"`
	UpdatedAt        string       `json:"updated_at" example:"2026-08-08T12:00:00Z"`
}

type RefundListResponse struct {
	Refunds []RefundResponse `json:"refunds"`
	Total   int              `json:"total" example:"1"`
	Page    int              `json:"page" example:"1"`
	Limit   int              `json:"limit" example:"10"`
}

type TransactionResponse struct {
	ID                    string            `json:"id" example:"tx_123abc456def"`
	PaymentID             string            `json:"payment_id" example:"pay_123abc456def"`
	ProviderTransactionID *string           `json:"provider_transaction_id,omitempty" example:"ch_stripe_123"`
	Type                  TransactionType   `json:"type" example:"CHARGE"`
	Amount                float64           `json:"amount" example:"99.99"`
	Currency              string            `json:"currency" example:"USD"`
	Status                TransactionStatus `json:"status" example:"SUCCESS"`
	RawResponse           map[string]any    `json:"raw_response,omitempty" swaggertype:"object"`
	ErrorCode             *string           `json:"error_code,omitempty"`
	ErrorMessage          *string           `json:"error_message,omitempty"`
	CreatedAt             string            `json:"created_at" example:"2026-08-08T12:00:00Z"`
}

type CreateCustomerRequest struct {
	Email            string `json:"email" binding:"required,email" example:"customer@example.com"`
	Name             string `json:"name,omitempty" example:"Alex Morgan"`
	Phone            string `json:"phone,omitempty" example:"+1234567890"`
	StripeCustomerID string `json:"stripe_customer_id,omitempty" example:"cus_123abc"`
}

type CustomerResponse struct {
	ID               string  `json:"id" example:"cust_987654"`
	Email            string  `json:"email" example:"customer@example.com"`
	Name             *string `json:"name,omitempty" example:"Alex Morgan"`
	Phone            *string `json:"phone,omitempty" example:"+1234567890"`
	StripeCustomerID *string `json:"stripe_customer_id,omitempty" example:"cus_123abc"`
	CreatedAt        string  `json:"created_at" example:"2026-08-08T12:00:00Z"`
	UpdatedAt        string  `json:"updated_at" example:"2026-08-08T12:00:00Z"`
}

type CustomerListResponse struct {
	Customers []CustomerResponse `json:"customers"`
	Total     int                `json:"total" example:"2"`
	Page      int                `json:"page" example:"1"`
	Limit     int                `json:"limit" example:"10"`
}
