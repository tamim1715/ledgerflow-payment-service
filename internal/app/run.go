package app

import (
	"context"
	"errors"
	"fmt"
	"ledgerflow-payment-service/internal/router"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func (a *Application) Run() error {
	// Register routes
	router.RegisterRoutes(a.Engine, a.Handlers)

	// Determine port
	port := a.Config.Server.Port
	if port == 0 {
		port = 8080
	}

	// Create HTTP server with timeouts
	a.HTTPServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      a.Engine,
		ReadTimeout:  a.Config.Server.ReadTimeout,
		WriteTimeout: a.Config.Server.WriteTimeout,
	}

	// Channel to listen for errors from server goroutine
	serverErrors := make(chan error, 1)

	// Start server in background
	go func() {
		a.Logger.Info("Starting HTTP server", zap.String("addr", a.HTTPServer.Addr))
		if err := a.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Listen for OS signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("http server error: %w", err)

	case sig := <-quit:
		a.Logger.Info("Shutdown signal received", zap.String("signal", sig.String()))

		shutdownTimeout := a.Config.Server.ShutdownTimeout
		if shutdownTimeout == 0 {
			shutdownTimeout = 10 * time.Second
		}

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := a.HTTPServer.Shutdown(ctx); err != nil {
			a.Logger.Error("Server forced to shutdown", zap.Error(err))
			_ = a.HTTPServer.Close()
		}

		if a.DB != nil {
			if err := a.DB.Close(); err != nil {
				a.Logger.Error("Failed to close database connection", zap.Error(err))
			} else {
				a.Logger.Info("Database connection closed cleanly")
			}
		}

		a.Logger.Info("Server exiting cleanly")
	}

	return nil
}

