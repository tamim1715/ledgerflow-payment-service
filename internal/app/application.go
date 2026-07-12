package app

import (
	"ledgerflow-payment-service/internal/config"
	"ledgerflow-payment-service/internal/handler"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Application struct {
	Config *config.Config

	Logger *zap.Logger

	DB *sqlx.DB

	Echo *echo.Echo

	HTTPServer *http.Server

	Handlers *handler.Handler
}
