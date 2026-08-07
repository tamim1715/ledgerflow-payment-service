package billing

import (
	"time"
)

type BillStatus string

const (
	BillStatusDraft     BillStatus = "DRAFT"
	BillStatusPending   BillStatus = "PENDING"
	BillStatusPaid      BillStatus = "PAID"
	BillStatusOverdue   BillStatus = "OVERDUE"
	BillStatusCancelled BillStatus = "CANCELLED"
)

type Bill struct {
	ID          string     `db:"id" json:"id"`
	BillNumber  string     `db:"bill_number" json:"bill_number"`
	CustomerID  string     `db:"customer_id" json:"customer_id"`
	Subtotal    float64    `db:"subtotal" json:"subtotal"`
	Tax         float64    `db:"tax" json:"tax"`
	Discount    float64    `db:"discount" json:"discount"`
	TotalAmount float64    `db:"total_amount" json:"total_amount"`
	Currency    string     `db:"currency" json:"currency"`
	Status      BillStatus `db:"status" json:"status"`
	IssueDate   time.Time  `db:"issue_date" json:"issue_date"`
	DueDate     time.Time  `db:"due_date" json:"due_date"`
	Notes       *string    `db:"notes" json:"notes,omitempty"`
	Metadata    *string    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`

	Items []BillItem `db:"-" json:"items,omitempty"`
}

type BillItem struct {
	ID          string    `db:"id" json:"id"`
	BillID      string    `db:"bill_id" json:"bill_id"`
	Description string    `db:"description" json:"description"`
	Quantity    int       `db:"quantity" json:"quantity"`
	UnitPrice   float64   `db:"unit_price" json:"unit_price"`
	Amount      float64   `db:"amount" json:"amount"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
