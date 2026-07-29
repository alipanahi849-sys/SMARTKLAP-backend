package middleware

import (
	"time"

	"clap/internal/shared/logger"

	"github.com/gin-gonic/gin"
)

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		requestID := GetRequestID(c)

		// Propagate request_id into the request context so service-layer code
		// can retrieve it without coupling to Gin.
		ctx := logger.ContextWithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		entry := logger.Info().
			Str("request_id", requestID).
			Str("method", method).
			Str("path", path).
			Str("client_ip", clientIP).
			Str("user_agent", userAgent).
			Int("status", statusCode).
			Dur("latency", latency)

		if len(c.Errors) > 0 {
			entry.Str("error", c.Errors.String())
		}

		entry.Msg("HTTP Request")
	}
}
