package match

import (
	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/match/handler"
	"clap/internal/modules/match/repository"
	"clap/internal/modules/match/service"
	statsrepo "clap/internal/modules/stats/repository"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires match endpoints without realtime publishing.
func RegisterRoutes(r *gin.RouterGroup) {
	RegisterRoutesWithPublisher(r, nil)
}

// RegisterRoutesWithPublisher wires match endpoints with mobile detail
// enrichment (contract §9.1) and optional realtime score-update publishing.
func RegisterRoutesWithPublisher(r *gin.RouterGroup, publisher service.MatchEventPublisher) {
	db := database.GetDB()

	matchRepo := repository.NewMatchRepository(db)
	matchService := service.NewMatchServiceWithDetail(
		matchRepo,
		clubrepo.NewClubRepository(db),
		statsrepo.NewMatchStatsRepository(db),
		statsrepo.NewPlayerRepository(db),
		publisher,
	)
	matchHandler := handler.NewMatchHandler(matchService)

	matches := r.Group("/matches")
	{
		matches.POST("", middleware.Auth(), matchHandler.Create)
		matches.GET("", matchHandler.List)
		matches.GET("/upcoming", matchHandler.ListUpcoming)
		matches.GET("/live", matchHandler.ListLive)
		matches.GET("/:id", matchHandler.GetByID)
		matches.PUT("/:id", middleware.Auth(), matchHandler.Update)
		matches.DELETE("/:id", middleware.Auth(), matchHandler.Delete)
	}

	// Gin requires the same wildcard names as /seasons/:id, /leagues/:id, /clubs/:id.
	seasons := r.Group("/seasons/:id")
	{
		seasons.GET("/matches", matchHandler.ListBySeason)
	}

	leagues := r.Group("/leagues/:id")
	{
		leagues.GET("/matches", matchHandler.ListByLeague)
	}

	clubs := r.Group("/clubs/:id")
	{
		clubs.GET("/matches", matchHandler.ListByClub)
	}
}
