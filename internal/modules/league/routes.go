package league

import (
	"clap/internal/modules/league/handler"
	"clap/internal/modules/league/repository"
	"clap/internal/modules/league/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	leagueRepo := repository.NewLeagueRepository(db)
	leagueService := service.NewLeagueService(leagueRepo)
	leagueHandler := handler.NewLeagueHandler(leagueService)

	leagues := r.Group("/leagues")
	{
		leagues.POST("", middleware.Auth(), leagueHandler.Create)
		leagues.GET("", leagueHandler.List)
		leagues.GET("/:id", leagueHandler.GetByID)
		leagues.PUT("/:id", middleware.Auth(), leagueHandler.Update)
		leagues.DELETE("/:id", middleware.Auth(), leagueHandler.Delete)
	}
}
