package stats

import (
	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/stats/handler"
	"clap/internal/modules/stats/repository"
	"clap/internal/modules/stats/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the Players endpoint (Mobile API Contract §9.2).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	playerSvc := service.NewPlayerService(
		repository.NewPlayerRepository(db),
		clubrepo.NewClubRepository(db),
	)
	h := handler.NewPlayerHandler(playerSvc)

	players := r.Group("/players")
	players.Use(middleware.Auth())
	{
		players.GET("/:player_id", h.PlayerDetail)
	}
}
