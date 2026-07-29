package realtime

import (
	"clap/internal/shared/config"
)

func NewRealtimeService() RealtimeService {
	switch config.AppConfig.Environment {
	case "production":
		return NewRedisPubSubService()
	default:
		return NewRedisPubSubService()
	}
}
