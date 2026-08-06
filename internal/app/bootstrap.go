package app

import (
	"fmt"
	"ledgerflow-payment-service/internal/config"
	"ledgerflow-payment-service/internal/handler"
	"ledgerflow-payment-service/internal/middleware"
	"ledgerflow-payment-service/pkg/database"
	"ledgerflow-payment-service/pkg/logger"

	"github.com/gin-gonic/gin"
)

func NewApplication() (*Application, error) {

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("initialize logger: %w", err)
	}

	// Connect MySQL
	db, err := database.NewMySQL(cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("initialize mysql: %w", err)
	}

	// Set Gin mode according to configuration environment
	if cfg.Logger.Development {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin engine without default middlewares (gin.New best practice)
	engine := gin.New()

	// Register core production middlewares
	engine.Use(middleware.ZapLogger(log))
	engine.Use(middleware.ZapRecovery(log))

	// Create health handler
	health := handler.NewHealthHandler()

	// Create handlers
	h := handler.NewHandler(health)

	app := &Application{
		Config:   cfg,
		Logger:   log,
		DB:       db,
		Engine:   engine,
		Handlers: h,
	}

	return app, nil
}

