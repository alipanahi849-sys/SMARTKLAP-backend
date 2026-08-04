package chant

import (
	"clap/internal/modules/chant/handler"
	"clap/internal/modules/chant/repository"
	"clap/internal/modules/chant/service"
	lyricssvc "clap/internal/modules/lyricssync/service"
	matchrepo "clap/internal/modules/match/repository"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Chants endpoints (Mobile API Contract §4).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	chantSvc := service.NewChantService(
		repository.NewChantRepository(db),
		matchrepo.NewMatchRepository(db),
		lyricssvc.NewLyricsSyncService(db),
		storageinit.Provider(),
	)
	h := handler.NewChantHandler(chantSvc)

	chants := r.Group("/chants")
	chants.Use(middleware.Auth())
	{
		chants.GET("", h.List)
		chants.GET("/:chant_id/lyrics", h.Lyrics)
		chants.POST("/:chant_id/complete", h.Complete)
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
		storageinit.Provider(),
	)
}
