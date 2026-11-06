package router

import (
	"ledgerflow-payment-service/internal/modules/billing"
	"ledgerflow-payment-service/internal/modules/health"
	"ledgerflow-payment-service/internal/modules/payment"
	"ledgerflow-payment-service/internal/modules/payment/customer"
	"ledgerflow-payment-service/internal/modules/payment/webhook"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Modules struct {
	Health   *health.Handler
	Billing  *billing.Handler
	Payment  *payment.Handler
	Customer *customer.Handler
	Webhook  *webhook.Handler
}

func RegisterRoutes(
	r *gin.Engine,
	modules *Modules,
) {
	// 1. Health Module Routes
	health.RegisterRoutes(r, modules.Health)

	// 2. API v1 Router Group
	v1 := r.Group("/api/v1")
	{
		// Billing Module Routes
		billing.RegisterRoutes(v1, modules.Billing)
		// Payment Module Routes
		payment.RegisterRoutes(v1, modules.Payment)
		// Customer Submodule Routes
		if modules.Customer != nil {
			customer.RegisterRoutes(v1, modules.Customer)
		}
		// Webhook Module Routes
		webhook.RegisterRoutes(v1, modules.Webhook)
	}

	// 3. Serve Static Swagger Documentation for Module
	r.Static("/docs/swagger", "./docs/swagger")

	// Global Swagger UI endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/all/swagger.json")))

	// Module specific Swagger UI endpoints
	r.GET("/swagger-ui/billing/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/billing/swagger.json")))
	r.GET("/swagger-ui/health/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/health/swagger.json")))
	r.GET("/swagger-ui/payment/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/payment/swagger.json")))
	r.GET("/swagger-ui/webhook/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/webhook/swagger.json")))
}
