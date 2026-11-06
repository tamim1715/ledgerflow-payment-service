package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

var (
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerAlreadyExists = errors.New("customer with this email already exists")
)

type IRepository interface {
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
