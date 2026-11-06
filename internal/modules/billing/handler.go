package billing

import (
	"errors"
	"net/http"
	"strconv"

	"ledgerflow-payment-service/internal/modules/payment"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service IService
}

func NewHandler(service IService) *Handler {
	return &Handler{service: service}
}

// Create godoc
// @Summary Create a new bill/invoice
// @Description Creates a new billing invoice with calculated totals and line items
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body CreateBillRequest true "Bill creation payload"
// @Success 201 {object} BillResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.CreateBill(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GetByID godoc
// @Summary Get a bill by ID
// @Description Fetches full bill details including line items by bill ID
// @Tags Billing
// @Produce json
// @Param id path string true "Bill ID"
// @Success 200 {object} BillResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.GetBillByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrBillNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// List godoc
// @Summary List bills
// @Description Retrieves paginated list of bills with optional status and customer filter
// @Tags Billing
// @Produce json
// @Param customer_id query string false "Filter by Customer ID"
// @Param status query string false "Filter by Bill status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} BillListResponse
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills [get]
func (h *Handler) List(c *gin.Context) {
	customerID := c.Query("customer_id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := h.service.ListBills(c.Request.Context(), customerID, status, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// Update godoc
// @Summary Update a bill
// @Description Updates existing bill metadata, status, or item list
// @Tags Billing
// @Accept json
// @Produce json
// @Param id path string true "Bill ID"
// @Param request body UpdateBillRequest true "Bill update payload"
// @Success 200 {object} BillResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.UpdateBill(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrBillNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// Delete godoc
// @Summary Delete a bill
// @Description Deletes a bill by ID
// @Tags Billing
// @Produce json
// @Param id path string true "Bill ID"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteBill(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrBillNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// PayBill godoc
// @Summary Pay a bill / invoice
// @Description Initiates a payment towards a specific invoice and updates its paid balance and status
// @Tags Billing
// @Accept json
// @Produce json
// @Param id path string true "Bill ID"
// @Param request body PayBillRequest true "Bill payment payload"
// @Success 201 {object} payment.PaymentResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills/{id}/pay [post]
func (h *Handler) PayBill(c *gin.Context) {
	id := c.Param("id")

	var req PayBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.PayBill(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrBillNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// GetBillPayments godoc
// @Summary Get payment history for a bill
// @Description Fetches full bill details along with all payment attempts and transactions
// @Tags Billing
// @Produce json
// @Param id path string true "Bill ID"
// @Success 200 {object} BillPaymentDetailResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/bills/{id}/payments [get]
func (h *Handler) GetBillPayments(c *gin.Context) {
	id := c.Param("id")

	res, err := h.service.GetBillPayments(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrBillNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
