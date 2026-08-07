package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// Health check endpoint
// @Summary System health check
// @Description Returns server liveness status
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

// Ready readiness check endpoint
// @Summary System readiness check
// @Description Returns readiness status for handling traffic
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /ready [get]
func (h *Handler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "READY",
	})
}

// Live liveness check endpoint
// @Summary System liveness check
// @Description Returns liveness probe status
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /live [get]
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ALIVE",
	})
}
