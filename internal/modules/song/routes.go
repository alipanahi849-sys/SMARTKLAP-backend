package song

import (
	"clap/internal/modules/song/handler"
	"clap/internal/modules/song/repository"
	"clap/internal/modules/song/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	songRepo := repository.NewSongRepository(db)
	songService := service.NewSongService(songRepo)
	songHandler := handler.NewSongHandler(songService)

	songs := r.Group("/songs")
	{
		songs.POST("", middleware.Auth(), songHandler.Create)
		songs.GET("", songHandler.List)
		songs.GET("/search", songHandler.Search)
		songs.GET("/:id", songHandler.GetByID)
		songs.PUT("/:id", middleware.Auth(), songHandler.Update)
		songs.DELETE("/:id", middleware.Auth(), songHandler.Delete)
	}
}
