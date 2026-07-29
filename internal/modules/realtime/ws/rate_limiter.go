package ws

import (
	"context"
	"fmt"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/logger"
	"clap/internal/shared/redis"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// WSConnectionLimit is the fallback maximum number of WebSocket upgrade
	// attempts allowed per IP per minute when config is unavailable.
	WSConnectionLimit = 20
	// WSSubscriptionLimit is the fallback maximum number of channel subscriptions
	// allowed per IP per minute when config is unavailable.
	WSSubscriptionLimit = 100
)

// slidingWindowScript atomically increments a counter and ensures a TTL is set
// on first use, in a single round-trip. This removes the INCR/EXPIRE race that
// could otherwise leave a key without an expiry (permanent lockout).
//
// KEYS[1] = counter key, ARGV[1] = window seconds.
// Returns the post-increment counter value.
var slidingWindowScript = goredis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func connectionLimit() int64 {
	if config.AppConfig != nil && config.AppConfig.Realtime.WSConnectionLimitPerMin > 0 {
		return int64(config.AppConfig.Realtime.WSConnectionLimitPerMin)
	}
	return WSConnectionLimit
}

func subscriptionLimit() int64 {
	if config.AppConfig != nil && config.AppConfig.Realtime.WSSubscriptionLimitPerMin > 0 {
		return int64(config.AppConfig.Realtime.WSSubscriptionLimitPerMin)
	}
	return WSSubscriptionLimit
}

// ConnectionRateLimit returns a Gin middleware that limits WebSocket connection
// attempts per IP per minute using Redis.
// Falls through (does not block) if Redis is unavailable to avoid false positives.
func ConnectionRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("ws_conn_rl:%s", ip)
		ctx := c.Request.Context()

		allowed, count := checkAndIncrement(ctx, key, connectionLimit(), time.Minute)
		if !allowed {
			logger.Warn().
				Str("client_ip", ip).
				Int64("count", count).
				Msg("websocket connection rate limit exceeded")
			c.JSON(429, gin.H{
				"error": "WebSocket connection rate limit exceeded. Try again shortly.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// SubscriptionRateLimiter checks the per-IP subscription rate limit.
// It is called from client.handleInbound on every subscribe message.
// Returns false if the limit is exceeded. Fails open when Redis is unavailable.
func SubscriptionRateLimiter(ctx context.Context, clientIP string) bool {
	if clientIP == "" {
		// No IP context (e.g. unit tests / non-HTTP transport) — do not block.
		return true
	}
	key := fmt.Sprintf("ws_sub_rl:%s", clientIP)
	allowed, count := checkAndIncrement(ctx, key, subscriptionLimit(), time.Minute)
	if !allowed {
		logger.Warn().
			Str("client_ip", clientIP).
			Int64("count", count).
			Msg("websocket subscription rate limit exceeded")
	}
	return allowed
}

// checkAndIncrement atomically increments a counter and sets the TTL on first
// use using a Lua script (single round-trip, no INCR/EXPIRE race).
// Returns (allowed, currentCount). Falls through (returns true) if Redis is
// unavailable to avoid false-positive lockouts.
func checkAndIncrement(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64) {
	rdb := redis.GetClient()
	if rdb == nil {
		return true, 0
	}

	result, err := slidingWindowScript.Run(ctx, rdb, []string{key}, int(window.Seconds())).Int64()
	if err != nil {
		// Redis unavailable — allow the request to avoid false positives.
		return true, 0
	}

	return result <= limit, result
}
