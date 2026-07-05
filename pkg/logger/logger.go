package logger

import (
	"fmt"
	"ledgerflow-payment-service/internal/config"

	"go.uber.org/zap"
)

func New(cfg config.LoggerConfig) (*zap.Logger, error) {

	zapConfig := BuildConfig(
		cfg.Development,
		cfg.Level,
	)

	log, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	return log, nil
}
