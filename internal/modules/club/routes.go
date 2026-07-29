package club

import (
	"clap/internal/modules/club/handler"
	"clap/internal/modules/club/repository"
	"clap/internal/modules/club/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	clubRepo := repository.NewClubRepository(db)
	clubService := service.NewClubService(clubRepo)
	clubHandler := handler.NewClubHandler(clubService)

	clubs := r.Group("/clubs")
	{
		clubs.POST("", middleware.Auth(), clubHandler.Create)
		clubs.GET("", clubHandler.List)
		clubs.GET("/search", clubHandler.Search)
		clubs.GET("/:id", clubHandler.GetByID)
		clubs.PUT("/:id", middleware.Auth(), clubHandler.Update)
		clubs.DELETE("/:id", middleware.Auth(), clubHandler.Delete)
	}
}
