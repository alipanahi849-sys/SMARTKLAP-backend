package middleware

import (
	"clap/internal/shared/config"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	cfg := config.AppConfig.CORS

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		c.Writer.Header().Set("Access-Control-Allow-Origin", getAllowedOrigin(origin, cfg.AllowedOrigins))
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", getAllowedMethods(cfg.AllowedMethods))
		c.Writer.Header().Set("Access-Control-Allow-Headers", getAllowedHeaders(cfg.AllowedHeaders))
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func getAllowedOrigin(origin string, allowedOrigins []string) string {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return origin
		}
	}
	return allowedOrigins[0]
}

func getAllowedMethods(methods []string) string {
	if len(methods) == 0 {
		return "GET, POST, PUT, DELETE, OPTIONS"
	}
	result := ""
	for i, method := range methods {
		if i > 0 {
			result += ", "
		}
		result += method
	}
	return result
}

func getAllowedHeaders(headers []string) string {
	if len(headers) == 0 {
		return "Origin, Content-Type, Authorization"
	}
	result := ""
	for i, header := range headers {
		if i > 0 {
			result += ", "
		}
		result += header
	}
	return result
}
