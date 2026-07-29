package media

import (
	"clap/internal/modules/media/handler"
	"clap/internal/modules/media/repository"
	"clap/internal/modules/media/service"
	songrepository "clap/internal/modules/song/repository"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"
	"clap/pkg/storage"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	mediaRepo := repository.NewMediaRepository(db)
	songRepo := songrepository.NewSongRepository(db)

	// Initialize storage provider from configuration
	storageProvider := initializeStorageProvider()

	cfg := config.AppConfig
	maxFileSizeMB := cfg.Storage.MaxAudioFileSizeMB
	signedURLExpirationMin := cfg.Storage.SignedURLExpirationMin

	mediaService := service.NewMediaService(mediaRepo, songRepo, storageProvider, maxFileSizeMB, signedURLExpirationMin)
	mediaHandler := handler.NewMediaHandler(mediaService)

	media := r.Group("/media")
	{
		media.POST("/upload", middleware.Auth(), mediaHandler.Upload)
		media.GET("/:id/playback-url", middleware.Auth(), mediaHandler.GetPlaybackURL)
	}

	songs := r.Group("/songs")
	{
		songs.POST("/:id/audio", middleware.Auth(), mediaHandler.UploadSongAudio)
	}
}

func initializeStorageProvider() storage.StorageProvider {
	return storageinit.Provider()
}
