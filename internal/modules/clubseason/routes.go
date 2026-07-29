package clubseason

import (
	"clap/internal/modules/clubseason/handler"
	"clap/internal/modules/clubseason/repository"
	"clap/internal/modules/clubseason/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	clubSeasonRepo := repository.NewClubSeasonRepository(db)
	clubSeasonService := service.NewClubSeasonService(clubSeasonRepo)
	clubSeasonHandler := handler.NewClubSeasonHandler(clubSeasonService)

	clubSeasons := r.Group("/club-seasons")
	{
		clubSeasons.POST("", middleware.Auth(), clubSeasonHandler.AddClubToSeason)
		clubSeasons.PATCH("/:id", middleware.Auth(), clubSeasonHandler.UpdateStatus)
	}

	// Gin requires the same wildcard names as /seasons/:id and /clubs/:id.
	seasons := r.Group("/seasons/:id")
	{
		seasons.GET("/clubs", middleware.Auth(), clubSeasonHandler.ListClubsInSeason)
		seasons.DELETE("/clubs/:club_id", middleware.Auth(), clubSeasonHandler.RemoveClubFromSeason)
	}

	clubs := r.Group("/clubs/:id")
	{
		clubs.GET("/seasons", middleware.Auth(), clubSeasonHandler.ListSeasonsForClub)
	}
}
