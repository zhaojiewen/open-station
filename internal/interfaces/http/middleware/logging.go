package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Info("request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		)
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		logger.Error("panic recovered",
			zap.Any("error", err),
			zap.String("path", c.Request.URL.Path),
		)

		c.JSON(500, gin.H{
			"error":   "INT_001",
			"message": "internal server error",
		})
	})
}