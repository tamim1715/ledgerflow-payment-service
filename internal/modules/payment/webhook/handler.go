package webhook

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

// HandleWebhook godoc
// @Summary Ingest payment provider webhook event
// @Description Receives asynchronous webhooks from payment gateways (Stripe, SSLCommerz, bKash, PayPal), verifies event deduplication, normalizes payloads, and updates payment state and linked invoice
// @Tags Webhooks
// @Accept json,x-www-form-urlencoded
// @Produce json
// @Param provider path string true "Payment Gateway Provider (STRIPE, SSLCOMMERZ, BKASH, PAYPAL)"
// @Success 200 {object} WebhookProcessResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/webhooks/{provider} [post]
func (h *Handler) HandleWebhook(c *gin.Context) {
	provider := c.Param("provider")

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read webhook body"})
		return
	}

	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	res, err := h.service.ProcessWebhook(c.Request.Context(), provider, headers, body)
	if err != nil {
		if errors.Is(err, ErrInvalidWebhookData) || errors.Is(err, ErrUnsupportedProvider) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// ListEvents godoc
// @Summary List received webhook events
// @Description Returns paginated audit log of ingested webhook events
// @Tags Webhooks
// @Produce json
// @Param provider query string false "Filter by provider"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} map[string]any
// @Failure 500 {object} map[string]string
// @Router /api/v1/webhooks/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	provider := c.Query("provider")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	events, total, err := h.service.ListEvents(c.Request.Context(), provider, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}
