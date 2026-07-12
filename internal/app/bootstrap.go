package app

import (
	"fmt"
	"ledgerflow-payment-service/internal/config"
	"ledgerflow-payment-service/internal/handler"
	"ledgerflow-payment-service/pkg/database"
	"ledgerflow-payment-service/pkg/logger"

	"github.com/labstack/echo/v4"
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

	// Create Echo
	e := echo.New()

	// Create health handler
	health := handler.NewHealthHandler()

	// Create handlers
	h := handler.NewHandler(health)

	app := &Application{
		Config: cfg,
		Logger: log,
		DB:     db,
		Echo:   e,
		Handlers: h,
	}

	return app, nil
}
