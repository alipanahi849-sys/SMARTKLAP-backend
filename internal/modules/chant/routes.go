package chant

import (
	"clap/internal/modules/chant/handler"
	"clap/internal/modules/chant/repository"
	"clap/internal/modules/chant/service"
	lyricssvc "clap/internal/modules/lyricssync/service"
	matchrepo "clap/internal/modules/match/repository"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Chants endpoints (Mobile API Contract §4).
func RegisterRoutes(r *gin.RouterGroup) {
	h := handler.NewChantHandler(NewService())

	chants := r.Group("/chants")
	chants.Use(middleware.Auth())
	{
		chants.GET("", h.List)
		chants.GET("/me/today", h.TodayStats)
		chants.GET("/program", h.Program)
		chants.GET("/:chant_id/lyrics", h.Lyrics)
		chants.POST("/:chant_id/complete", h.Complete)
		chants.POST("/:chant_id/cancel", h.Cancel)
	}

	admin := r.Group("/admin")
	admin.Use(middleware.Auth(), middleware.RequireRole(string(utils.RoleAdmin)))
	{
		admin.GET("/settings/chant-points", h.GetPoints)
		admin.PUT("/settings/chant-points", h.UpdatePoints)
		admin.GET("/chants", h.ListOnlineChants)
		admin.POST("/chants", h.SetOnlineChant)
		admin.DELETE("/chants/:chant_id", h.UnsetOnlineChant)
	}
}

// NewService exposes a fully wired ChantService for cross-module composition
// (the mobile Home aggregate reuses it).
func NewService() service.ChantService {
	db := database.GetDB()
	return service.NewChantService(
		repository.NewChantRepository(db),
		matchrepo.NewMatchRepository(db),
		lyricssvc.NewLyricsSyncService(db),
		settingsrepo.NewSettingsRepository(db),
		storageinit.Provider(),
	)
}
