package health

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, h *Handler) {
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)
	r.GET("/live", h.Live)
}
