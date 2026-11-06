package webhook

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	webhooks := rg.Group("/webhooks")
	{
		webhooks.POST("/:provider", h.HandleWebhook)
		webhooks.GET("/events", h.ListEvents)
	}
}
