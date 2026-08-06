package router

import (
	"ledgerflow-payment-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	h *handler.Handlers,
) {
	// Health routes
	r.GET("/health", h.Health.Health)
	r.GET("/ready", h.Health.Ready)
	r.GET("/live", h.Health.Live)
}
