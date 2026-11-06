package customer

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

type mockCustomerRepo struct {
	customers map[string]*Customer
	byEmail   map[string]*Customer
}

func newMockCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{
		customers: make(map[string]*Customer),
		byEmail:   make(map[string]*Customer),
	}
}

func (m *mockCustomerRepo) CreateCustomer(ctx context.Context, c *Customer) error {
	if _, exists := m.byEmail[c.Email]; exists {
		return ErrCustomerAlreadyExists
	}
	m.customers[c.ID] = c
	m.byEmail[c.Email] = c
	return nil
}

func (m *mockCustomerRepo) GetCustomerByID(ctx context.Context, id string) (*Customer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	return c, nil
}

func (m *mockCustomerRepo) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	c, ok := m.byEmail[email]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	return c, nil
}

func (m *mockCustomerRepo) ListCustomers(ctx context.Context, page int, limit int) ([]Customer, int, error) {
	var list []Customer
	for _, c := range m.customers {
		list = append(list, *c)
	}
	return list, len(list), nil
}

func TestCustomerService_CreateAndGet(t *testing.T) {
	repo := newMockCustomerRepo()
	logger := zap.NewNop()
	svc := NewService(repo, logger)
	ctx := context.Background()

	// 1. Create Customer
	req := CreateCustomerRequest{
		Email:            "alice@example.com",
		Name:             "Alice Smith",
		Phone:            "+1234567890",
		StripeCustomerID: "cus_alice123",
	}

	created, err := svc.CreateCustomer(ctx, req)
	if err != nil {
		t.Fatalf("expected create customer to succeed, got %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected non-empty customer ID")
	}
	if created.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, created.Email)
	}

	// 2. Duplicate Email should fail
	_, err = svc.CreateCustomer(ctx, req)
	if err != ErrCustomerAlreadyExists {
		t.Errorf("expected ErrCustomerAlreadyExists, got %v", err)
	}

	// 3. Get Customer by ID
	fetched, err := svc.GetCustomerByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected get customer to succeed, got %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}

	// 4. Get Customer Model by ID
	model, err := svc.GetCustomerModelByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected get customer model to succeed, got %v", err)
	}
	if model.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, model.ID)
	}

	// 5. List Customers
	list, err := svc.ListCustomers(ctx, 1, 10)
	if err != nil {
		t.Fatalf("expected list customers to succeed, got %v", err)
	}
	if list.Total != 1 {
		t.Errorf("expected 1 customer, got %d", list.Total)
	}

	// 6. Non-existing customer
	_, err = svc.GetCustomerByID(ctx, "non_existent")
	if err != ErrCustomerNotFound {
		t.Errorf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestToCustomerResponse(t *testing.T) {
	now := time.Now()
	name := "Bob"
	c := &Customer{
		ID:        "c_1",
		Email:     "bob@example.com",
		Name:      &name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	res := ToCustomerResponse(c)
	if res.ID != "c_1" || res.Email != "bob@example.com" || *res.Name != "Bob" {
		t.Errorf("unexpected customer response: %+v", res)
	}
}
