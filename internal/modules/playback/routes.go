package playback

import (
	"clap/internal/modules/playback/handler"
	"clap/internal/modules/playback/repository"
	"clap/internal/modules/playback/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires song playback scheduling endpoints without realtime
// event scheduling (e.g. integration tests / environments without a Hub).
func RegisterRoutes(r *gin.RouterGroup) {
	RegisterRoutesWithEvents(r, nil)
}

// RegisterRoutesWithEvents wires song playback scheduling endpoints with an
// optional SongEventScheduler. When non-nil, scheduling a song also persists
// the playback.started and lyrics.line.changed events for realtime dispatch.
func RegisterRoutesWithEvents(r *gin.RouterGroup, eventScheduler service.SongEventScheduler) {
	db := database.GetDB()

	repo := repository.NewPlaybackRepository(db)
	songChecker := &service.GormSongChecker{DB: db}
	matchChecker := &service.GormMatchChecker{DB: db}

	var svc service.PlaybackService
	if eventScheduler != nil {
		svc = service.NewPlaybackServiceWithEvents(repo, songChecker, matchChecker, eventScheduler)
	} else {
		svc = service.NewPlaybackService(repo, songChecker, matchChecker)
	}
	h := handler.NewPlaybackHandler(svc)

	songs := r.Group("/songs")
	songs.Use(middleware.Auth())
	{
		songs.POST("/schedule", h.ScheduleSong)
		songs.DELETE("/schedule/:id", h.CancelSong)
		songs.GET("/schedule/upcoming", h.GetUpcomingSongs)
	}
}
