package season

import (
	"clap/internal/modules/season/handler"
	"clap/internal/modules/season/repository"
	"clap/internal/modules/season/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	seasonRepo := repository.NewSeasonRepository(db)
	seasonService := service.NewSeasonService(seasonRepo)
	seasonHandler := handler.NewSeasonHandler(seasonService)

	seasons := r.Group("/seasons")
	{
		seasons.POST("", middleware.Auth(), seasonHandler.Create)
		seasons.GET("", seasonHandler.List)
		seasons.GET("/:id", seasonHandler.GetByID)
		seasons.PUT("/:id", middleware.Auth(), seasonHandler.Update)
		seasons.DELETE("/:id", middleware.Auth(), seasonHandler.Delete)
	}

	// Gin requires the same wildcard name as /leagues/:id (registered by league).
	leagues := r.Group("/leagues/:id")
	{
		leagues.GET("/seasons", seasonHandler.ListByLeagueID)
	}
}
