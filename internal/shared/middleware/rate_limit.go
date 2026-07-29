package middleware

import (
	"context"
	"fmt"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/redis"

	"github.com/gin-gonic/gin"
)

const (
	RateLimitKey = "rate_limit"
)

type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
}

func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Simple rate limiting using Redis
		// In production, use a more sophisticated rate limiter like redis-rate-limit
		minuteKey := fmt.Sprintf("ratelimit:%s:minute", clientIP)
		hourKey := fmt.Sprintf("ratelimit:%s:hour", clientIP)

		ctx := c.Request.Context()

		// Check minute limit
		minuteCount, _ := redis.Get(ctx, minuteKey)
		if minuteCount != "" && minuteCount == fmt.Sprintf("%d", config.RequestsPerMinute) {
			c.JSON(429, gin.H{"error": "Rate limit exceeded (per minute)"})
			c.Abort()
			return
		}

		// Check hour limit
		hourCount, _ := redis.Get(ctx, hourKey)
		if hourCount != "" && hourCount == fmt.Sprintf("%d", config.RequestsPerHour) {
			c.JSON(429, gin.H{"error": "Rate limit exceeded (per hour)"})
			c.Abort()
			return
		}

		// Increment counters
		if minuteCount == "" {
			redis.Set(ctx, minuteKey, "1", time.Minute)
		} else {
			// This is a simplified approach - in production use atomic increments
		}

		if hourCount == "" {
			redis.Set(ctx, hourKey, "1", time.Hour)
		}

		c.Next()
	}
}

func AuthRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("auth_ratelimit:%s", clientIP)

		ctx := context.Background()
		redisClient := redis.GetClient()

		// Get rate limit config
		maxRequests := config.AppConfig.RateLimit.AuthRequests
		windowMinutes := config.AppConfig.RateLimit.AuthWindowMinutes

		// Check current count
		count, err := redisClient.Get(ctx, key).Int()
		if err != nil && err.Error() != "redis: nil" {
			c.Next()
			return
		}

		// Increment count atomically
		newCount, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		// Set expiration on first request
		if count == 0 {
			redisClient.Expire(ctx, key, time.Duration(windowMinutes)*time.Minute)
		}

		// Check limit
		if newCount > int64(maxRequests) {
			c.JSON(429, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
