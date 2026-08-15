// Package main Clap Backend API.
//
//	@title						Clap Backend API
//	@version					1.0
//	@description				Clap match-day experience backend (auth, clubs, realtime).
//	@termsOfService				http://swagger.io/terms/
//
//	@contact.name				Clap API Support
//
//	@license.name				Proprietary
//
// Host is intentionally omitted so Swagger UI calls the same origin that serves
// the docs (e.g. http://SERVER:8081 via nginx), instead of localhost:8080.
//
//	@BasePath					/
//	@schemes					http https
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT access token. Example: "Bearer {token}"
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clap/cmd/api/docs"
	"clap/internal/modules/auth"
	"clap/internal/modules/chant"
	chantrepo "clap/internal/modules/chant/repository"
	schedulerrepo "clap/internal/modules/eventscheduler/repository"
	schedulersvc "clap/internal/modules/eventscheduler/service"
	"clap/internal/modules/guess"
	lyricssvc "clap/internal/modules/lyricssync/service"
	"clap/internal/modules/match"
	matchsvc "clap/internal/modules/match/service"
	"clap/internal/modules/media"
	"clap/internal/modules/news"
	"clap/internal/modules/notification"
	notifrepo "clap/internal/modules/notification/repository"
	notifsvc "clap/internal/modules/notification/service"
	"clap/internal/modules/order"
	"clap/internal/modules/playback"
	playbackrepo "clap/internal/modules/playback/repository"
	playbacksvc "clap/internal/modules/playback/service"
	"clap/internal/modules/realtime"
	realtimegw "clap/internal/modules/realtime/gateway"
	realtimemetrics "clap/internal/modules/realtime/metrics"
	realtimerepo "clap/internal/modules/realtime/repository"
	realtimesvc "clap/internal/modules/realtime/service"
	realtimews "clap/internal/modules/realtime/ws"
	"clap/internal/modules/shop"
	"clap/internal/modules/song"
	"clap/internal/modules/songlyric"
	"clap/internal/modules/user"
	"clap/internal/modules/video"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/logger"
	"clap/internal/shared/middleware"
	"clap/internal/shared/redis"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	if err := run(); err != nil {
		logger.Fatal().Err(err).Msg("Application failed to start")
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadFromEnv(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger.Init(config.AppConfig.Environment)
	logger.Info().Msg("Starting Clap Backend API...")

	if err := database.Init(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := redis.Init(); err != nil {
		return fmt.Errorf("failed to initialize redis: %w", err)
	}

	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to run auto migrations: %w", err)
	}

	db := database.GetDB()
	rtCfg := config.AppConfig.Realtime

	// Shared durable + in-memory scheduler.
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	schedEventRepo := schedulerrepo.NewSchedulerEventRepository(db)

	// Phase 4.2: WebSocket realtime delivery layer.
	wsMetrics := realtimemetrics.New()
	hub := realtimews.NewHub(wsMetrics)

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Phase 4.1/4.2.1: recover pending + reclaim stale processing events on startup.
	staleTimeout := time.Duration(rtCfg.StaleProcessingTimeoutMinutes) * time.Minute
	schedRecovery := schedulersvc.NewSchedulerRecoveryServiceWithConfig(
		schedEventRepo, sched, staleTimeout, staleResetObserver{m: wsMetrics},
	)
	if recovered, err := schedRecovery.RecoverPendingEvents(appCtx); err != nil {
		logger.Warn().Err(err).Msg("Scheduler recovery failed — events may be missed until next sweep")
	} else {
		logger.Info().Int("recovered_events", recovered).Msg("Scheduler recovery complete")
	}

	go hub.Run(appCtx)

	cm := realtimews.NewConnectionManager(hub)
	wsGateway := realtimegw.NewWebSocketRealtimeGateway(cm, wsMetrics)

	// Start the event dispatcher that polls the scheduler and publishes events.
	dispatcher := realtimesvc.NewEventDispatcher(sched, schedEventRepo, wsGateway, 0)
	go dispatcher.Run(appCtx)

	// Lyrics + durable event scheduler powering the realtime song/chant pipeline.
	lyricsSvc := lyricssvc.NewLyricsSyncService(db)
	eventSchedulerSvc := schedulersvc.NewEventSchedulerService(schedEventRepo, sched)
	chantEventScheduler := realtimesvc.NewChantEventScheduler(
		realtimerepo.NewRealtimeSessionRepository(db),
		lyricsSvc,
		eventSchedulerSvc,
	)

	pushDevices := notifrepo.NewPushDeviceRepository(db)
	fcmSender, fcmErr := notifsvc.NewFirebaseSender(appCtx, config.AppConfig.Firebase)
	if fcmErr != nil {
		logger.Warn().Err(fcmErr).Msg("Firebase push sender unavailable; chant countdown pushes disabled")
		fcmSender = nil
	} else if fcmSender == nil {
		logger.Info().Msg("Firebase credentials not configured; chant countdown pushes disabled")
	} else {
		logger.Info().Msg("Firebase push sender ready")
	}
	pushSvc := notifsvc.NewNotificationService(pushDevices, fcmSender)

	// Notify connected users ~2 minutes before an active chant starts and
	// auto-schedule chant.started + lyric sync events. Also send one FCM
	// push so backgrounded devices see the song name and 2-minute warning.
	chantNotifier := realtimesvc.NewChantUpcomingNotifier(
		chantrepo.NewChantRepository(db),
		chant.NewService(),
		chantEventScheduler,
		cm,
		wsGateway,
		pushSvc,
		0, 0,
	)
	go chantNotifier.Run(appCtx)

	matchSyncer := match.NewSyncService()
	go matchSyncer.Run(appCtx)

	// Periodic watchdog: reclaim stale processing events and rehydrate (CR-2).
	go schedRecovery.RunWatchdog(appCtx, time.Duration(rtCfg.WatchdogIntervalMinutes)*time.Minute)

	songEventScheduler := playbacksvc.NewSongEventScheduler(
		realtimerepo.NewRealtimeSessionRepository(db),
		lyricsSvc,
		eventSchedulerSvc,
	)

	// Reconnection recovery service (cross-module read), now wired with lyrics.
	recoverySvc := realtimesvc.NewReconnectionRecoveryService(
		playbackrepo.NewPlaybackRepository(db),
		chantrepo.NewChantRepository(db),
		lyricsSvc,
	)

	// Data retention + heartbeat cleanup services (CR-14).
	retentionSvc := realtimesvc.NewDataRetentionService(
		schedEventRepo,
		realtimerepo.NewRealtimeEventRepository(db),
		realtimesvc.RetentionConfig{
			SchedulerEventRetentionDays: rtCfg.SchedulerEventRetentionDays,
			RealtimeEventRetentionDays:  rtCfg.RealtimeEventRetentionDays,
		},
	)
	heartbeatSvc := realtimesvc.NewHeartbeatCleanupService(realtimerepo.NewClientHeartbeatRepository(db))

	router := setupRouter(routerDeps{
		wsGateway:          wsGateway,
		cm:                 cm,
		recoverySvc:        recoverySvc,
		wsMetrics:          wsMetrics,
		retentionSvc:       retentionSvc,
		heartbeatSvc:       heartbeatSvc,
		songEventScheduler: songEventScheduler,
		matchSyncer:        matchSyncer,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.AppConfig.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(config.AppConfig.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.AppConfig.Server.WriteTimeout) * time.Second,
	}

	go func() {
		logger.Info().Str("port", config.AppConfig.Server.Port).Msg("Server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
	gracefulShutdown(server, appCancel, dispatcher, hub)
	logger.Info().Msg("Server exited gracefully")

	return nil
}

// gracefulShutdown tears down the application in dependency order (CR-3):
//  1. Stop accepting new requests (HTTP server shutdown)
//  2. Cancel the application context (stops dispatcher, Hub, watchdog)
//  3. Wait for the dispatcher to drain
//  4. Wait for the Hub to close all connections
//  5. Close the database
//  6. Close Redis
//
// No database/Redis operation runs after their respective Close calls.
func gracefulShutdown(
	server *http.Server,
	appCancel context.CancelFunc,
	dispatcher *realtimesvc.EventDispatcher,
	hub *realtimews.Hub,
) {
	drain := time.Duration(config.AppConfig.Realtime.ShutdownDrainTimeoutSeconds) * time.Second
	if drain <= 0 {
		drain = 15 * time.Second
	}

	// 1. Stop accepting new HTTP/WebSocket requests.
	httpCtx, cancelHTTP := context.WithTimeout(context.Background(), drain)
	defer cancelHTTP()
	if err := server.Shutdown(httpCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	// 2. Cancel the application context to stop background goroutines.
	appCancel()

	// 3. Wait for the dispatcher to finish in-flight work.
	select {
	case <-dispatcher.Done():
		logger.Info().Msg("event dispatcher stopped")
	case <-time.After(drain):
		logger.Warn().Msg("event dispatcher did not stop within drain timeout")
	}

	// 4. Wait for the Hub to close all client connections.
	select {
	case <-hub.Done():
		logger.Info().Msg("websocket hub stopped")
	case <-time.After(drain):
		logger.Warn().Msg("websocket hub did not stop within drain timeout")
	}

	// 5. Close the database (after all DB users have stopped).
	if err := database.Close(); err != nil {
		logger.Error().Err(err).Msg("Error closing database connection")
	}

	// 6. Close Redis last.
	if err := redis.Close(); err != nil {
		logger.Error().Err(err).Msg("Error closing redis connection")
	}
}

// staleResetObserver bridges scheduler recovery to the realtime metrics.
type staleResetObserver struct {
	m *realtimemetrics.Metrics
}

func (o staleResetObserver) ObserveStaleReset(count int64) {
	if o.m != nil && count > 0 {
		o.m.StaleProcessingRecovered.Add(count)
	}
}

// routerDeps bundles the runtime dependencies wired into the HTTP router.
type routerDeps struct {
	wsGateway          *realtimegw.WebSocketRealtimeGateway
	cm                 *realtimews.ConnectionManager
	recoverySvc        realtimesvc.ReconnectionRecoveryService
	wsMetrics          *realtimemetrics.Metrics
	retentionSvc       realtimesvc.DataRetentionService
	heartbeatSvc       realtimesvc.HeartbeatCleanupService
	songEventScheduler playbacksvc.SongEventScheduler
	matchSyncer        *matchsvc.SyncService
}

func setupRouter(deps routerDeps) *gin.Engine {
	if config.AppConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Timeout())
	router.Use(middleware.Recovery())
	router.Use(middleware.Logging())
	router.Use(middleware.CORS())

	router.GET("/health", healthCheck)

	// Disk-backed uploads are served from /uploads (local provider, or R2 fallback).
	if config.AppConfig != nil {
		root := config.AppConfig.Storage.LocalRoot
		if root == "" {
			root = "./uploads"
		}
		uploads := router.Group("/uploads")
		uploads.Use(func(c *gin.Context) {
			// Avatars are overwritten rarely but browsers cache aggressively by URL;
			// short cache + must-revalidate avoids stale images if a URL is reused.
			c.Header("Cache-Control", "public, max-age=60, must-revalidate")
			c.Next()
		})
		uploads.StaticFS("/", gin.Dir(root, false))
	}

	// Swagger UI (disabled in production). Empty Host = same origin as the page
	// (important behind nginx / remote servers; avoids localhost:8080 in Try it out).
	// /swagger and /swagger/ otherwise 404 under /*any — redirect to the UI entrypoint.
	if config.AppConfig.Environment != "production" {
		docs.SwaggerInfo.Host = ""
		docs.SwaggerInfo.BasePath = "/"
		docs.SwaggerInfo.Schemes = []string{"http", "https"}
		swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.Handler)
		router.GET("/swagger/*any", func(c *gin.Context) {
			if any := c.Param("any"); any == "" || any == "/" {
				c.Redirect(http.StatusFound, "/swagger/index.html")
				return
			}
			swaggerHandler(c)
		})
	}

	v1 := router.Group("/api/v1")
	{
		auth.RegisterRoutes(v1)
		user.RegisterRoutes(v1)
		song.RegisterRoutes(v1)
		songlyric.RegisterRoutes(v1)
		media.RegisterRoutes(v1)
		match.RegisterRoutes(v1, deps.matchSyncer)
		news.RegisterRoutes(v1)
		// Phase 4.3: Mobile API Contract modules
		chant.RegisterRoutes(v1)
		guess.RegisterRoutes(v1)
		video.RegisterRoutes(v1)
		shop.RegisterRoutes(v1)
		order.RegisterRoutes(v1)
		notification.RegisterRoutes(v1)
		// Phase 4 + 4.2: Realtime Engine Foundation + WebSocket Delivery Layer
		realtime.RegisterRoutesWithWS(v1, realtime.WSConfig{
			CM:           deps.cm,
			Gateway:      deps.wsGateway,
			RecoverySvc:  deps.recoverySvc,
			Metrics:      deps.wsMetrics,
			RetentionSvc: deps.retentionSvc,
			HeartbeatSvc: deps.heartbeatSvc,
		})
		playback.RegisterRoutesWithEvents(v1, deps.songEventScheduler)
	}

	return router
}

// healthCheck godoc
//
//	@Summary		Health check
//	@Description	Returns API, database, and Redis health status
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		503	{object}	map[string]interface{}
//	@Router			/health [get]
func healthCheck(c *gin.Context) {
	status := "healthy"
	statusCode := http.StatusOK

	if err := database.HealthCheck(); err != nil {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	if err := redis.HealthCheck(); err != nil {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC(),
	})
}

func autoMigrate() error {
	if config.AppConfig.Environment == "development" {
		logger.Info().Msg("Running auto migrations...")

		// Import models to ensure they're registered with GORM
		// This is a simple approach - in production, use proper migration files
		// For now, we'll skip auto-migration and rely on SQL migration files

		logger.Info().Msg("Auto migrations skipped - use SQL migration files instead")
	}

	return nil
}
