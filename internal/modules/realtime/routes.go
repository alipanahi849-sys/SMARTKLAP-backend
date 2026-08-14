package realtime

import (
	realtimehandler "clap/internal/modules/realtime/handler"
	"clap/internal/modules/realtime/gateway"
	"clap/internal/modules/realtime/metrics"
	realtimeservice "clap/internal/modules/realtime/service"
	"clap/internal/modules/realtime/ws"
	"clap/internal/shared/middleware"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the read-only time-sync endpoints.
// Call RegisterRoutesWithWS to also mount the WebSocket endpoint and recovery
// handler — this requires a running Hub and recovery service.
func RegisterRoutes(r *gin.RouterGroup) {
	svc := realtimeservice.NewTimeSyncService(realtimeservice.SystemClock())
	h := realtimehandler.NewTimeSyncHandler(svc)

	rt := r.Group("/realtime")
	{
		rt.GET("/time", h.GetServerTime)
		rt.POST("/time-sync", h.Sync)
	}
}

// WSConfig holds the runtime dependencies needed to mount WebSocket endpoints.
type WSConfig struct {
	CM           *ws.ConnectionManager
	Gateway      *gateway.WebSocketRealtimeGateway
	RecoverySvc  realtimeservice.ReconnectionRecoveryService
	Metrics      *metrics.Metrics
	RetentionSvc realtimeservice.DataRetentionService
	HeartbeatSvc realtimeservice.HeartbeatCleanupService
}

// RegisterRoutesWithWS mounts all realtime endpoints including:
//
//	GET  /realtime/time                          — server time (no auth)
//	POST /realtime/time-sync                     — drift correction (no auth)
//	GET  /realtime/ws                            — WebSocket upgrade (JWT required)
//	GET  /realtime/session/:matchId              — reconnection recovery (JWT required)
//	POST /realtime/admin/test-emit               — push a test WS event (admin only)
//	POST /realtime/admin/cleanup/*               — data retention (admin only)
func RegisterRoutesWithWS(r *gin.RouterGroup, cfg WSConfig) {
	// Time-sync (always present)
	timeSvc := realtimeservice.NewTimeSyncService(realtimeservice.SystemClock())
	timeHandler := realtimehandler.NewTimeSyncHandler(timeSvc)

	wsHandler := realtimehandler.NewWSHandler(cfg.CM, cfg.Metrics)
	recoveryHandler := realtimehandler.NewRecoveryHandler(cfg.RecoverySvc)

	rt := r.Group("/realtime")
	{
		rt.GET("/time", timeHandler.GetServerTime)
		rt.POST("/time-sync", timeHandler.Sync)

		// WebSocket upgrade — rate limited, JWT authenticated (in the handler).
		rt.GET("/ws",
			ws.ConnectionRateLimit(),
			wsHandler.Connect,
		)

		// Reconnection recovery — authenticated users only (CR-6 / F-014).
		rt.GET("/session/:matchId",
			middleware.Auth(),
			recoveryHandler.GetMatchState,
		)

		admin := rt.Group("/admin")
		admin.Use(middleware.Auth(), middleware.RequireRole(string(utils.RoleAdmin)))
		{
			if cfg.Gateway != nil {
				testEmit := realtimehandler.NewTestEmitHandler(cfg.Gateway)
				admin.POST("/test-emit", testEmit.Emit)
			}

			// Data retention — admin only (CR-14).
			if cfg.RetentionSvc != nil && cfg.HeartbeatSvc != nil {
				retentionHandler := realtimehandler.NewRetentionHandler(cfg.RetentionSvc, cfg.HeartbeatSvc)
				cleanup := admin.Group("/cleanup")
				{
					cleanup.POST("/scheduler-events", retentionHandler.CleanupSchedulerEvents)
					cleanup.POST("/realtime-events", retentionHandler.CleanupRealtimeEvents)
					cleanup.POST("/heartbeats", retentionHandler.CleanupHeartbeats)
					cleanup.POST("/all", retentionHandler.CleanupAll)
				}
			}
		}
	}
}
