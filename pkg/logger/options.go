package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func BuildConfig(development bool, level string) zap.Config {

	cfg := zap.NewProductionConfig()

	if development {
		cfg = zap.NewDevelopmentConfig()
	}

	cfg.Encoding = "json"

	if development {
		cfg.Encoding = "console"
	}

	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	cfg.DisableStacktrace = true

	switch level {

	case "debug":
		cfg.Level.SetLevel(zap.DebugLevel)

	case "warn":
		cfg.Level.SetLevel(zap.WarnLevel)

	case "error":
		cfg.Level.SetLevel(zap.ErrorLevel)

	default:
		cfg.Level.SetLevel(zap.InfoLevel)

	}

	return cfg
}
