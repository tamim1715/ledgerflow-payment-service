package billing

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	bills := rg.Group("/bills")
	{
		bills.POST("", h.Create)
		bills.GET("", h.List)
		bills.GET("/:id", h.GetByID)
		bills.PUT("/:id", h.Update)
		bills.DELETE("/:id", h.Delete)
		bills.POST("/:id/pay", h.PayBill)
		bills.GET("/:id/payments", h.GetBillPayments)
	}
}
