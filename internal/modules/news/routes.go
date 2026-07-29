package news

import (
	"clap/internal/modules/news/handler"
	"clap/internal/modules/news/repository"
	"clap/internal/modules/news/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the news list endpoint referenced by the mobile Home
// (Club Mode) preview (Mobile API Contract §3.2).
func RegisterRoutes(r *gin.RouterGroup) {
	h := handler.NewNewsHandler(NewService())

	newsGroup := r.Group("/news")
	newsGroup.Use(middleware.Auth())
	{
		newsGroup.GET("", h.List)
	}
}

// NewService exposes the news service for cross-module composition (Home).
func NewService() service.NewsService {
	return service.NewNewsService(repository.NewNewsRepository(database.GetDB()))
}
