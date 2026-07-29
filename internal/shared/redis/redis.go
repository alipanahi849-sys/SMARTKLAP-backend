package redis

import (
	"context"
	"fmt"
	"time"

	"clap/internal/shared/config"
	"clap/internal/shared/logger"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client
var Ctx = context.Background()

func Init() error {
	Client = redis.NewClient(&redis.Options{
		Addr:     config.GetRedisAddr(),
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(Ctx, 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info().Msg("Redis connection established successfully")
	return nil
}

func Close() error {
	if Client == nil {
		return nil
	}

	if err := Client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	logger.Info().Msg("Redis connection closed")
	return nil
}

func GetClient() *redis.Client {
	return Client
}

func HealthCheck() error {
	if Client == nil {
		return fmt.Errorf("redis connection is not initialized")
	}

	ctx, cancel := context.WithTimeout(Ctx, 2*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return Client.Set(ctx, key, value, expiration).Err()
}

func Get(ctx context.Context, key string) (string, error) {
	return Client.Get(ctx, key).Result()
}

func Del(ctx context.Context, keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

func Exists(ctx context.Context, keys ...string) (int64, error) {
	return Client.Exists(ctx, keys...).Result()
}

func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return Client.Expire(ctx, key, expiration).Err()
}

func Publish(ctx context.Context, channel string, message interface{}) error {
	return Client.Publish(ctx, channel, message).Err()
}

func Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return Client.Subscribe(ctx, channels...)
}
