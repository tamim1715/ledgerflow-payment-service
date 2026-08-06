package app

import (
	"ledgerflow-payment-service/internal/config"
	"ledgerflow-payment-service/internal/handler"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Application struct {
	Config *config.Config

	Logger *zap.Logger

	DB *sqlx.DB

	Engine *gin.Engine

	HTTPServer *http.Server

	Handlers *handler.Handlers
}

