package mobilehome

import (
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/chant"
	clubrepo "clap/internal/modules/club/repository"
	matchrepo "clap/internal/modules/match/repository"
	"clap/internal/modules/mobilehome/handler"
	"clap/internal/modules/mobilehome/service"
	"clap/internal/modules/news"
	"clap/internal/modules/shop"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Home aggregates (Mobile API Contract §3).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	homeSvc := service.NewHomeService(
		authrepo.NewUserRepository(),
		matchrepo.NewMatchRepository(db),
		clubrepo.NewClubRepository(db),
		chant.NewService(),
		shop.NewService(),
		news.NewService(),
	)
	h := handler.NewHomeHandler(homeSvc)

	home := r.Group("/mobile/home")
	home.Use(middleware.Auth())
	{
		home.GET("/stadium", h.Stadium)
		home.GET("/club", h.Club)
	}
}
