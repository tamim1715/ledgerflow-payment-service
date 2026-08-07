package app

import (
	"net/http"

	"ledgerflow-payment-service/internal/config"
	"ledgerflow-payment-service/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Application struct {
	Config     *config.Config
	Logger     *zap.Logger
	DB         *sqlx.DB
	Engine     *gin.Engine
	HTTPServer *http.Server
	Modules    *router.Modules
}
