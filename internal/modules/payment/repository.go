package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrRefundNotFound          = errors.New("refund not found")
	ErrCustomerNotFound        = errors.New("customer not found")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
	ErrCustomerAlreadyExists   = errors.New("customer with this email already exists")
	ErrBillNotFound            = errors.New("bill not found")
	ErrInvalidPaymentRequest   = errors.New("invalid payment request")
)

type BillSummary struct {
	ID          string  `db:"id"`
	CustomerID  string  `db:"customer_id"`
	TotalAmount float64 `db:"total_amount"`
	PaidAmount  float64 `db:"paid_amount"`
	Currency    string  `db:"currency"`
	Status      string  `db:"status"`
}

type IRepository interface {
	CreatePayment(ctx context.Context, payment *Payment, tx *Transaction) error
	GetPaymentByID(ctx context.Context, id string) (*Payment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, key string) (*Payment, error)
	ListPayments(ctx context.Context, customerID string, status string, provider string, page int, limit int) ([]Payment, int, error)
	ListPaymentsByBillID(ctx context.Context, billID string) ([]Payment, error)
	GetBillSummary(ctx context.Context, billID string) (*BillSummary, error)
	UpdatePayment(ctx context.Context, payment *Payment) error
	UpdatePaymentWithTransaction(ctx context.Context, payment *Payment, tx *Transaction) error

	CreateRefund(ctx context.Context, refund *Refund, payment *Payment, tx *Transaction) error
	GetRefundByID(ctx context.Context, id string) (*Refund, error)
	ListRefunds(ctx context.Context, paymentID string, page int, limit int) ([]Refund, int, error)
	GetTotalRefundedAmount(ctx context.Context, paymentID string) (float64, error)

	CreateTransaction(ctx context.Context, tx *Transaction) error
	ListTransactionsByPaymentID(ctx context.Context, paymentID string) ([]Transaction, error)

	RecordWebhookEvent(ctx context.Context, provider string, eventID string, eventType string, payload string) (bool, error)

	CreateCustomer(ctx context.Context, customer *Customer) error
	GetCustomerByID(ctx context.Context, id string) (*Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*Customer, error)
	ListCustomers(ctx context.Context, page int, limit int) ([]Customer, int, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) CreatePayment(ctx context.Context, payment *Payment, tx *Transaction) error {
	dbTx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	queryPayment := `
		INSERT INTO payments (
			id, idempotency_key, bill_id, customer_id, amount, currency, provider,
			status, description, metadata, created_at, updated_at
		) VALUES (
			:id, :idempotency_key, :bill_id, :customer_id, :amount, :currency, :provider,
			:status, :description, :metadata, :created_at, :updated_at
		)
	`
	_, err = dbTx.NamedExecContext(ctx, queryPayment, payment)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	if tx != nil {
		queryTx := `
			INSERT INTO transactions (
				id, payment_id, provider_transaction_id, type, amount, currency,
				status, raw_response, error_code, error_message, created_at
			) VALUES (
				:id, :payment_id, :provider_transaction_id, :type, :amount, :currency,
				:status, :raw_response, :error_code, :error_message, :created_at
			)
		`
		_, err = dbTx.NamedExecContext(ctx, queryTx, tx)
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	// Settle linked bill if payment succeeded immediately
	if payment.BillID != nil && *payment.BillID != "" && payment.Status == PaymentStatusSucceeded {
		queryUpdateBill := `
			UPDATE bills SET
				paid_amount = paid_amount + $1,
				status = CASE WHEN paid_amount + $1 >= total_amount THEN 'PAID' ELSE 'PARTIALLY_PAID' END,
				updated_at = $2
			WHERE id = $3
		`
		_, err = dbTx.ExecContext(ctx, queryUpdateBill, payment.Amount, payment.UpdatedAt, *payment.BillID)
		if err != nil {
			return fmt.Errorf("update linked bill on create payment: %w", err)
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repository) GetPaymentByID(ctx context.Context, id string) (*Payment, error) {
	var payment Payment
	query := `SELECT * FROM payments WHERE id = $1`
	err := r.db.GetContext(ctx, &payment, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return &payment, nil
}

func (r *repository) GetPaymentByIdempotencyKey(ctx context.Context, key string) (*Payment, error) {
	var payment Payment
	query := `SELECT * FROM payments WHERE idempotency_key = $1`
	err := r.db.GetContext(ctx, &payment, query, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment by idempotency key: %w", err)
	}
	return &payment, nil
}

func (r *repository) ListPayments(ctx context.Context, customerID string, status string, provider string, page int, limit int) ([]Payment, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM payments WHERE 1=1`
	args := []any{}
	argIdx := 1

	if customerID != "" {
		countQuery += fmt.Sprintf(" AND customer_id = $%d", argIdx)
		args = append(args, customerID)
		argIdx++
	}
	if status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if provider != "" {
		countQuery += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, provider)
		argIdx++
	}

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count payments: %w", err)
	}

	selectQuery := `SELECT * FROM payments WHERE 1=1`
	selectArgs := []any{}
	sArgIdx := 1

	if customerID != "" {
		selectQuery += fmt.Sprintf(" AND customer_id = $%d", sArgIdx)
		selectArgs = append(selectArgs, customerID)
		sArgIdx++
	}
	if status != "" {
		selectQuery += fmt.Sprintf(" AND status = $%d", sArgIdx)
		selectArgs = append(selectArgs, status)
		sArgIdx++
	}
	if provider != "" {
		selectQuery += fmt.Sprintf(" AND provider = $%d", sArgIdx)
		selectArgs = append(selectArgs, provider)
		sArgIdx++
	}

	selectQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", sArgIdx, sArgIdx+1)
	selectArgs = append(selectArgs, limit, offset)

	var payments []Payment
	err = r.db.SelectContext(ctx, &payments, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list payments: %w", err)
	}

	return payments, total, nil
}

func (r *repository) UpdatePayment(ctx context.Context, payment *Payment) error {
	query := `
		UPDATE payments SET
			status = :status,
			description = :description,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`
	res, err := r.db.NamedExecContext(ctx, query, payment)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *repository) UpdatePaymentWithTransaction(ctx context.Context, payment *Payment, tx *Transaction) error {
	dbTx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	queryPayment := `
		UPDATE payments SET
			status = :status,
			description = :description,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`
	res, err := dbTx.NamedExecContext(ctx, queryPayment, payment)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrPaymentNotFound
	}

	if tx != nil {
		queryTx := `
			INSERT INTO transactions (
				id, payment_id, provider_transaction_id, type, amount, currency,
				status, raw_response, error_code, error_message, created_at
			) VALUES (
				:id, :payment_id, :provider_transaction_id, :type, :amount, :currency,
				:status, :raw_response, :error_code, :error_message, :created_at
			)
		`
		_, err = dbTx.NamedExecContext(ctx, queryTx, tx)
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	// Settle linked bill if payment transitioned to succeeded
	if payment.BillID != nil && *payment.BillID != "" && payment.Status == PaymentStatusSucceeded {
		queryUpdateBill := `
			UPDATE bills SET
				paid_amount = paid_amount + $1,
				status = CASE WHEN paid_amount + $1 >= total_amount THEN 'PAID' ELSE 'PARTIALLY_PAID' END,
				updated_at = $2
			WHERE id = $3
		`
		_, err = dbTx.ExecContext(ctx, queryUpdateBill, payment.Amount, payment.UpdatedAt, *payment.BillID)
		if err != nil {
			return fmt.Errorf("update linked bill on payment success: %w", err)
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repository) CreateRefund(ctx context.Context, refund *Refund, payment *Payment, tx *Transaction) error {
	dbTx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	queryRefund := `
		INSERT INTO refunds (
			id, payment_id, provider_refund_id, amount, currency,
			reason, status, created_at, updated_at
		) VALUES (
			:id, :payment_id, :provider_refund_id, :amount, :currency,
			:reason, :status, :created_at, :updated_at
		)
	`
	_, err = dbTx.NamedExecContext(ctx, queryRefund, refund)
	if err != nil {
		return fmt.Errorf("insert refund: %w", err)
	}

	queryPayment := `
		UPDATE payments SET
			status = :status,
			updated_at = :updated_at
		WHERE id = :id
	`
	res, err := dbTx.NamedExecContext(ctx, queryPayment, payment)
	if err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrPaymentNotFound
	}

	if tx != nil {
		queryTx := `
			INSERT INTO transactions (
				id, payment_id, provider_transaction_id, type, amount, currency,
				status, raw_response, error_code, error_message, created_at
			) VALUES (
				:id, :payment_id, :provider_transaction_id, :type, :amount, :currency,
				:status, :raw_response, :error_code, :error_message, :created_at
			)
		`
		_, err = dbTx.NamedExecContext(ctx, queryTx, tx)
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	// Adjust linked bill if refund succeeded
	if payment.BillID != nil && *payment.BillID != "" && refund.Status == RefundStatusSucceeded {
		queryUpdateBill := `
			UPDATE bills SET
				paid_amount = GREATEST(0, paid_amount - $1),
				status = CASE WHEN paid_amount - $1 <= 0 THEN 'PENDING' ELSE 'PARTIALLY_PAID' END,
				updated_at = $2
			WHERE id = $3
		`
		_, err = dbTx.ExecContext(ctx, queryUpdateBill, refund.Amount, refund.UpdatedAt, *payment.BillID)
		if err != nil {
			return fmt.Errorf("update linked bill on refund: %w", err)
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repository) GetRefundByID(ctx context.Context, id string) (*Refund, error) {
	var refund Refund
	query := `SELECT * FROM refunds WHERE id = $1`
	err := r.db.GetContext(ctx, &refund, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRefundNotFound
		}
		return nil, fmt.Errorf("get refund: %w", err)
	}
	return &refund, nil
}

func (r *repository) ListRefunds(ctx context.Context, paymentID string, page int, limit int) ([]Refund, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM refunds WHERE 1=1`
	args := []any{}
	argIdx := 1

	if paymentID != "" {
		countQuery += fmt.Sprintf(" AND payment_id = $%d", argIdx)
		args = append(args, paymentID)
		argIdx++
	}

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count refunds: %w", err)
	}

	selectQuery := `SELECT * FROM refunds WHERE 1=1`
	selectArgs := []any{}
	sArgIdx := 1

	if paymentID != "" {
		selectQuery += fmt.Sprintf(" AND payment_id = $%d", sArgIdx)
		selectArgs = append(selectArgs, paymentID)
		sArgIdx++
	}

	selectQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", sArgIdx, sArgIdx+1)
	selectArgs = append(selectArgs, limit, offset)

	var refunds []Refund
	err = r.db.SelectContext(ctx, &refunds, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list refunds: %w", err)
	}

	return refunds, total, nil
}

func (r *repository) GetTotalRefundedAmount(ctx context.Context, paymentID string) (float64, error) {
	var total float64
	query := `SELECT COALESCE(SUM(amount), 0) FROM refunds WHERE payment_id = $1 AND status != 'FAILED'`
	err := r.db.GetContext(ctx, &total, query, paymentID)
	if err != nil {
		return 0, fmt.Errorf("get total refunded amount: %w", err)
	}
	return total, nil
}

func (r *repository) CreateTransaction(ctx context.Context, tx *Transaction) error {
	query := `
		INSERT INTO transactions (
			id, payment_id, provider_transaction_id, type, amount, currency,
			status, raw_response, error_code, error_message, created_at
		) VALUES (
			:id, :payment_id, :provider_transaction_id, :type, :amount, :currency,
			:status, :raw_response, :error_code, :error_message, :created_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, tx)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

func (r *repository) ListTransactionsByPaymentID(ctx context.Context, paymentID string) ([]Transaction, error) {
	var transactions []Transaction
	query := `SELECT * FROM transactions WHERE payment_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &transactions, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("list transactions by payment id: %w", err)
	}
	return transactions, nil
}

func (r *repository) CreateCustomer(ctx context.Context, customer *Customer) error {
	query := `
		INSERT INTO customers (
			id, email, name, phone, stripe_customer_id, created_at, updated_at
		) VALUES (
			:id, :email, :name, :phone, :stripe_customer_id, :created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, customer)
	if err != nil {
		return fmt.Errorf("insert customer: %w", err)
	}
	return nil
}

func (r *repository) GetCustomerByID(ctx context.Context, id string) (*Customer, error) {
	var customer Customer
	query := `SELECT * FROM customers WHERE id = $1`
	err := r.db.GetContext(ctx, &customer, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return &customer, nil
}

func (r *repository) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	var customer Customer
	query := `SELECT * FROM customers WHERE email = $1`
	err := r.db.GetContext(ctx, &customer, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer by email: %w", err)
	}
	return &customer, nil
}

func (r *repository) ListCustomers(ctx context.Context, page int, limit int) ([]Customer, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM customers`
	err := r.db.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}

	var customers []Customer
	selectQuery := `SELECT * FROM customers ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err = r.db.SelectContext(ctx, &customers, selectQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}

	return customers, total, nil
}

func (r *repository) ListPaymentsByBillID(ctx context.Context, billID string) ([]Payment, error) {
	var payments []Payment
	query := `SELECT * FROM payments WHERE bill_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &payments, query, billID)
	if err != nil {
		return nil, fmt.Errorf("list payments by bill id: %w", err)
	}
	return payments, nil
}

func (r *repository) GetBillSummary(ctx context.Context, billID string) (*BillSummary, error) {
	var bill BillSummary
	query := `SELECT id, customer_id, total_amount, paid_amount, currency, status FROM bills WHERE id = $1`
	err := r.db.GetContext(ctx, &bill, query, billID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("get bill summary: %w", err)
	}
	return &bill, nil
}

func (r *repository) RecordWebhookEvent(ctx context.Context, provider string, eventID string, eventType string, payload string) (bool, error) {
	query := `
		INSERT INTO webhook_events (id, provider, event_id, event_type, payload, status, processed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, 'PROCESSED', $6, $6)
		ON CONFLICT (provider, event_id) DO NOTHING
	`
	id := uuid.New().String()
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, query, id, provider, eventID, eventType, payload, now)
	if err != nil {
		return false, fmt.Errorf("record webhook event: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}
