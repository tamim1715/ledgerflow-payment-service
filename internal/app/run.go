package app

import "ledgerflow-payment-service/internal/router"

func (a *Application) Run() error {

	// Register routes
	router.RegisterRoutes(a.Echo, a.Handlers)

	// Register middleware

	// Create HTTP server

	// Start server

	// Wait for shutdown

	return nil
}
