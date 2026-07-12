package router

import (
	"ledgerflow-payment-service/internal/handler"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(
	e *echo.Echo,
	h *handler.Handler,
) {

	e.GET("/health", h.Health.Health)

	e.GET("/ready", h.Health.Ready)

	e.GET("/live", h.Health.Live)
}
