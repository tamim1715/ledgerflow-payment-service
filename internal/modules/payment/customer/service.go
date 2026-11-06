package customer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IService interface {
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error)
	GetCustomerByID(ctx context.Context, id string) (*CustomerResponse, error)
	GetCustomerModelByID(ctx context.Context, id string) (*Customer, error)
	ListCustomers(ctx context.Context, page int, limit int) (*CustomerListResponse, error)
}

type service struct {
	repo   IRepository
	logger *zap.Logger
}

func NewService(repo IRepository, logger *zap.Logger) IService {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error) {
	existing, err := s.repo.GetCustomerByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrCustomerAlreadyExists
	}

	custID := fmt.Sprintf("cust_%s", uuid.New().String()[:16])
	now := time.Now().UTC()

	var namePtr *string
	if req.Name != "" {
		namePtr = &req.Name
	}
	var phonePtr *string
	if req.Phone != "" {
		phonePtr = &req.Phone
	}
	var stripeIDPtr *string
	if req.StripeCustomerID != "" {
		stripeIDPtr = &req.StripeCustomerID
	}

	cust := &Customer{
		ID:               custID,
		Email:            req.Email,
		Name:             namePtr,
		Phone:            phonePtr,
		StripeCustomerID: stripeIDPtr,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreateCustomer(ctx, cust); err != nil {
		s.logger.Error("Failed to create customer", zap.Error(err))
		return nil, fmt.Errorf("create customer: %w", err)
	}

	s.logger.Info("Customer created successfully", zap.String("id", cust.ID), zap.String("email", cust.Email))
	return ToCustomerResponse(cust), nil
}

func (s *service) GetCustomerByID(ctx context.Context, id string) (*CustomerResponse, error) {
	cust, err := s.repo.GetCustomerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToCustomerResponse(cust), nil
}

func (s *service) GetCustomerModelByID(ctx context.Context, id string) (*Customer, error) {
	return s.repo.GetCustomerByID(ctx, id)
}

func (s *service) ListCustomers(ctx context.Context, page int, limit int) (*CustomerListResponse, error) {
	customers, total, err := s.repo.ListCustomers(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	responses := make([]CustomerResponse, len(customers))
	for i, c := range customers {
		responses[i] = *ToCustomerResponse(&c)
	}

	return &CustomerListResponse{
		Customers: responses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

func ToCustomerResponse(c *Customer) *CustomerResponse {
	return &CustomerResponse{
		ID:               c.ID,
		Email:            c.Email,
		Name:             c.Name,
		Phone:            c.Phone,
		StripeCustomerID: c.StripeCustomerID,
		CreatedAt:        c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        c.UpdatedAt.Format(time.RFC3339),
	}
}
