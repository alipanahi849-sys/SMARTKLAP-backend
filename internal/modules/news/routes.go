package news

import (
	clubrepo "clap/internal/modules/club/repository"
	matchsvc "clap/internal/modules/match/service"
	"clap/internal/modules/news/handler"
	"clap/internal/modules/news/service"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/utils"
	"clap/pkg/newsfeed"

	"github.com/gin-gonic/gin"
)

func newProvider() newsfeed.Provider {
	apiKey := ""
	baseURL := ""
	if config.AppConfig != nil {
		apiKey = config.AppConfig.News.APIKey
		baseURL = config.AppConfig.News.BaseURL
	}
	return newsfeed.NewGuardian(apiKey, baseURL)
}

// RegisterRoutes wires mobile news endpoints and the admin news-club setting.
func RegisterRoutes(r *gin.RouterGroup, clubsFrom *matchsvc.SyncService) {
	db := database.GetDB()
	svc := service.NewNewsService(
		newProvider(),
		settingsrepo.NewSettingsRepository(db),
		clubrepo.NewClubRepository(db),
		clubsFrom,
	)
	h := handler.NewNewsHandler(svc)

	newsGroup := r.Group("/news")
	newsGroup.Use(middleware.Auth())
	{
		newsGroup.GET("", h.List)
		newsGroup.GET("/:news_id", h.GetByID)
	}

	admin := r.Group("/admin")
	admin.Use(middleware.Auth(), middleware.RequireRole(string(utils.RoleAdmin)))
	{
		admin.GET("/settings/news-club", h.GetNewsClub)
		admin.PUT("/settings/news-club", h.SetNewsClub)
	}
}
