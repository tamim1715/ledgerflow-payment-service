package customer

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
