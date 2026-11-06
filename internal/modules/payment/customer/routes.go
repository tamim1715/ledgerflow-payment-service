package customer

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	customers := rg.Group("/customers")
	{
		customers.POST("", h.CreateCustomer)
		customers.GET("", h.ListCustomers)
		customers.GET("/:id", h.GetCustomerByID)
	}
}
