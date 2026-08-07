package payment

type CreatePaymentRequest struct {
	IdempotencyKey string          `json:"idempotency_key" binding:"required"`
	CustomerID     string          `json:"customer_id" binding:"required"`
	Amount         float64         `json:"amount" binding:"required,gt=0"`
	Currency       string          `json:"currency" binding:"required,len=3"`
	Provider       PaymentProvider `json:"provider" binding:"required"`
	Description    string          `json:"description,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

type PaymentResponse struct {
	ID             string          `json:"id"`
	IdempotencyKey string          `json:"idempotency_key"`
	CustomerID     string          `json:"customer_id"`
	Amount         float64         `json:"amount"`
	Currency       string          `json:"currency"`
	Provider       PaymentProvider `json:"provider"`
	Status         PaymentStatus   `json:"status"`
	Description    *string         `json:"description,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
	PaymentURL     *string         `json:"payment_url,omitempty"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

type CreateRefundRequest struct {
	PaymentID string  `json:"payment_id" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Reason    string  `json:"reason,omitempty"`
}
