package middleware

import (
	"context"
	"net/http"
	"time"

	"clap/internal/shared/config"

	"github.com/gin-gonic/gin"
)

func Timeout() gin.HandlerFunc {
	timeout := time.Duration(config.AppConfig.Server.RequestTimeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			c.Next()
			close(finished)
		}()

		select {
		case <-finished:
			return
		case <-ctx.Done():
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
			})
			return
		}
	}
}
