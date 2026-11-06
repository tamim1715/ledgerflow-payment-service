package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

var ErrBillNotFound = errors.New("bill not found")

type IRepository interface {
	Create(ctx context.Context, bill *Bill, items []BillItem) error
	GetByID(ctx context.Context, id string) (*Bill, error)
	List(ctx context.Context, customerID string, status string, page int, limit int) ([]Bill, int, error)
	Update(ctx context.Context, bill *Bill, items []BillItem) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, bill *Bill, items []BillItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	queryBill := `
		INSERT INTO bills (
			id, bill_number, customer_id, subtotal, tax, discount, total_amount, paid_amount,
			currency, status, issue_date, due_date, notes, metadata, created_at, updated_at
		) VALUES (
			:id, :bill_number, :customer_id, :subtotal, :tax, :discount, :total_amount, :paid_amount,
			:currency, :status, :issue_date, :due_date, :notes, :metadata, :created_at, :updated_at
		)
	`
	_, err = tx.NamedExecContext(ctx, queryBill, bill)
	if err != nil {
		return fmt.Errorf("insert bill: %w", err)
	}

	queryItem := `
		INSERT INTO bill_items (id, bill_id, description, quantity, unit_price, amount, created_at)
		VALUES (:id, :bill_id, :description, :quantity, :unit_price, :amount, :created_at)
	`
	for _, item := range items {
		_, err = tx.NamedExecContext(ctx, queryItem, item)
		if err != nil {
			return fmt.Errorf("insert bill item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*Bill, error) {
	var bill Bill
	queryBill := `SELECT * FROM bills WHERE id = $1`
	err := r.db.GetContext(ctx, &bill, queryBill, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("get bill: %w", err)
	}

	var items []BillItem
	queryItems := `SELECT * FROM bill_items WHERE bill_id = $1 ORDER BY created_at ASC`
	err = r.db.SelectContext(ctx, &items, queryItems, id)
	if err != nil {
		return nil, fmt.Errorf("get bill items: %w", err)
	}

	bill.Items = items
	return &bill, nil
}

func (r *repository) List(ctx context.Context, customerID string, status string, page int, limit int) ([]Bill, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM bills WHERE 1=1`
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

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count bills: %w", err)
	}

	selectQuery := `SELECT * FROM bills WHERE 1=1`
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

	selectQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", sArgIdx, sArgIdx+1)
	selectArgs = append(selectArgs, limit, offset)

	var bills []Bill
	err = r.db.SelectContext(ctx, &bills, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list bills: %w", err)
	}

	return bills, total, nil
}

func (r *repository) Update(ctx context.Context, bill *Bill, items []BillItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	queryBill := `
		UPDATE bills SET 
			subtotal = :subtotal,
			tax = :tax,
			discount = :discount,
			total_amount = :total_amount,
			paid_amount = :paid_amount,
			status = :status,
			due_date = :due_date,
			notes = :notes,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`
	res, err := tx.NamedExecContext(ctx, queryBill, bill)
	if err != nil {
		return fmt.Errorf("update bill: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrBillNotFound
	}

	if items != nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM bill_items WHERE bill_id = $1`, bill.ID)
		if err != nil {
			return fmt.Errorf("delete old bill items: %w", err)
		}

		queryItem := `
			INSERT INTO bill_items (id, bill_id, description, quantity, unit_price, amount, created_at)
			VALUES (:id, :bill_id, :description, :quantity, :unit_price, :amount, :created_at)
		`
		for _, item := range items {
			_, err = tx.NamedExecContext(ctx, queryItem, item)
			if err != nil {
				return fmt.Errorf("insert updated bill item: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM bills WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete bill: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrBillNotFound
	}
	return nil
}
