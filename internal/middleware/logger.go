package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapLogger returns a gin.HandlerFunc that logs requests using Zap logger.
func ZapLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		if query != "" {
			path = path + "?" + query
		}

		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", userAgent),
		}

		if len(c.Errors) > 0 {
			for _, err := range c.Errors.Errors() {
				log.Error("HTTP request error", zap.String("error", err))
			}
		}

		switch {
		case status >= 500:
			log.Error("HTTP Server Error", fields...)
		case status >= 400:
			log.Warn("HTTP Client Error", fields...)
		default:
			log.Info("HTTP Request", fields...)
		}
	}
}
