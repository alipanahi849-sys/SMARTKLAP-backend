package middleware

import (
	"clap/internal/shared/logger"
	"clap/internal/shared/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Interface("error", err).
					Msg("Panic recovered")

				response.Error(c, fmt.Errorf("panic recovered: %v", err))
				c.Abort()
			}
		}()

		c.Next()
	}
}
