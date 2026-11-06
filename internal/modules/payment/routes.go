package payment

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	payments := rg.Group("/payments")
	{
		payments.POST("", h.CreatePayment)
		payments.GET("", h.ListPayments)
		payments.GET("/:id", h.GetPaymentByID)
		payments.POST("/:id/capture", h.CapturePayment)
		payments.POST("/:id/cancel", h.CancelPayment)
		payments.GET("/:id/transactions", h.ListTransactions)
		payments.GET("/:id/refunds", h.ListPaymentRefunds)

		payments.POST("/refunds", h.CreateRefund)
		payments.GET("/refunds/:id", h.GetRefundByID)

		payments.POST("/customers", h.CreateCustomer)
		payments.GET("/customers", h.ListCustomers)
		payments.GET("/customers/:id", h.GetCustomerByID)
	}
}
