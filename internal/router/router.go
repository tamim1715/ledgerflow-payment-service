package router

import (
	"ledgerflow-payment-service/internal/modules/billing"
	"ledgerflow-payment-service/internal/modules/health"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Modules struct {
	Health  *health.Handler
	Billing *billing.Handler
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
	}

	// 3. Serve Static Swagger Documentation for Modules
	r.Static("/docs/swagger", "./docs/swagger")

	// Global Swagger UI endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/all/swagger.json")))

	// Module specific Swagger UI endpoints
	r.GET("/swagger-ui/billing/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/billing/swagger.json")))
	r.GET("/swagger-ui/health/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger/health/swagger.json")))
}
