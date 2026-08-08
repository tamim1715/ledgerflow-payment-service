package payment

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service IService
}

func NewHandler(service IService) *Handler {
	return &Handler{service: service}
}

// CreatePayment godoc
// @Summary Create and initiate a payment
// @Description Creates a new payment, executes idempotency check, initiates gateway transaction, and records ledger entries
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body CreatePaymentRequest true "Payment initiation payload"
// @Success 201 {object} PaymentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string "Customer not found"
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments [post]
func (h *Handler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.CreatePayment(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GetPaymentByID godoc
// @Summary Get payment details
// @Description Fetches comprehensive payment information including transactions and refunds
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} PaymentDetailResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/{id} [get]
func (h *Handler) GetPaymentByID(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.GetPaymentByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListPayments godoc
// @Summary List payments
// @Description Retrieves paginated list of payments with optional customer, status, and provider filters
// @Tags Payments
// @Produce json
// @Param customer_id query string false "Filter by Customer ID"
// @Param status query string false "Filter by Payment status"
// @Param provider query string false "Filter by Provider (STRIPE, SSLCOMMERZ, PAYPAL, BKASH)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} PaymentListResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments [get]
func (h *Handler) ListPayments(c *gin.Context) {
	customerID := c.Query("customer_id")
	status := c.Query("status")
	provider := c.Query("provider")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := h.service.ListPayments(c.Request.Context(), customerID, status, provider, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// CapturePayment godoc
// @Summary Capture or confirm a payment
// @Description Captures an authorized or processing payment and transitions status to SUCCEEDED
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} PaymentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/{id}/capture [post]
func (h *Handler) CapturePayment(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.CapturePayment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		if errors.Is(err, ErrInvalidPaymentStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// CancelPayment godoc
// @Summary Cancel a payment
// @Description Cancels a pending or processing payment and records reversal transaction
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param request body CancelPaymentRequest false "Cancellation reason"
// @Success 200 {object} PaymentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/{id}/cancel [post]
func (h *Handler) CancelPayment(c *gin.Context) {
	id := c.Param("id")

	var req CancelPaymentRequest
	_ = c.ShouldBindJSON(&req)

	res, err := h.service.CancelPayment(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		if errors.Is(err, ErrInvalidPaymentStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListTransactions godoc
// @Summary List transactions for a payment
// @Description Fetches audit ledger transaction history for a specific payment
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {array} TransactionResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/{id}/transactions [get]
func (h *Handler) ListTransactions(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.ListTransactions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListPaymentRefunds godoc
// @Summary List refunds for a specific payment
// @Description Fetches all refunds executed against a payment ID
// @Tags Refunds
// @Produce json
// @Param id path string true "Payment ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} RefundListResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/{id}/refunds [get]
func (h *Handler) ListPaymentRefunds(c *gin.Context) {
	id := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := h.service.ListRefunds(c.Request.Context(), id, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// CreateRefund godoc
// @Summary Create a refund
// @Description Issues a full or partial refund for an existing settled payment
// @Tags Refunds
// @Accept json
// @Produce json
// @Param request body CreateRefundRequest true "Refund creation payload"
// @Success 201 {object} RefundResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/refunds [post]
func (h *Handler) CreateRefund(c *gin.Context) {
	var req CreateRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.ProcessRefund(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		if errors.Is(err, ErrInvalidPaymentStatus) || errors.Is(err, ErrRefundAmountExceeded) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GetRefundByID godoc
// @Summary Get refund by ID
// @Description Fetches refund details by refund ID
// @Tags Refunds
// @Produce json
// @Param id path string true "Refund ID"
// @Success 200 {object} RefundResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/refunds/{id} [get]
func (h *Handler) GetRefundByID(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.GetRefundByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRefundNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "refund not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// CreateCustomer godoc
// @Summary Create a customer
// @Description Registers a new customer profile for payment processing
// @Tags Customers
// @Accept json
// @Produce json
// @Param request body CreateCustomerRequest true "Customer creation payload"
// @Success 201 {object} CustomerResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string "Customer with email already exists"
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/customers [post]
func (h *Handler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.CreateCustomer(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCustomerAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "customer with this email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GetCustomerByID godoc
// @Summary Get customer by ID
// @Description Fetches customer profile by customer ID
// @Tags Customers
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} CustomerResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/customers/{id} [get]
func (h *Handler) GetCustomerByID(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.GetCustomerByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListCustomers godoc
// @Summary List customers
// @Description Retrieves paginated list of registered customers
// @Tags Customers
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} CustomerListResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/payments/customers [get]
func (h *Handler) ListCustomers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := h.service.ListCustomers(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
