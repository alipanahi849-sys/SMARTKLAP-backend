package news

import (
	"clap/internal/modules/news/handler"
	"clap/internal/modules/news/repository"
	"clap/internal/modules/news/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	h := handler.NewNewsHandler(NewService())

	newsGroup := r.Group("/news")
	newsGroup.Use(middleware.Auth())
	{
		newsGroup.GET("", h.List)
		newsGroup.GET("/:news_id", h.GetByID)
	}
}

func NewService() service.NewsService {
	return service.NewNewsService(repository.NewNewsRepository(database.GetDB()))
}
