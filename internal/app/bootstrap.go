package app

import (
	"fmt"
	"ledgerflow-payment-service/internal/config"
	"ledgerflow-payment-service/internal/middleware"
	"ledgerflow-payment-service/internal/modules/billing"
	"ledgerflow-payment-service/internal/modules/health"
	"ledgerflow-payment-service/internal/modules/payment"
	"ledgerflow-payment-service/internal/router"
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

	// Connect PostgreSQL
	db, err := database.NewPostgres(cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres: %w", err)
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

	// Instantiate Modules
	healthHandler := health.NewHandler()

	paymentRepo := payment.NewRepository(db)
	paymentSvc := payment.NewService(paymentRepo, log)
	paymentHandler := payment.NewHandler(paymentSvc)

	billingRepo := billing.NewRepository(db)
	billingSvc := billing.NewService(billingRepo, paymentSvc, log)
	billingHandler := billing.NewHandler(billingSvc)

	modules := &router.Modules{
		Health:  healthHandler,
		Billing: billingHandler,
		Payment: paymentHandler,
	}

	app := &Application{
		Config:  cfg,
		Logger:  log,
		DB:      db,
		Engine:  engine,
		Modules: modules,
	}

	return app, nil
}
