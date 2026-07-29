package matchsongschedule

import (
	"clap/internal/modules/matchsongschedule/handler"
	"clap/internal/modules/matchsongschedule/repository"
	"clap/internal/modules/matchsongschedule/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	scheduleRepo := repository.NewMatchSongScheduleRepository(db)
	scheduleService := service.NewMatchSongScheduleService(scheduleRepo)
	scheduleHandler := handler.NewMatchSongScheduleHandler(scheduleService)

	schedules := r.Group("/match-song-schedules")
	{
		schedules.POST("", middleware.Auth(), scheduleHandler.Create)
		schedules.GET("", scheduleHandler.List)
		schedules.GET("/:id", scheduleHandler.GetByID)
		schedules.PUT("/:id", middleware.Auth(), scheduleHandler.Update)
		schedules.DELETE("/:id", middleware.Auth(), scheduleHandler.Delete)
	}

	// Gin requires the same wildcard names as /matches/:id and /songs/:id.
	matches := r.Group("/matches/:id")
	{
		matches.GET("/song-schedules", scheduleHandler.ListByMatchID)
	}

	songs := r.Group("/songs/:id")
	{
		songs.GET("/match-schedules", scheduleHandler.ListBySongID)
	}
}
