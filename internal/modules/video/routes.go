package video

import (
	userrepo "clap/internal/modules/user/repository"
	"clap/internal/modules/video/handler"
	"clap/internal/modules/video/repository"
	"clap/internal/modules/video/service"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Video endpoints (Mobile API Contract §8).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	maxVideoMB := 50
	if config.AppConfig != nil && config.AppConfig.Storage.MaxVideoFileSizeMB > 0 {
		maxVideoMB = config.AppConfig.Storage.MaxVideoFileSizeMB
	}

	videoSvc := service.NewVideoService(
		repository.NewVideoRepository(db),
		userrepo.NewProfileRepository(),
		storageinit.Provider(),
		maxVideoMB,
	)
	h := handler.NewVideoHandler(videoSvc)

	videos := r.Group("/videos")
	videos.Use(middleware.Auth())
	{
		videos.GET("/feed", h.Feed)
		videos.GET("/mine", h.Mine)
		videos.POST("", h.Upload)
		videos.POST("/:video_id/like", h.Like)
		videos.DELETE("/:video_id/like", h.Unlike)
	}
}
