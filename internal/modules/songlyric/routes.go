package songlyric

import (
	"clap/internal/modules/songlyric/handler"
	"clap/internal/modules/songlyric/repository"
	"clap/internal/modules/songlyric/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	lyricRepo := repository.NewSongLyricRepository(db)
	lyricService := service.NewSongLyricService(lyricRepo)
	lyricHandler := handler.NewSongLyricHandler(lyricService)

	lyrics := r.Group("/song-lyrics")
	{
		lyrics.POST("", middleware.AdminAuth(), lyricHandler.Create)
		lyrics.GET("/:id", lyricHandler.GetByID)
		lyrics.PUT("/:id", middleware.AdminAuth(), lyricHandler.Update)
		lyrics.DELETE("/:id", middleware.AdminAuth(), lyricHandler.Delete)
	}

	// Gin requires the same wildcard name as /songs/:id.
	songs := r.Group("/songs/:id")
	{
		songs.GET("/lyrics", lyricHandler.ListBySongID)
		songs.GET("/lyrics/:language", lyricHandler.GetBySongID)
		songs.POST("/lyrics/import", middleware.AdminAuth(), lyricHandler.ImportLyrics)
	}
}
