package customer

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
// @Router /api/v1/customers [post]
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
// @Router /api/v1/customers/{id} [get]
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
// @Router /api/v1/customers [get]
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
