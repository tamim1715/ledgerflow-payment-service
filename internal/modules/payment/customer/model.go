package customer

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
