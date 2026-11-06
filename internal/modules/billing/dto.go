package billing

import (
	"time"
	"ledgerflow-payment-service/internal/modules/payment"
)

type CreateBillItemRequest struct {
	Description string  `json:"description" binding:"required" example:"Cloud Hosting & Infrastructure"`
	Quantity    int     `json:"quantity" binding:"required,gt=0" example:"2"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gte=0" example:"150.00"`
}

type CreateBillRequest struct {
	CustomerID string                  `json:"customer_id" binding:"required" example:"cust_987654"`
	Currency   string                  `json:"currency" binding:"required,len=3" example:"USD"`
	Tax        float64                 `json:"tax" binding:"gte=0" example:"15.00"`
	Discount   float64                 `json:"discount" binding:"gte=0" example:"10.00"`
	DueDate    time.Time               `json:"due_date" binding:"required" example:"2026-12-31T23:59:59Z"`
	Notes      string                  `json:"notes,omitempty" example:"Monthly enterprise subscription invoice"`
	Metadata   map[string]any          `json:"metadata,omitempty" swaggertype:"object"`
	Items      []CreateBillItemRequest `json:"items" binding:"required,dive,required"`
}

type UpdateBillRequest struct {
	Status   *BillStatus             `json:"status,omitempty" example:"PAID"`
	Tax      *float64                `json:"tax,omitempty" binding:"omitempty,gte=0" example:"20.00"`
	Discount *float64                `json:"discount,omitempty" binding:"omitempty,gte=0" example:"5.00"`
	DueDate  *time.Time              `json:"due_date,omitempty" example:"2026-12-31T23:59:59Z"`
	Notes    *string                 `json:"notes,omitempty" example:"Payment received and verified"`
	Metadata map[string]any          `json:"metadata,omitempty" swaggertype:"object"`
	Items    []CreateBillItemRequest `json:"items,omitempty" binding:"omitempty,dive"`
}

type PayBillRequest struct {
	IdempotencyKey string                  `json:"idempotency_key" binding:"required" example:"ik_pay_bill_123"`
	Amount         *float64                `json:"amount,omitempty" binding:"omitempty,gt=0" example:"150.00"`
	Provider       payment.PaymentProvider `json:"provider" binding:"required" example:"STRIPE"`
	Description    string                  `json:"description,omitempty" example:"Payment for Invoice INV-2026-0001"`
	Metadata       map[string]any          `json:"metadata,omitempty" swaggertype:"object"`
}

type BillResponse struct {
	ID          string         `json:"id" example:"bill_123abc456def"`
	BillNumber  string         `json:"bill_number" example:"INV-2026-0001"`
	CustomerID  string         `json:"customer_id" example:"cust_987654"`
	Subtotal    float64        `json:"subtotal" example:"300.00"`
	Tax         float64        `json:"tax" example:"15.00"`
	Discount    float64        `json:"discount" example:"10.00"`
	TotalAmount float64        `json:"total_amount" example:"305.00"`
	PaidAmount  float64        `json:"paid_amount" example:"305.00"`
	Currency    string         `json:"currency" example:"USD"`
	Status      BillStatus     `json:"status" example:"DRAFT"`
	IssueDate   string         `json:"issue_date" example:"2026-08-07T15:00:00Z"`
	DueDate     string         `json:"due_date" example:"2026-12-31T23:59:59Z"`
	Notes       *string        `json:"notes,omitempty" example:"Monthly enterprise subscription invoice"`
	Metadata    map[string]any `json:"metadata,omitempty" swaggertype:"object"`
	Items       []BillItem     `json:"items"`
	CreatedAt   string         `json:"created_at" example:"2026-08-07T15:00:00Z"`
	UpdatedAt   string         `json:"updated_at" example:"2026-08-07T15:00:00Z"`
}

type BillPaymentDetailResponse struct {
	BillResponse
	Payments []payment.PaymentResponse `json:"payments"`
}

type BillListResponse struct {
	Bills []BillResponse `json:"bills"`
	Total int            `json:"total" example:"1"`
	Page  int            `json:"page" example:"1"`
	Limit int            `json:"limit" example:"10"`
}
